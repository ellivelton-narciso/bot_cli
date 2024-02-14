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
		roi            float64
		err            error
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
	for {
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
	} // roi
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
	var currentValue float64

	var ordemAtiva bool
	var valueComprado float64
	var valueCompradoCoin float64
	var primeiraExec bool
	var entryPrice string
	ordemAtiva = false
	primeiraExec = true
	valueComprado = 0.0
	valueCompradoCoin = 0.0

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

		if !ordemAtiva {
			if side == "BUY" {
				if ultimos[0].Price > ultimos[1].Price && ultimos[1].Price > ultimos[2].Price && ultimos[2].Price > ultimos[3].Price /*&& ultimos[3].Price > ultimos[4].Price*/ {
					currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
					valueComprado = currentValue
					valueCompradoCoin, _ = strconv.ParseFloat(ultimos[0].Price, 64)
					fmt.Println("Entrada em LONG: " + ultimos[0].Price)
					entryPrice, err = criar_ordem.CriarOrdem(currentCoin, side, "LONG", fmt.Sprint(currentValue))
					if err != nil {
						fmt.Println(err)
					}
					ordemAtiva = true
					fmt.Println(entryPrice)
				}

			} else if side == "SELL" {
				if ultimos[0].Price < ultimos[1].Price && ultimos[1].Price < ultimos[2].Price && ultimos[2].Price < ultimos[3].Price /*&& ultimos[3].Price < ultimos[4].Price*/ {
					currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
					valueComprado = currentValue
					valueCompradoCoin, _ = strconv.ParseFloat(ultimos[0].Price, 64)
					fmt.Println("Entrada em SHORT: " + ultimos[0].Price)
					entryPrice, err = criar_ordem.CriarOrdem(currentCoin, side, "SHORT", fmt.Sprint(currentValue))
					if err != nil {
						fmt.Println(err)
					}
					ordemAtiva = true
					fmt.Println(entryPrice)

				}
			}
		} else {
			if side == "BUY" {
				currentPrice, _ := strconv.ParseFloat(ultimos[0].Price, 64)
				ROI := ((currentPrice - valueCompradoCoin) / (valueCompradoCoin / alavancagem)) * 100 * alavancagem
				now = time.Now()
				timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
				formattedTime := timeValue.Format("2006-01-02 15:04:05")

				fmt.Println(fmt.Sprintf("%.4f", ROI) + " - " + formattedTime + " - " + fmt.Sprint(currentPrice))

				if ultimos[0].Price < ultimos[1].Price && ultimos[1].Price < ultimos[2].Price {
					fmt.Println("Ordem encerrada - desceu 2 consecutivos: " + ultimos[0].Price + "\n\n")
					_, err = criar_ordem.CriarOrdem(currentCoin, "SELL", "LONG", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false

				} else if ultimos[0].Price >= fmt.Sprint(margemSuperior) {
					fmt.Println("Ordem encerrada - Atingiu a Margem Superior: " + ultimos[0].Price + "\n\n")
					_, err = criar_ordem.CriarOrdem(currentCoin, "SELL", "LONG", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				} else if ultimos[0].Price <= fmt.Sprint(margemInferior) {
					fmt.Println("Ordem encerrada - atingiu a margem inferior: " + ultimos[0].Price + "\n\n")
					_, err = criar_ordem.CriarOrdem(currentCoin, "SELL", "LONG", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				} else if ROI >= roi {
					fmt.Println(ROI)
					fmt.Println("Ordem encerrada - ROI atingido: " + ultimos[0].Price + "\n\n")
					_, err = criar_ordem.CriarOrdem(currentCoin, "SELL", "LONG", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				}
			} else if side == "SELL" {
				currentPrice, _ := strconv.ParseFloat(ultimos[0].Price, 64)
				ROI := ((valueCompradoCoin - currentPrice) / (valueCompradoCoin / alavancagem)) * 100 * alavancagem
				now = time.Now()
				timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
				formattedTime := timeValue.Format("2006-01-02 15:04:05")

				fmt.Println(fmt.Sprintf("%.4f", ROI) + " - " + formattedTime + " - " + fmt.Sprint(currentPrice))

				if ultimos[0].Price > ultimos[1].Price && ultimos[1].Price > ultimos[2].Price {
					fmt.Println("Ordem encerrada - subiu 2 consecutivos: " + ultimos[0].Price + "\n\n")
					_, err = criar_ordem.CriarOrdem(currentCoin, "BUY", "SHORT", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false

				} else if ultimos[0].Price >= fmt.Sprint(margemSuperior) {
					fmt.Println("Ordem encerrada - atingiu a margem superior: " + ultimos[0].Price + "\n\n")
					_, err = criar_ordem.CriarOrdem(currentCoin, "BUY", "SHORT", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				} else if ultimos[0].Price <= fmt.Sprint(margemInferior) {
					fmt.Println("Ordem encerrada - atingiu a margem inferior: " + ultimos[0].Price + "\n\n")
					_, err = criar_ordem.CriarOrdem(currentCoin, "BUY", "SHORT", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				} else if ROI >= roi {
					fmt.Println("Ordem encerrada - atingiu roi: " + ultimos[0].Price + "\n\n")
					_, err = criar_ordem.CriarOrdem(currentCoin, "BUY", "SHORT", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				}
			}
		}
		time.Sleep(1 * time.Second)
	}

}
