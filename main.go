package main

import (
	"binance_robot/config"
	"binance_robot/criar_ordem"
	"binance_robot/database"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"binance_robot/util"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	currentCoin    string
	side           string
	value          float64
	margemInferior float64
	margemSuperior float64
	alavancagem    float64
	currentPrice   float64
	//roi               float64
	err               error
	currentValue      float64
	currentPriceStr   string
	ordemAtiva        bool
	valueCompradoCoin float64
	primeiraExec      bool
	roiAcumulado      float64
	stop              float64
	allOrders         []models.CryptoPosition
	ultimosEntrada    []models.PriceResponse
	ultimosSaida      []models.PriceResponse
	ultimosVelas      []models.PriceResponse
	entrada           string
	velasqtd          int64
	velasperiodo      int64
	segEntrada        int64
	segSaida          int64
	entrarBuy         bool
	entrarSell        bool
	sairBuy           bool
	sairSell          bool
	slAtingido        bool
	neutro            bool
	longsSeguidas     int64
	shortsSeguidas    int64
	qtdSeguidas       int64
	primeiraOrdem     string
	command           string
	now               time.Time
	ROI               float64
	num               int
	order             int
)

func main() {

	database.DBCon()

	config.ReadFile()

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
		fmt.Println("Irá trabalhar em LONG, SHORT ou NEUTRO? (ex: BUY, SELL, NEUTRO)")
		_, err = fmt.Scanln(&side)
		if err != nil {
			fmt.Println("Erro, tente digitar somente letras: ", err)
			continue
		}
		side = strings.ToUpper(side)
		if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" || side == "NEUTRO" {
			if side == "LONG" {
				side = "BUY"
			}
			if side == "SHORT" {
				side = "SELL"
			}
			if side == "NEUTRO" {
				neutro = true
				break
			}
			neutro = false
			break
		} else {
			fmt.Println("Deve entrar somente em LONG, SHORT ou NEUTRO")
			continue
		}
	} // side
	if side == "NEUTRO" {
		for {
			fmt.Println("Quer definir a primeira ordem em alguma direção? (BUY, SELL, DIGITE 'N' PARA NAO DEFINIR)")
			_, err = fmt.Scan(&primeiraOrdem)
			if err != nil {
				fmt.Println("Erro, tente digitar somente letras: ", err)
				continue
			}
			primeiraOrdem = strings.ToUpper(primeiraOrdem)
			if primeiraOrdem == "BUY" || primeiraOrdem == "SELL" || primeiraOrdem == "N" {
				break
			}

		}
	} // Definir a primeira entrada
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
		fmt.Println("Quer definir entrada e saída por VELAS ou por SEGUNDOS?")
		_, err = fmt.Scanln(&entrada)
		if err != nil {
			fmt.Println("Erro, tente digitar somente letras: ", err)
			continue
		}
		entrada = strings.ToUpper(entrada)
		if entrada == "VELAS" || entrada == "SEGUNDOS" {
			break
		} else {
			fmt.Println("Só pode escolher entre VELAS OU SEGUNDOS")
			continue
		}
	} // VELAS ou SEGUNDOS
	if entrada == "VELAS" {
		for {
			fmt.Println("Quantos segundos quer que cada vela? (Máximo 100)")
			_, err = fmt.Scanln(&velasperiodo)
			if err != nil {
				fmt.Println("Erro, tente digitar somente números: ", err)
				continue
			}
			if velasperiodo > 0 && velasperiodo <= 100 {
				break
			} else {
				continue
			}
		} // Quantidade de segundos que terá cada vela
		qtdPermitida := int64(math.Floor(300 / float64(velasperiodo)))
		for {
			fmt.Printf("Quantas velas quer calcular?. Máximo: %d\n", qtdPermitida)
			_, err = fmt.Scanln(&velasqtd)
			if err != nil {
				fmt.Println("Erro, tente digitar somente números: ", err)
				continue
			}
			if velasqtd > qtdPermitida {
				fmt.Println("Ultrapassou a quantidade máxima permitida.")
				continue
			} else if velasqtd < 2 {
				fmt.Println("É preciso pelo menos 2 velas para fazer uma comparação")
				continue
			} else {
				break
			}
		} // Quantidade de velas que irá ser usado como comparação
		for {
			fmt.Println("Quantos segundos quer comparar para saída (2 - 60): ")
			_, err = fmt.Scanln(&segSaida)
			if err != nil {
				fmt.Println("Erro, tente digitar somente números: ", err)
				continue
			}
			if segSaida > 60 {
				fmt.Println("Só é permitido comparar os ultimos 60 segundos.")
				continue
			} else if segSaida < 2 {
				fmt.Println("Só é permitido comparar pelo menos 2 segundos.")
				continue
			} else if segSaida >= 2 && segSaida <= 60 {
				segSaida++
				break
			}

		} // Quantidade de segundos para saída
	} else if entrada == "SEGUNDOS" {
		for {
			fmt.Println("Quantos segundos quer comparar para entrada (2 - 60): ")
			_, err = fmt.Scanln(&segEntrada)
			if err != nil {
				fmt.Println("Erro, tente digitar somente números: ", err)
				continue
			}
			if segEntrada > 60 {
				fmt.Println("Só é permitido comparar os ultimos 60 segundos.")
				continue
			} else if segEntrada < 2 {
				fmt.Println("Só é permitido comparar pelo menos 2 segundos.")
				continue
			} else if segEntrada >= 2 && segEntrada <= 60 {
				segEntrada++
				break
			}

		} // Quantidade de segundos para entrada
		for {
			fmt.Println("Quantos segundos quer comparar para saída (2 - 60): ")
			_, err = fmt.Scanln(&segSaida)
			if err != nil {
				fmt.Println("Erro, tente digitar somente números: ", err)
				continue
			}
			if segSaida > 60 {
				fmt.Println("Só é permitido comparar os ultimos 60 segundos.")
				continue
			} else if segSaida < 2 {
				fmt.Println("Só é permitido comparar pelo menos 2 segundos.")
				continue
			} else if segSaida >= 2 && segSaida <= 60 {
				segSaida++
				break
			}

		} // Quantidade de segundos para saída
	}
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
	for {
		fmt.Println("Quantas vezes quer seguir na mesma direção em seguidas. (Digite 0 pra desativar)")
		_, err = fmt.Scanln(&qtdSeguidas)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if qtdSeguidas < 0 {
			fmt.Println("Quantidade precisa ser maior que 0")
			continue
		} else {
			break
		}
	} // Quantidade seguidas na mesma direção

	fmt.Println("Para parar as transações pressione Ctrl + C")

	ordemAtiva = false
	primeiraExec = true
	valueCompradoCoin = 0.0
	roiAcumulado = 0.0
	fee := 0.05
	longsSeguidas = 0
	shortsSeguidas = 0

	util.DefinirAlavancagem(currentCoin, alavancagem)

	go handleCommands()

	// Encerrar a aplicação graciosamente
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("Sinal capturado: %v\n", sig)

		if ordemAtiva {
			if side == "BUY" {
				order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), currentPriceStr)
				if err != nil {
					log.Println("Erro ao fechar a ordem: ", err)
					return
				}
				if config.Development || order == 200 {
					util.Write("Ordens encerradas com sucesso ao finalizar a aplicação.", currentCoin+config.BaseCoin)
				} else {
					util.Write("Erro ao encerrar ordem. Finalize manulmente no site da Binance.", currentCoin+config.BaseCoin)
				}
			} else if side == "SELL" {
				order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), currentPriceStr)
				if err != nil {
					log.Println("Erro ao fechar a ordem: ", err)
					return
				}
				if config.Development || order == 200 {
					util.Write("Ordens encerradas com sucesso ao finalizar a aplicação.", currentCoin+config.BaseCoin)
				} else {
					util.Write("Erro ao encerrar ordem. Finalize manulmente no site da Binance.", currentCoin+config.BaseCoin)
				}

			}
		}
		err = criar_ordem.RemoverCoinDB(currentCoin)
		if err != nil {
			fmt.Println("Erro ao remover a moeda do banco de dados:", err)
		}

		os.Exit(0)
	}()

	for {
		criar_ordem.EnviarCoinDB(currentCoin)

		if primeiraExec {
			fmt.Println("Primeira execução. Estou lendo os primeiros valores.")
			if entrada == "SEGUNDOS" {
				time.Sleep(time.Duration(segEntrada+segSaida) * time.Second)
			} else if entrada == "VELAS" {
				time.Sleep(time.Duration(velasperiodo*velasqtd) * time.Second)
			}
			primeiraExec = false
			fmt.Println("Iniciado!! Aguarde a primeira ordem.")
		}

		if entrada == "SEGUNDOS" {
			ultimosEntrada = listar_ordens.ListarUltimosValores(currentCoin, segEntrada)
			ultimosSaida = listar_ordens.ListarUltimosValores(currentCoin, segSaida)
			currentPrice, err = strconv.ParseFloat(ultimosSaida[0].Price, 64)
		} else if entrada == "VELAS" {
			ultimosVelas = listar_ordens.ListarUltimosValores(currentCoin, velasqtd*velasperiodo)
			ultimosSaida = listar_ordens.ListarUltimosValores(currentCoin, segSaida)
			currentPrice, err = strconv.ParseFloat(ultimosVelas[0].Price, 64)
		}
		if err != nil {
			log.Println(err)
		}
		currentPriceStr = fmt.Sprint(currentPrice)
		if !ordemAtiva { // Não tem ordem ainda
			if neutro {
				side = "" // Zerar o side para garantir que sempre pegue as duas ordens.
			}
			if currentPrice > margemInferior && margemSuperior > currentPrice {
				if (neutro || side == "BUY") && (longsSeguidas < qtdSeguidas || qtdSeguidas == 0) && (primeiraOrdem == "BUY" || primeiraOrdem == "N") {
					if entrada == "SEGUNDOS" {
						if ultimosEntrada[0].Price > ultimosEntrada[int(segEntrada)-1].Price { // BUY
							for i := 0; i < int(segEntrada)-1; i++ {
								entrarBuy = false
								if ultimosEntrada[i].Price <= ultimosEntrada[i+1].Price {
									break
								}
								entrarBuy = true
							}
							if entrarBuy {
								o := comprarBuy()
								if config.Development {
									ordemAtiva = true
								} else {
									ordemAtiva = o == 200
								}
							}

						}
					} else if entrada == "VELAS" {
						num = 0
						for i := 1; i <= int(velasqtd); i++ {
							entrarBuy = false
							if ultimosVelas[num].Price < ultimosVelas[(int(velasperiodo)*i)-1].Price {
								break
							}
							num = num + int(velasperiodo)
							entrarBuy = true
						}
						if entrarBuy {
							o := comprarBuy()
							if config.Development {
								ordemAtiva = true
							} else {
								ordemAtiva = o == 200
							}
						}
					}
				}
				if (neutro || side == "SELL") && (shortsSeguidas < qtdSeguidas || qtdSeguidas == 0) && (primeiraOrdem == "SELL" || primeiraOrdem == "N") {
					if entrada == "SEGUNDOS" {
						if ultimosEntrada[0].Price < ultimosEntrada[int(segEntrada)-1].Price { // SELL
							for i := 0; i < int(segEntrada)-1; i++ {
								entrarSell = false
								if ultimosEntrada[i].Price >= ultimosEntrada[i+1].Price {
									break
								}
								entrarSell = true
							}
							if entrarSell {
								o := comprarSell()
								if config.Development {
									ordemAtiva = true
								} else {
									ordemAtiva = o == 200
								}
							}
						}
					} else if entrada == "VELAS" {
						num = 0
						for i := 1; i >= int(velasqtd); i++ {
							entrarSell = false
							if ultimosVelas[num].Price > ultimosVelas[(int(velasperiodo)*i)-1].Price {
								break
							}
							num = num + int(velasperiodo)
							entrarSell = true
						}
						if entrarSell {
							o := comprarSell()
							if config.Development {
								ordemAtiva = true
							} else {
								ordemAtiva = o == 200
							}
						}
					}
				}
			} else {
				util.Write("Atenção uma das margens foi atingida.: Margem Inferior: "+fmt.Sprint(margemInferior)+"- Margem Superior: "+fmt.Sprint(margemSuperior)+" - Preço atul: "+fmt.Sprint(currentPrice), currentCoin+config.BaseCoin)
				fmt.Println("\nDefina novos parametros.")
				for {
					fmt.Println("Irá trabalhar em LONG, SHORT ou NEUTRO? (ex: BUY, SELL, NEUTRO)")
					_, err = fmt.Scanln(&side)
					if err != nil {
						fmt.Println("Erro, tente digitar somente letras: ", err)
						continue
					}
					side = strings.ToUpper(side)
					if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" || side == "NEUTRO" {
						if side == "LONG" {
							side = "BUY"
						}
						if side == "SHORT" {
							side = "SELL"
						}
						if side == "NEUTRO" {
							neutro = true
							break
						}
						neutro = false
						break
					} else {
						fmt.Println("Deve entrar somente em LONG, SHORT ou NEUTRO")
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
		} else { // Já possui uma ordem ativa
			now = time.Now()
			timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
			formattedTime := timeValue.Format("2006-01-02 15:04:05")
			if side == "BUY" {
				ROI = (((currentPrice - valueCompradoCoin) / (valueCompradoCoin / alavancagem)) * 100) - (fee * 2)
				roiTempoReal := roiAcumulado + ROI
				util.Write("Valor de entrada (LONG): "+fmt.Sprint(valueCompradoCoin)+" | "+fmt.Sprintf("%.4f", ROI)+"% | "+formattedTime+" | "+fmt.Sprint(currentPrice)+" | Roi acumulado: "+fmt.Sprintf("%.4f", roiTempoReal)+"%", currentCoin+config.BaseCoin)

				if ROI > (fee * 2) {
					for i := 0; i < int(segSaida)-1; i++ {
						sairBuy = false
						if ultimosSaida[i].Price >= ultimosSaida[i+1].Price {
							break
						}
						sairBuy = true
					}
					if sairBuy {
						roiAcumulado = roiAcumulado + ROI
						util.Write("Ordem encerrada - desceu "+fmt.Sprint(segSaida-1)+" consecutivos após atingir o ROI. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
						order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), currentPriceStr)
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							ordemAtiva = false
						} else {
							util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
							ordemAtiva = true
						}
					}

				} else if currentPrice <= margemInferior {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - Atingiu margem inferior. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), currentPriceStr)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						fmt.Println("\nDefina novos parametros.")
						for {
							fmt.Println("Irá trabalhar em LONG, SHORT ou NEUTRO? (ex: BUY, SELL, NEUTRO)")
							_, err = fmt.Scanln(&side)
							if err != nil {
								fmt.Println("Erro, tente digitar somente letras: ", err)
								continue
							}
							side = strings.ToUpper(side)
							if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" || side == "NEUTRO" {
								if side == "LONG" {
									side = "BUY"
								}
								if side == "SHORT" {
									side = "SELL"
								}
								if side == "NEUTRO" {
									neutro = true
									break
								}
								neutro = false
								break
							} else {
								fmt.Println("Deve entrar somente em LONG, SHORT ou NEUTRO")
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
					} else {
						util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
						ordemAtiva = true
					}

				} else if ROI <= 0-(stop) {
					roiAcumulado = roiAcumulado + ROI
					util.Write("StopLoss atingido. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), currentPriceStr)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						ordemAtiva = false
					} else {
						util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
						ordemAtiva = true
					}
				} else if ROI <= 0-(stop)*75/100 {
					for i := 0; i < 2; i++ {
						slAtingido = false
						if ultimosSaida[i].Price >= ultimosSaida[i+1].Price {
							break
						}
						slAtingido = true
					}
					if slAtingido {
						roiAcumulado = roiAcumulado + ROI
						util.Write("75% stopLoss atingido e desceu "+fmt.Sprint(segSaida-1)+" vezes consecutivas. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
						order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), currentPriceStr)
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							ordemAtiva = false
						} else {
							util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
							ordemAtiva = true
						}
					}
				}
			} else if side == "SELL" {
				ROI = (((valueCompradoCoin - currentPrice) / (valueCompradoCoin / alavancagem)) * 100) - (fee * 2)
				roiTempoReal := roiAcumulado + ROI
				util.Write("Valor de entrada (SHORT): "+fmt.Sprint(valueCompradoCoin)+" | "+fmt.Sprintf("%.4f", ROI)+"% | "+formattedTime+" | "+currentPriceStr+" | Roi acumulado: "+fmt.Sprintf("%.4f", roiTempoReal)+"%", currentCoin+config.BaseCoin)
				if ROI >= (fee*2)*2 {
					for i := 0; i < int(segSaida)-1; i++ {
						sairSell = false
						if ultimosSaida[i].Price <= ultimosSaida[i+1].Price {
							break
						}
						sairSell = true

					}
					if sairSell {
						roiAcumulado = roiAcumulado + ROI
						util.Write("Ordem encerrada - subiu "+fmt.Sprint(segSaida-1)+" consecutivos após atingir o ROI. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
						order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), ultimosSaida[0].Price)
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							ordemAtiva = false
						} else {
							util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
							ordemAtiva = true
						}
					}

				} else if currentPrice >= margemSuperior {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - atingiu a margem superior. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), ultimosSaida[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						fmt.Println("\nDefina novos parametros.")
						for {
							fmt.Println("Irá trabalhar em LONG, SHORT ou NEUTRO? (ex: BUY, SELL, NEUTRO)")
							_, err = fmt.Scanln(&side)
							if err != nil {
								fmt.Println("Erro, tente digitar somente letras: ", err)
								continue
							}
							side = strings.ToUpper(side)
							if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" || side == "NEUTRO" {
								if side == "LONG" {
									side = "BUY"
								}
								if side == "SHORT" {
									side = "SELL"
								}
								if side == "NEUTRO" {
									neutro = true
									break
								}
								neutro = false
								break
							} else {
								fmt.Println("Deve entrar somente em LONG, SHORT ou NEUTRO")
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
					} else {
						util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
						ordemAtiva = true
					}

				} else if ROI <= 0-(stop) {
					roiAcumulado = roiAcumulado + ROI
					util.Write("Ordem encerrada - StopLoss atingido. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), ultimosSaida[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						ordemAtiva = false
					} else {
						util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
						ordemAtiva = true
					}
				} else if ROI <= 0-(stop)*75/100 {
					for i := 0; i < 2; i++ {
						slAtingido = false
						if ultimosSaida[i].Price <= ultimosSaida[i+1].Price {
							break
						}
						slAtingido = true
					}
					if slAtingido {
						roiAcumulado = roiAcumulado + ROI
						util.Write("75% stopLoss atingido e desceu "+fmt.Sprint(segSaida-1, 64)+" vezes consecutivas. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
						order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), currentPriceStr)
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							ordemAtiva = false
						} else {
							util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
							ordemAtiva = true
						}
					}

				}
			}
		}
		time.Sleep(900 * time.Millisecond)
	}
}

