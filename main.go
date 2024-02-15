package main

import (
	"binance_robot/config"
	"binance_robot/criar_ordem"
	"binance_robot/database"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"binance_robot/util"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {

	database.DBCon()

	config.ReadFile()

	var (
		currentCoin    string
		side           string
		value          float64
		margemInferior float64
		margemSuperior float64
		alavancagem    float64
		//roi               float64
		err               error
		currentValue      float64
		ordemAtiva        bool
		valueCompradoCoin float64
		primeiraExec      bool
		//entryPrice        string
		roiAcumulado float64
		stop         float64
		allOrders    []models.CryptoPosition
	)

	for {
		fmt.Print("Digite a moeda (ex: BTC): ")
		_, err = fmt.Scanln(&currentCoin)
		if err != nil {
			fmt.Println("Erro, tente digitar somente letras: ", err)
			continue
		}
		currentCoin = strings.ToUpper(currentCoin)
		if len(currentCoin) > 0 {
			break
		} else {
			fmt.Println("Por favor, insira uma moeda válida.")
		}
	} // currentCoin
	for {
		fmt.Print("Digite a quantidade em " + config.BaseCoin + ": ")
		_, err = fmt.Scanln(&value)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if value > 0 {
			break
		} else {
			fmt.Println("Por favor, digite um valor válido.")
		}
	} // value
	for {
		fmt.Println("Irá trabalhar em LONG ou SHORT? (ex: BUY, SELL)")
		_, err = fmt.Scanln(&side)
		if err != nil {
			fmt.Println("Erro, tente digitar somente letras: ", err)
			continue
		}
		side = strings.ToUpper(side)
		if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" {
			break
		} else {
			fmt.Println("Deve entrar somente em LONG ou SHORT")
			continue
		}
	} // side
	for {
		fmt.Println("Qual sua alavancagem (1 - 20): ")
		_, err = fmt.Scanln(&alavancagem)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if alavancagem > 20 {
			alavancagem = 20
			fmt.Println("Alavancagem maior que 20, definido como 20.")
			break
		} else if alavancagem <= 0 {
			alavancagem = 1
			fmt.Println("Alavancagem menor que 0 definido como 1.")
			break
		}
		break
	} // alavancagem
	/*for {
		fmt.Println("Qual o ROI que deseja trabalhar (ex: 1.5): ")
		_, err = fmt.Scanln(&roi)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if roi > 0 {
			break
		} else {
			fmt.Println("ROI precisa ser maior que 0")
			continue
		}
	} // roi*/
	for {
		fmt.Println("Qual o Stop Loss que deseja trabalhar em porcentagem (ex: 0.5): ")
		_, err = fmt.Scanln(&stop)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if stop > 0 {
			break
		} else {
			fmt.Println("Stop Loss precisa ser maior que 0")
			continue
		}
	} // stopLoss
	for {
		fmt.Println("Qual sua margem inferior: ")
		_, err = fmt.Scanln(&margemInferior)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if margemInferior < 0 {
			fmt.Println("Margem inferior precisa ser maior que 0")
			continue
		}

		fmt.Println("Qual sua margem superior: ")
		_, err = fmt.Scanln(&margemSuperior)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if margemSuperior < 0 {
			fmt.Println("Margem superior precisa ser maior que 0")
			continue
		}
		if margemSuperior > margemInferior {
			break
		} else {
			fmt.Println("Margem Superior precisa ser maior que a Margem Inferior.")
			continue
		}
	} // margens

	fmt.Println("Para parar as transações pressione Ctrl + C")

	ordemAtiva = false
	primeiraExec = true
	valueCompradoCoin = 0.0
	roiAcumulado = 0.0
	fee := 0.05

	now := time.Now()
	timestamp := now.UnixMilli()
	apiParams := "symbol=" + currentCoin + config.BaseCoin + "&leverage=" + fmt.Sprint(alavancagem) + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)
	url := config.BaseURL + "fapi/v1/leverage?" + apiParams + "&signature=" + signature

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	fmt.Println(string(body))

	// Encerrar a aplicação graciosamente
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("Sinal capturado: %v\n", sig)

		err := criar_ordem.RemoverCoinDB(currentCoin)
		if err != nil {
			fmt.Println("Erro ao remover a moeda do banco de dados:", err)
		}

		os.Exit(0)
	}()

	for {
		var ultimos []models.PriceResponse
		criar_ordem.EnviarCoinDB(currentCoin)

		if primeiraExec {
			time.Sleep(15 * time.Second)
			primeiraExec = false
		}
		ultimos = listar_ordens.ListarUltimosValores(currentCoin, 5)
		currentPrice, _ := strconv.ParseFloat(ultimos[0].Price, 64)
		if !ordemAtiva {
			if side == "BUY" {
				if ultimos[0].Price >= ultimos[1].Price && ultimos[1].Price >= ultimos[2].Price && ultimos[2].Price >= ultimos[3].Price /*&& ultimos[3].Price >= ultimos[4].Price*/ &&
					ultimos[0].Price > fmt.Sprint(margemInferior) && fmt.Sprint(margemSuperior) > ultimos[0].Price &&
					ultimos[0].Price > ultimos[4].Price {
					currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
					valueCompradoCoin = currentPrice
					util.Write("Entrada em LONG: "+ultimos[0].Price, currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao criar conta: ", err)
					}
					ordemAtiva = true
					allOrders, err = listar_ordens.ListarOrdens(currentCoin)
					if err != nil {
						log.Println("Erro ao listar ordens: ", err)
					}
					for _, item := range allOrders {
						if item.PositionSide == side {
							valueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
							if err != nil {
								log.Println("Erro ao buscar valor de entrada: ", err)
							}
						}
					}
				}

			} else if side == "SELL" {
				if ultimos[0].Price <= ultimos[1].Price && ultimos[1].Price <= ultimos[2].Price && ultimos[2].Price <= ultimos[3].Price /*&&*ultimos[3].Price <= ultimos[4].Price*/ &&
					fmt.Sprint(margemInferior) > ultimos[0].Price && ultimos[0].Price > fmt.Sprint(margemSuperior) &&
					ultimos[0].Price < ultimos[4].Price {

					currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
					valueCompradoCoin = currentPrice
					util.Write("Entrada em SHORT: "+ultimos[0].Price, currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao criar conta: ", err)
					}
					ordemAtiva = true
					allOrders, err = listar_ordens.ListarOrdens(currentCoin)
					if err != nil {
						log.Println("Erro ao listar ordens: ", err)
					}
					for _, item := range allOrders {
						if item.PositionSide == side {
							valueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
							if err != nil {
								log.Println("Erro ao buscar valor de entrada: ", err)
							}
						}
					}

				}
			}
			if currentPrice > margemSuperior || currentPrice < margemInferior {
				fmt.Println("Atenção uma das margens foi atingida.: Margem Inferior: " + fmt.Sprint(margemInferior) + "- Margem Superior: " + fmt.Sprint(margemSuperior) + " - Preço atul: " + fmt.Sprint(currentPrice))
				fmt.Println("\nDefina novos parametros.")
				for {
					fmt.Println("Irá trabalhar em LONG ou SHORT? (ex: BUY, SELL)")
					_, err = fmt.Scanln(&side)
					if err != nil {
						fmt.Println("Erro, tente digitar somente letras: ", err)
						continue
					}
					side = strings.ToUpper(side)
					if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" {
						break
					} else {
						fmt.Println("Deve entrar somente em LONG ou SHORT")
						continue
					}
				} // side
				for {
					fmt.Println("Qual sua margem inferior: ")
					_, err = fmt.Scanln(&margemInferior)
					if err != nil {
						fmt.Println("Erro, tente digitar somente números: ", err)
						continue
					}
					if margemInferior < 0 {
						fmt.Println("Margem inferior precisa ser maior que 0")
						continue
					}

					fmt.Println("Qual sua margem superior: ")
					_, err = fmt.Scanln(&margemSuperior)
					if err != nil {
						fmt.Println("Erro, tente digitar somente números: ", err)
						continue
					}
					if margemSuperior < 0 {
						fmt.Println("Margem superior precisa ser maior que 0")
						continue
					}
					if margemSuperior > margemInferior {
						break
					} else {
						fmt.Println("Margem Superior precisa ser maior que a Margem Inferior.")
						continue
					}
				} // margens

			}
		} else {
			if side == "BUY" {
				ROI := (((currentPrice - valueCompradoCoin) / (valueCompradoCoin / alavancagem)) * 100) - (fee * 2)
				now = time.Now()
				timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
				formattedTime := timeValue.Format("2006-01-02 15:04:05")

				util.Write("Valor de entrada: "+fmt.Sprint(valueCompradoCoin)+" - "+fmt.Sprintf("%.4f", ROI)+" - "+formattedTime+" - "+fmt.Sprint(currentPrice), currentCoin+config.BaseCoin)

				if (ultimos[0].Price < ultimos[1].Price && ultimos[1].Price < ultimos[2].Price) && ROI > (fee*2) {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - desceu 2 consecutivos após atingir o ROI. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"\n\n", currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					ordemAtiva = false

				} else if ultimos[0].Price >= fmt.Sprint(margemSuperior) {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - Atingiu a Margem Superior: "+ultimos[0].Price+" Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"\n\n", currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					fmt.Println("\nDefina novos parametros.")
					for {
						fmt.Println("Irá trabalhar em LONG ou SHORT? (ex: BUY, SELL)")
						_, err = fmt.Scanln(&side)
						if err != nil {
							fmt.Println("Erro, tente digitar somente letras: ", err)
							continue
						}
						side = strings.ToUpper(side)
						if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" {
							break
						} else {
							fmt.Println("Deve entrar somente em LONG ou SHORT")
							continue
						}
					} // side
					for {
						fmt.Println("Qual sua margem inferior: ")
						_, err = fmt.Scanln(&margemInferior)
						if err != nil {
							fmt.Println("Erro, tente digitar somente números: ", err)
							continue
						}
						if margemInferior < 0 {
							fmt.Println("Margem inferior precisa ser maior que 0")
							continue
						}

						fmt.Println("Qual sua margem superior: ")
						_, err = fmt.Scanln(&margemSuperior)
						if err != nil {
							fmt.Println("Erro, tente digitar somente números: ", err)
							continue
						}
						if margemSuperior < 0 {
							fmt.Println("Margem superior precisa ser maior que 0")
							continue
						}
						if margemSuperior > margemInferior {
							break
						} else {
							fmt.Println("Margem Superior precisa ser maior que a Margem Inferior.")
							continue
						}
					} // margens
					ordemAtiva = false
				} else if ultimos[0].Price <= fmt.Sprint(margemInferior) {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - atingiu a margem inferior: "+ultimos[0].Price+" Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"\n\n", currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					fmt.Println("\nDefina novos parametros.")
					for {
						fmt.Println("Irá trabalhar em LONG ou SHORT? (ex: BUY, SELL)")
						_, err = fmt.Scanln(&side)
						if err != nil {
							fmt.Println("Erro, tente digitar somente letras: ", err)
							continue
						}
						side = strings.ToUpper(side)
						if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" {
							break
						} else {
							fmt.Println("Deve entrar somente em LONG ou SHORT")
							continue
						}
					} // side
					for {
						fmt.Println("Qual sua margem inferior: ")
						_, err = fmt.Scanln(&margemInferior)
						if err != nil {
							fmt.Println("Erro, tente digitar somente números: ", err)
							continue
						}
						if margemInferior < 0 {
							fmt.Println("Margem inferior precisa ser maior que 0")
							continue
						}

						fmt.Println("Qual sua margem superior: ")
						_, err = fmt.Scanln(&margemSuperior)
						if err != nil {
							fmt.Println("Erro, tente digitar somente números: ", err)
							continue
						}
						if margemSuperior < 0 {
							fmt.Println("Margem superior precisa ser maior que 0")
							continue
						}
						if margemSuperior > margemInferior {
							break
						} else {
							fmt.Println("Margem Superior precisa ser maior que a Margem Inferior.")
							continue
						}
					} // margens
					ordemAtiva = false
				} else if ROI <= 0-(stop) {
					roiAcumulado = roiAcumulado + ROI
					util.Write("StopLoss atingido: "+ultimos[0].Price+" Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"\n\n", currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					ordemAtiva = false
				}
			} else if side == "SELL" {
				ROI := (((valueCompradoCoin - currentPrice) / (valueCompradoCoin / alavancagem)) * 100) - (fee * 2)
				now = time.Now()
				timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
				formattedTime := timeValue.Format("2006-01-02 15:04:05")

				util.Write("Valor de entrada: "+fmt.Sprint(valueCompradoCoin)+" - "+fmt.Sprintf("%.4f", ROI)+" - "+formattedTime+" - "+fmt.Sprint(currentPrice), currentCoin+config.BaseCoin)

				if ultimos[0].Price > ultimos[1].Price && ultimos[1].Price > ultimos[2].Price && ROI >= (fee*2)*2 {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - subiu 2 consecutivos após atingir o ROI: "+ultimos[0].Price+" Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"\n\n", currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					ordemAtiva = false

				} else if ultimos[0].Price >= fmt.Sprint(margemSuperior) {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - atingiu a margem superior: "+ultimos[0].Price+" Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"\n\n", currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					fmt.Println("\nDefina novos parametros.")
					for {
						fmt.Println("Irá trabalhar em LONG ou SHORT? (ex: BUY, SELL)")
						_, err = fmt.Scanln(&side)
						if err != nil {
							fmt.Println("Erro, tente digitar somente letras: ", err)
							continue
						}
						side = strings.ToUpper(side)
						if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" {
							break
						} else {
							fmt.Println("Deve entrar somente em LONG ou SHORT")
							continue
						}
					} // side
					for {
						fmt.Println("Qual sua margem inferior: ")
						_, err = fmt.Scanln(&margemInferior)
						if err != nil {
							fmt.Println("Erro, tente digitar somente números: ", err)
							continue
						}
						if margemInferior < 0 {
							fmt.Println("Margem inferior precisa ser maior que 0")
							continue
						}

						fmt.Println("Qual sua margem superior: ")
						_, err = fmt.Scanln(&margemSuperior)
						if err != nil {
							fmt.Println("Erro, tente digitar somente números: ", err)
							continue
						}
						if margemSuperior < 0 {
							fmt.Println("Margem superior precisa ser maior que 0")
							continue
						}
						if margemSuperior > margemInferior {
							break
						} else {
							fmt.Println("Margem Superior precisa ser maior que a Margem Inferior.")
							continue
						}
					} // margens
					ordemAtiva = false
				} else if ultimos[0].Price <= fmt.Sprint(margemInferior) {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - atingiu a margem inferior: "+ultimos[0].Price+"Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"\n\n", currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					fmt.Println("\nDefina novos parametros.")
					for {
						fmt.Println("Irá trabalhar em LONG ou SHORT? (ex: BUY, SELL)")
						_, err = fmt.Scanln(&side)
						if err != nil {
							fmt.Println("Erro, tente digitar somente letras: ", err)
							continue
						}
						side = strings.ToUpper(side)
						if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" {
							break
						} else {
							fmt.Println("Deve entrar somente em LONG ou SHORT")
							continue
						}
					} // side
					for {
						fmt.Println("Qual sua margem inferior: ")
						_, err = fmt.Scanln(&margemInferior)
						if err != nil {
							fmt.Println("Erro, tente digitar somente números: ", err)
							continue
						}
						if margemInferior < 0 {
							fmt.Println("Margem inferior precisa ser maior que 0")
							continue
						}

						fmt.Println("Qual sua margem superior: ")
						_, err = fmt.Scanln(&margemSuperior)
						if err != nil {
							fmt.Println("Erro, tente digitar somente números: ", err)
							continue
						}
						if margemSuperior < 0 {
							fmt.Println("Margem superior precisa ser maior que 0")
							continue
						}
						if margemSuperior > margemInferior {
							break
						} else {
							fmt.Println("Margem Superior precisa ser maior que a Margem Inferior.")
							continue
						}
					} // margens
					ordemAtiva = false
				} else if ROI <= 0-(stop) {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - StopLoss atingido. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"\n\n", currentCoin+config.BaseCoin)
					err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), ultimos[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					ordemAtiva = false
				}
			}
		}
		time.Sleep(900 * time.Millisecond)
	}

}
