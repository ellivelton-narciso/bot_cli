package main

import (
	"binance_robot/config"
	"binance_robot/criar_ordem"
	"binance_robot/database"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"binance_robot/util"
	"fmt"
	"github.com/fatih/color"
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
	currentCoin       string
	side              string
	value             float64
	margemInferior    float64
	margemSuperior    float64
	alavancagem       float64
	currentPrice      float64
	roi               float64
	err               error
	currentValue      float64
	currentPriceStr   string
	ordemAtiva        bool
	valueCompradoCoin float64
	primeiraExec      bool
	roiAcumulado      float64
	stop              float64
	stopLossAll       float64
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
	ROIStr            string
	roiAcumuladoStr   string
	roiTempoRealStr   string
	red               func(a ...interface{}) string
	green             func(a ...interface{}) string
	cmdRun            bool
	printUltimos      bool
	roiTempoReal      float64
)

func main() {
	database.DBCon()
	config.ReadFile()

	if config.ApiKey == "" || config.SecretKey == "" || config.BaseURL == "" || config.BaseCoin == "" {
		log.Panic("Arquivo user.json incompleto.")
	}

	red = color.New(color.FgHiRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	cmdRun = false
	ordemAtiva = false
	primeiraExec = true
	valueCompradoCoin = 0.0
	roiAcumulado = 0.0
	roiTempoReal = 0.0
	fee := 0.05
	longsSeguidas = 0
	shortsSeguidas = 0

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
			primeiraOrdem = side
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

		} // Definir a primeira entrada
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
	}
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
			fmt.Println("Quantos segundos quer que cada vela? (Min 3 - Máx 100)")
			_, err = fmt.Scanln(&velasperiodo)
			if err != nil {
				fmt.Println("Erro, tente digitar somente números: ", err)
				continue
			}
			if velasperiodo > 2 && velasperiodo <= 100 {
				break
			} else {
				continue
			}
		} // Quantidade de segundos que terá cada vela
		qtdPermitida := int64(math.Floor(300 / float64(velasperiodo)))
		for {
			fmt.Printf("Quantas velas quer calcular?. Min 2 -  Máx: %d\n", qtdPermitida)
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
		fmt.Println("Qual o Stop Loss por ORDEM que deseja trabalhar em porcentagem (ex: 0.5): ")
		_, err = fmt.Scanln(&stop)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if stop > 0 {
			break
		} else {
			fmt.Println("Stop Loss por ORDEM precisa ser maior que 0")
			continue
		}
	} // stopLoss
	for {
		fmt.Println("Qual o Stop Loss TOTAL que deseja trabalhar em porcentagem (ex: 0.5): ")
		_, err = fmt.Scanln(&stopLossAll)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if stopLossAll > stop {
			break
		} else {
			fmt.Println("Stop Loss TOTAL precisa ser maior que o Stop Loss por Ordem")
			continue
		}
	} // stopLoss Total
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
		fmt.Println("Qual será seu TAKE PROFIT em % total? (Ao atingir o valor a aplicação será encerrada totalmente).")
		_, err = fmt.Scanln(&roi)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if roi <= 0 {
			fmt.Println("TAKEPROFIT precisa ser maior que 0")
		} else {
			break
		}
	} // TAKEPROFIT Total

	fmt.Println("Para parar as transações pressione Ctrl + C")

	go handleCommands()

	util.DefinirAlavancagem(currentCoin, alavancagem)

	// Encerrar a aplicação graciosamente
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("Sinal capturado: %v\n", sig)
		roiTempoReal = roiAcumulado + ROI

		if ordemAtiva {
			if side == "BUY" {
				order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
				if err != nil {
					log.Println("Erro ao fechar a ordem: ", err)
					return
				}
				if config.Development || order == 200 {
					util.Write("Ordens encerradas com sucesso ao finalizar a aplicação.", currentCoin+config.BaseCoin)
					err = util.SalvarHistorico(currentCoin, side, "EXIT", currentPrice, roiTempoReal)
					if err != nil {
						util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
					}
				} else {
					util.Write("Erro ao encerrar ordem. Finalize manulmente no site da Binance.", currentCoin+config.BaseCoin)
				}
			} else if side == "SELL" {
				order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
				if err != nil {
					log.Println("Erro ao fechar a ordem: ", err)
					return
				}
				if config.Development || order == 200 {
					util.Write("Ordens encerradas com sucesso ao finalizar a aplicação.", currentCoin+config.BaseCoin)
					err = util.SalvarHistorico(currentCoin, side, "EXIT", currentPrice, roiTempoReal)
					if err != nil {
						util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
					}
				} else {
					util.Write("Erro ao encerrar ordem. Finalize manulmente no site da Binance.", currentCoin+config.BaseCoin)
				}
			}
		} else {
			fmt.Println(roiAcumulado)
			err = util.SalvarHistorico(currentCoin, side, "EXIT", currentPrice, roiAcumulado)
			if err != nil {
				util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
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
			err = util.SalvarHistorico(currentCoin, side, "START", currentPrice, roiAcumulado)
			if err != nil {
				util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
			}
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
			printUltimos = false
			if currentPrice > margemInferior && margemSuperior > currentPrice {
				if (neutro || side == "BUY") && (longsSeguidas < qtdSeguidas || qtdSeguidas == 0) && (primeiraOrdem == "BUY" || primeiraOrdem == "N") {
					if entrada == "SEGUNDOS" {
						ultimosValores := "| "
						for i := 0; i < int(segEntrada)-1; i++ {
							ultimosValores += ultimosEntrada[i].Price + " | "
						}
						if !printUltimos && !cmdRun {
							fmt.Println("Preço atual: "+currentPriceStr+" | Ultimos valores: "+ultimosValores, currentCoin+config.BaseCoin)
						}

						if ultimosEntrada[0].Price > ultimosEntrada[int(segEntrada)-1].Price { // BUY
							for i := 0; i < int(segEntrada)-1; i++ {
								entrarBuy = false
								if ultimosEntrada[i].Price <= ultimosEntrada[i+1].Price {
									break
								}
								entrarBuy = true
							}
							if entrarBuy {
								printUltimos = true
								o := comprarBuy()
								if config.Development {
									ordemAtiva = true
								} else {
									ordemAtiva = o == 200
									err = util.SalvarHistorico(currentCoin, "BUY", "COMPRA_BOT", currentPrice, roiTempoReal)
									if err != nil {
										util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
									}
								}
							}
						}
					} else if entrada == "VELAS" {
						num = 0
						for i := 1; i <= int(velasqtd); i++ {
							entrarBuy = false
							num2 := (int(velasperiodo) * i) - 1
							if (int(velasperiodo)*i)-1 >= len(ultimosVelas) {
								break
							}
							ultimosValores := "| "
							ultimosValores += ultimosVelas[num2].Price + " - " + ultimosVelas[num].Price + " | "
							if ultimosVelas[num].Price < ultimosVelas[num2].Price {
								if !printUltimos && !cmdRun {
									fmt.Println("Ultima vela de " + fmt.Sprint(velasperiodo) + " segundos: " + red(ultimosValores))
								}
								break
							}
							if !printUltimos && !cmdRun {
								fmt.Println("Ultima vela de " + fmt.Sprint(velasperiodo) + " segundos: " + green(ultimosValores))
							}
							num = num + int(velasperiodo)
							entrarBuy = true
						}
						if entrarBuy {
							printUltimos = true
							o := comprarBuy()
							if config.Development {
								ordemAtiva = true
							} else {
								ordemAtiva = o == 200
								err = util.SalvarHistorico(currentCoin, "BUY", "COMPRA_BOT", currentPrice, roiTempoReal)
								if err != nil {
									util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
								}
							}
						}
					}
				}
				if (neutro || side == "SELL") && (shortsSeguidas < qtdSeguidas || qtdSeguidas == 0) && (primeiraOrdem == "SELL" || primeiraOrdem == "N") {
					if entrada == "SEGUNDOS" {
						ultimosValores := "| "
						for i := 0; i < int(segEntrada)-1; i++ {
							ultimosValores += ultimosEntrada[i].Price + " | "
						}
						if !printUltimos && !cmdRun {
							fmt.Println("Preço atual: "+currentPriceStr+" | Ultimos valores: "+ultimosValores, currentCoin+config.BaseCoin)
						}
						if ultimosEntrada[0].Price < ultimosEntrada[int(segEntrada)-1].Price { // SELL
							for i := 0; i < int(segEntrada)-1; i++ {
								entrarSell = false
								if ultimosEntrada[i].Price >= ultimosEntrada[i+1].Price {
									break
								}
								entrarSell = true
							}
							if entrarSell {
								printUltimos = true
								o := comprarSell()
								if config.Development {
									ordemAtiva = true
								} else {
									ordemAtiva = o == 200
									err = util.SalvarHistorico(currentCoin, "SELL", "COMPRA_BOT", currentPrice, roiTempoReal)
									if err != nil {
										util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
									}
								}
							}
						}
					} else if entrada == "VELAS" {
						num = 0
						for i := 1; i <= int(velasqtd); i++ {
							entrarSell = false
							num2 := (int(velasperiodo) * i) - 1
							if num2 >= len(ultimosVelas) {
								break
							}
							ultimosValores := "| "
							ultimosValores += ultimosVelas[num2].Price + " - " + ultimosVelas[num].Price + " | "
							if ultimosVelas[num].Price > ultimosVelas[num2].Price {
								if !printUltimos && !cmdRun {
									fmt.Println("Ultima vela de " + fmt.Sprint(velasperiodo) + " segundos: " + green(ultimosValores))
								}
								break
							}
							if !printUltimos && !cmdRun {
								fmt.Println("Ultima vela de " + fmt.Sprint(velasperiodo) + " segundos: " + red(ultimosValores))
							}
							num = num + int(velasperiodo)
							entrarSell = true

						}
						if entrarSell {
							printUltimos = true
							o := comprarSell()
							if config.Development {
								ordemAtiva = true
							} else {
								ordemAtiva = o == 200
								err = util.SalvarHistorico(currentCoin, "SELL", "COMPRA_BOT", currentPrice, roiTempoReal)
								if err != nil {
									util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
								}
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
				roiTempoReal = roiAcumulado + ROI
				if ROI > 0 {
					ROIStr = green(fmt.Sprintf("%.4f", ROI) + "%")
				} else {
					ROIStr = red(fmt.Sprintf("%.4f", ROI) + "%")
				}
				if roiTempoReal > 0 {
					roiTempoRealStr = green(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				} else {
					roiTempoRealStr = red(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				}
				if !cmdRun {
					util.Write("Valor de entrada ("+green("LONG")+"): "+fmt.Sprint(valueCompradoCoin)+" | "+ROIStr+" | "+formattedTime+" | "+fmt.Sprint(currentPrice)+" | Roi acumulado: "+roiTempoRealStr, currentCoin+config.BaseCoin)
				}

				if roiTempoReal >= roi {
					roiAcumulado = roiAcumulado + ROI

					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Ordem encerrada - TAKE PROFIT atingido :). Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						ordemAtiva = false
						err = util.SalvarHistorico(currentCoin, side, "TPTOTAL_BOT", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
						os.Exit(0)
					} else {
						util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
						ordemAtiva = true
					}
				} else if ROI > (fee * 2) {
					for i := 0; i < int(segSaida)-1; i++ {
						sairBuy = false
						if ultimosSaida[i].Price >= ultimosSaida[i+1].Price {
							break
						}
						sairBuy = true
					}
					if sairBuy {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						util.Write("Ordem encerrada - desceu "+fmt.Sprint(segSaida-1)+" consecutivos após atingir o ROI. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
						order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							ordemAtiva = false
							err = util.SalvarHistorico(currentCoin, side, "TPORDEM_BOT", currentPrice, roiTempoReal)
							if err != nil {
								util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
							}
						} else {
							util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
							ordemAtiva = true
						}
					}

				} else if currentPrice <= margemInferior {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Ordem encerrada - Atingiu margem inferior. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						err = util.SalvarHistorico(currentCoin, side, "MARGEM_BOT", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
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
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("StopLoss atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						ordemAtiva = false
						err = util.SalvarHistorico(currentCoin, side, "SLORDEM_BOT", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
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
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						util.Write("75% stopLoss atingido e desceu "+fmt.Sprint(segSaida-1)+" vezes consecutivas. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
						order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							ordemAtiva = false
							err = util.SalvarHistorico(currentCoin, side, "SLORDEM_BOT", currentPrice, roiTempoReal)
							if err != nil {
								util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
							}
						} else {
							util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
							ordemAtiva = true
						}
					}
				} else if roiAcumulado <= 0-(stopLossAll) {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Stop Loss TOTAL atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						err = util.SalvarHistorico(currentCoin, side, "SLTOTAL_BOT", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
						os.Exit(1)
					} else {
						util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
						ordemAtiva = true
					}
				}
			} else if side == "SELL" {
				ROI = (((valueCompradoCoin - currentPrice) / (valueCompradoCoin / alavancagem)) * 100) - (fee * 2)
				roiTempoReal := roiAcumulado + ROI
				if ROI > 0 {
					ROIStr = green(fmt.Sprintf("%.4f", ROI) + "%")
				} else {
					ROIStr = red(fmt.Sprintf("%.4f", ROI) + "%")
				}

				if roiTempoReal > 0 {
					roiTempoRealStr = green(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				} else {
					roiTempoRealStr = red(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				}
				if !cmdRun {
					util.Write("Valor de entrada ("+red("SHORT")+"): "+fmt.Sprint(valueCompradoCoin)+" | "+ROIStr+" | "+formattedTime+" | "+currentPriceStr+" | Roi acumulado: "+roiTempoRealStr, currentCoin+config.BaseCoin)
				}

				if roiTempoReal >= roi {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Ordem encerrada - TAKE PROFIT atingido :). Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						ordemAtiva = false
						err = util.SalvarHistorico(currentCoin, side, "TPTOTAL_BOT", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
						os.Exit(0)
					} else {
						util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
						ordemAtiva = true
					}
				} else if ROI >= (fee*2)*2 {
					for i := 0; i < int(segSaida)-1; i++ {
						sairSell = false
						if ultimosSaida[i].Price <= ultimosSaida[i+1].Price {
							break
						}
						sairSell = true

					}
					if sairSell {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						util.Write("Ordem encerrada - subiu "+fmt.Sprint(segSaida-1)+" consecutivos após atingir o ROI. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
						order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							ordemAtiva = false
							err = util.SalvarHistorico(currentCoin, side, "TPORDEM_BOT", currentPrice, roiTempoReal)
							if err != nil {
								util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
							}
						} else {
							util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
							ordemAtiva = true
						}
					}

				} else if currentPrice >= margemSuperior {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Ordem encerrada - atingiu a margem superior. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						err = util.SalvarHistorico(currentCoin, side, "MARGEM_BOT", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
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
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Ordem encerrada - StopLoss atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						ordemAtiva = false
						err = util.SalvarHistorico(currentCoin, side, "SLORDEM_BOT", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
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
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						util.Write("75% stopLoss atingido e desceu "+fmt.Sprint(segSaida-1, 64)+" vezes consecutivas. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
						order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							ordemAtiva = false
							err = util.SalvarHistorico(currentCoin, side, "SLORDEM_BOT", currentPrice, roiTempoReal)
							if err != nil {
								util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
							}
						} else {
							util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
							ordemAtiva = true
						}
					}

				} else if roiAcumulado <= 0-(stopLossAll) {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Stop Loss TOTAL atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
					order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
					if err != nil {
						log.Println("Erro ao fechar a ordem: ", err)
						return
					}
					if config.Development || order == 200 {
						err = util.SalvarHistorico(currentCoin, side, "SLTOTAL_BOT", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
						os.Exit(1)
					} else {
						util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
						ordemAtiva = true
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
	side = "BUY"
	util.Write("Entrada em LONG: "+currentPriceStr+". Ultimos valores: "+ultimosValores, currentCoin+config.BaseCoin)
	order, err = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue))
	if err != nil {
		log.Println("Erro ao criar conta: ", err)
	}
	if config.Development || order == 200 {
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
		side = ""
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
	side = "SELL"
	order, err = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue))
	if err != nil {
		log.Println("Erro ao criar conta: ", err)
	}
	if config.Development || order == 200 {

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
		side = ""
		util.Write("A ordem de SHORT não foi totalmente completada. Irei voltar a buscar novas oportunidades. Pode a qualquer momento digitar SELL para entrar em SHORT.", currentCoin+config.BaseCoin)
		ordemAtiva = false
	}
	return order

}

func handleCommands() {
	if !cmdRun {
		for {
			_, err = fmt.Scanln(&command)
			if err != nil {
				fmt.Println("Erro ao ler o comando:", err)
				continue
			}
			command = strings.ToUpper(command)
			cmdRun = true
			switch strings.ToUpper(command) {
			case "BUY":
				if !ordemAtiva {
					o := comprarBuy()
					if config.Development || o == 200 {
						side = "BUY"
						err = util.SalvarHistorico(currentCoin, side, "BUY_CMD", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
						break
					} else {
						break
					}

				} else {
					fmt.Println("\nJá tem uma ordem ativa.")
					break
				}
			case "SELL":
				if !ordemAtiva {
					o := comprarSell()
					if config.Development || o == 200 {
						side = "SELL"
						err = util.SalvarHistorico(currentCoin, side, "SELL_CMD", currentPrice, roiTempoReal)
						if err != nil {
							util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
						}
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
				primeiraOrdem = "N"
				fmt.Println("Neutro ativado/desativado.")
				break
			case "STOP":
				if ordemAtiva {
					if side == "BUY" {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							util.Write("Ordem encerrada manualmente. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
							ordemAtiva = false
							err = util.SalvarHistorico(currentCoin, side, "STOP_CMD", currentPrice, roiAcumulado)
							if err != nil {
								util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
							}
							break
						} else {
							fmt.Println("Erro ao encerrar a ordem, pode tentar novamente digitando STOP.")
							ordemAtiva = true
							break
						}
					} else if side == "SELL" {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							util.Write("Ordem encerrada manualmente. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
							ordemAtiva = false
							err = util.SalvarHistorico(currentCoin, side, "STOP_CMD", currentPrice, roiAcumulado)
							if err != nil {
								util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
							}
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
				if ordemAtiva {
					if side == "BUY" {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							util.Write("Ordem encerrada manualmente. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
							ordemAtiva = false
							o := comprarSell()
							if config.Development || o == 200 {
								side = "SELL"
								ordemAtiva = true
								err = util.SalvarHistorico(currentCoin, side, "REVERSE_CMD", currentPrice, roiAcumulado)
								if err != nil {
									util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
								}
								break
							} else {
								break
							}
						} else {
							fmt.Println("Erro ao encerrar a ordem, pode tentar novamente digitando STOP.")
							ordemAtiva = true
							break
						}

					} else if side == "SELL" {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
						if err != nil {
							log.Println("Erro ao fechar a ordem: ", err)
							return
						}
						if config.Development || order == 200 {
							util.Write("Ordem encerrada manualmente. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin+config.BaseCoin)
							ordemAtiva = false
							o := comprarBuy()
							if o == 200 {
								side = "BUY"
								ordemAtiva = true
								err = util.SalvarHistorico(currentCoin, side, "REVERSE_CMD", currentPrice, roiTempoReal)
								if err != nil {
									util.Write(red("Erro ao salvar histórico: ", err), currentCoin)
								}
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
			case "MARGEM":
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
				break
			case "VALUE":
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
			case "ENTRADA":
				if entrada == "VELAS" {
					for {
						fmt.Println("Quantos segundos quer que cada vela? (Min 3 - Máx 100)")
						_, err = fmt.Scanln(&velasperiodo)
						if err != nil {
							fmt.Println("Erro, tente digitar somente números: ", err)
							continue
						}
						if velasperiodo > 2 && velasperiodo <= 100 {
							break
						} else {
							continue
						}
					} // Quantidade de segundos que terá cada vela
					qtdPermitida := int64(math.Floor(300 / float64(velasperiodo)))
					for {
						fmt.Printf("Quantas velas quer calcular?. Min 2 -  Máx: %d\n", qtdPermitida)
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
				}
			case "SAIDA":
				if entrada == "VELAS" {
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
			case "SL":
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
			case "SLT":
				for {
					fmt.Println("Qual o Stop Loss TOTAL que deseja trabalhar em porcentagem (ex: 0.5): ")
					_, err = fmt.Scanln(&stopLossAll)
					if err != nil {
						fmt.Println("Erro, tente digitar somente números: ", err)
						continue
					}
					if stopLossAll > stop {
						break
					} else {
						fmt.Println("Stop Loss TOTAL precisa ser maior que o Stop Loss por Ordem")
						continue
					}
				} // stopLoss Total
			default:
				fmt.Println("Comando inválido. Tente: BUY(Entrar em LONG imediatamente), SELL(Entrar em SHORT imediatamente), NEUTRO(Ativar/Desativar Neutro), REVERSE(Trocar de lado imediatamente), STOP(Parar a ordem imeditamente).")
				break
			}
			cmdRun = false
		}
	} else {
		fmt.Println("Não posso executar comandos ainda.")
	}
}