func comprarBuy() int {
	currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
	ultimosValores := "| "
	for i := 0; i < int(segEntrada)-1; i++ {
		ultimosValores += ultimosEntrada[i].Price + " | "
	}
	valueCompradoCoin = currentPrice
	util.Write("Entrada em LONG: "+currentPriceStr+". Ultimo valores: "+ultimosValores, currentCoin+config.BaseCoin)
	order, err = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue), currentPriceStr)
	if err != nil {
		log.Println("Erro ao criar conta: ", err)
	}
	if config.Development || order == 200 {
		side = "BUY"
		ordemAtiva = true
		longsSeguidas++
		shortsSeguidas = 0
		primeiraOrdem = "N"
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
	} else {
		util.Write("A ordem de LONG não foi totalmente completada. Irei voltar a buscar novas oportunidades. Pode a qualquer momento digitar BUY para entrar em LONG.", currentCoin+config.BaseCoin)
		ordemAtiva = false

	}
	return order

}

func comprarSell() int {
	currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
	ultimosValores := "| "
	for i := 0; i < int(segEntrada)-1; i++ {
		ultimosValores += ultimosEntrada[i].Price + " | "
	}
	valueCompradoCoin = currentPrice
	util.Write("Entrada em SHORT: "+currentPriceStr+". Ultimos valores: "+ultimosValores, currentCoin+config.BaseCoin)

	order, err = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue), currentPriceStr)
	if err != nil {
		log.Println("Erro ao criar conta: ", err)
	}
	if config.Development || order == 200 {
		side = "SELL"
		ordemAtiva = true
		shortsSeguidas++
		longsSeguidas = 0
		primeiraOrdem = "N"
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
	} else {
		util.Write("A ordem de SHORT não foi totalmente completada. Irei voltar a buscar novas oportunidades. Pode a qualquer momento digitar SELL para entrar em SHORT.", currentCoin+config.BaseCoin)
		ordemAtiva = false
	}
	return order

}

func handleCommands() {
	for {
		_, err = fmt.Scanln(&command)
		if err != nil {
			fmt.Println("Erro ao ler o comando:", err)
			continue
		}
		command = strings.ToUpper(command)

		switch strings.ToUpper(command) {
		case "BUY":
		case "LONG":
			if !ordemAtiva {
				o := comprarBuy()
				if config.Development || o == 200 {
					side = "BUY"
					break
				} else {
					break
				}

			} else {
				fmt.Println("\nJá tem uma ordem ativa.")
				break
			}
		case "SELL":
		case "SHORT":
			if !ordemAtiva {
				o := comprarSell()
				if config.Development || o == 200 {
					side = "SELL"
					break
				} else {
					break
				}
			} else {
				fmt.Println("\nJá tem uma ordem ativa.")
				break
			}
		case "NEUTRO":
			neutro = !neutro
			fmt.Println("Neutro ativado/desativado.")
			break
		case "STOP":
			if ordemAtiva {
				if side == "BUY" {
					roiAcumulado = roiAcumulado + ROI
					order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), ultimosSaida[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						util.Write("Ordem encerrada manualmente. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
						ordemAtiva = false
						break
					} else {
						fmt.Println("Erro ao encerrar a ordem, pode tentar novamente digitando STOP.")
						ordemAtiva = true
						break
					}
				} else if side == "SELL" {
					roiAcumulado = roiAcumulado + ROI
					order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), ultimosSaida[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						util.Write("Ordem encerrada manualmente. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
						ordemAtiva = false
						break
					} else {
						util.Write("Erro ao encerrar a ordem, pode tentar novamente digitando STOP.", currentCoin+config.BaseCoin)
						ordemAtiva = true
						break
					}
				}

			} else {
				fmt.Println("Não tens ordens ativas.")
				break
			}
		case "REVERSE":
		case "SWITCH":
			if ordemAtiva {
				if side == "BUY" {
					roiAcumulado = roiAcumulado + ROI
					order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), ultimosSaida[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if order != 200 {
						fmt.Println("Erro ao encerrar a ordem, pode tentar novamente digitando STOP.")
						ordemAtiva = true
						break
					} else {
						util.Write("Ordem encerrada manualmente. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
						ordemAtiva = false
						o := comprarSell()
						if config.Development || o == 200 {
							side = "SELL"
							ordemAtiva = true
							break
						} else {
							break
						}

					}

				} else if side == "SELL" {
					roiAcumulado = roiAcumulado + ROI
					order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), ultimosSaida[0].Price)
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						util.Write("Ordem encerrada manualmente. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado)+"%\n\n", currentCoin+config.BaseCoin)
						ordemAtiva = false
						o := comprarBuy()
						if o == 200 {
							side = "BUY"
							ordemAtiva = true
							break
						} else {
							break
						}
					} else {
						fmt.Println("Erro ao encerrar a ordem, pode tentar novamente digitando STOP.")
						ordemAtiva = true
						break
					}

				}
				break
			} else {
				fmt.Println("Não tens ordens ativas.")
				break
			}

		default:
			fmt.Println("Comando inválido. Tente: BUY/LONG(Entrar em LONG imediatamente), SELL/SHORT(Entrar em SHORT imediatamente), NEUTRO(Ativar/Desativar Neutro), REVERSE/SWITCH(Trocar de lado imediatamente), STOP(Parar a ordem imeditamente).")
			break
		}
	}
}
