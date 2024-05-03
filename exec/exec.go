package exec

import (
	"candles/config"
	"candles/global"
	"candles/listar"
	"candles/models"
	"candles/ordem"
	"candles/strategy"
	"candles/util"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func ExecutarOrdem(symbol, modo string, alavancagem float64, priority int) {
	config.ReadFile()

	var (
		quantity        float64
		posSide         string
		currentPrice    float64
		err             error
		currentPriceStr string
		primeiraExec    bool
		roiAcumulado    float64
		allOrders       []models.CryptoPosition
		ultimosSaida    []models.PriceResponse
		now             time.Time
		roi             float64
		order           int
		roiStr          string
		roiAcumuladoStr string
		roiTempoRealStr string
		roiTempoReal    float64
		date            string
		newSide         string
		roiMaximo       float64
		start           time.Time
		timeValue       time.Time
		priceBuy        float64
		paragem         float64
		canClose        bool
		condicaoOK      bool
		ultimosSaida15  []models.HistoricoAll
		condicaoLossOK  bool
		posSideText     string
		side            string
	)

	go func() {
		for {
			now = time.Now()
			if now.Hour() == 7 && !global.OrdemAtiva {
				primeiraExec = true
				roiAcumulado = 0.0
				roiTempoReal = 0.0
				roiMaximo = 0
				date = ""
				newSide = ""
				paragem = 0
				canClose = false
				condicaoLossOK = false
				condicaoOK = false
			}
			nextDay := now.AddDate(0, 0, 1)
			nextDay = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 7, 0, 0, 0, now.Location())
			time.Sleep(nextDay.Sub(now))
		}
	}()

	primeiraExec = true
	global.ValueCompradoCoin = 0.0
	roiAcumulado = 0.0
	roiTempoReal = 0.0
	roiMaximo = 0
	fee := (0.05 * 2) * alavancagem
	global.Alavancagem = alavancagem
	date = ""
	newSide = ""
	start = time.Now()
	timeValue = time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
	global.Started = timeValue.Format("2006-01-02 15:04:05")
	global.ForTime = time.Second
	paragem = 0
	global.NextValue = global.Value
	global.CmdRun = true
	canClose = false
	condicaoLossOK = false
	condicaoOK = false

	log.Println("Para parar as transações pressione Ctrl + C")

	if !config.Development {
		err = util.DefinirAlavancagem(symbol, alavancagem)
		if err != nil {
			err = ordem.RemoverCoinDB(symbol, global.Key)
			if err != nil {
				msgErr := "Erro ao remover " + symbol + " do banco de dados: "
				util.WriteError(msgErr, err, symbol)
				log.Println(msgErr, err)
				return
			}
			return

		}

		err = util.DefinirMargim(symbol, modo)
		if err != nil {
			err = ordem.RemoverCoinDB(symbol, global.Key)
			if err != nil {
				msgErr := "Erro ao remover " + symbol + " do banco de dados: "
				util.WriteError(msgErr, err, symbol)
				log.Println(msgErr, err)
				return
			}
			return
		}
	}

	// Commands

	go func() {
		var (
			comando string
		)
		time.Sleep(10 * time.Second)
		for {
			if !global.CmdRun {
				_, err = fmt.Scanln(&comando)
				comando = strings.ToUpper(comando)
				commandArr := strings.Split(comando, "-")
				if len(commandArr) > 1 {
					global.CmdRun = true
					if !strings.Contains(commandArr[1], config.BaseCoin) {
						commandArr[1] = commandArr[1] + config.BaseCoin
					}

					if commandArr[1] == symbol {
						switch strings.ToUpper(commandArr[0]) {
						case "BUY":
							if !global.OrdemAtiva {
								if err := strategy.InsertAlert(global.Key, symbol, "BUY"); err != nil {
									util.WriteError("["+symbol+"] Erro ao enviar alerta LONG para "+symbol+" key :"+global.Key+" - ", err, symbol)
								}
								break
							} else {
								fmt.Println("\nJá tem uma ordem ativa.")
								break
							}
						case "SELL":
							if !global.OrdemAtiva {
								if err := strategy.InsertAlert(global.Key, symbol, "SELL"); err != nil {
									util.WriteError("["+symbol+"] Erro ao enviar alerta LONG para "+symbol+" key :"+global.Key+" - ", err, symbol)
								}
								break
							} else {
								fmt.Println("\nJá tem uma ordem ativa.")
								break
							}
						case "STOP":
							if global.OrdemAtiva && global.CurrentCoin == symbol {
								if err := strategy.InsertAlert(global.Key, symbol, "STOP"); err != nil {
									util.WriteError("["+symbol+"] Erro ao enviar alerta LONG para "+symbol+" key :"+global.Key+" - ", err, symbol)
								}
								break

							} else {
								log.Println("Não tem nenhuma ordem ativa.")
								break
							}
						case "REVERSE":
							if global.OrdemAtiva && global.CurrentCoin == symbol {
								order = ordem.EncerrarOrdem(global.CurrentCoin, side, posSide, quantity)
								if config.Development || order == 200 {
									global.OrdemAtiva = false
									util.EncerrarHistorico(global.CurrentCoin, side, global.Started, currentPrice, roi)
								} else if order == 400 {
									break
								} else {
									util.Write("Erro ao encerrar ordem. ", global.CurrentCoin)
									break
								}

								util.Write("Revertendo...", symbol)
								time.Sleep(5 * time.Second)

								if side == "BUY" {
									if err := strategy.InsertAlert(global.Key, symbol, "SELL"); err != nil {
										util.WriteError("["+symbol+"] Erro ao enviar alerta LONG para "+symbol+" key :"+global.Key+" - ", err, symbol)
									}
								} else if side == "SELL" {
									if err := strategy.InsertAlert(global.Key, symbol, "BUY"); err != nil {
										util.WriteError("["+symbol+"] Erro ao enviar alerta LONG para "+symbol+" key :"+global.Key+" - ", err, symbol)
									}
								} else {
									util.Write("Side inválido", symbol)
								}
								break
							} else {
								log.Println("Não tens ordens ativas.")
								break
							}
						case "VALUE":
							for {
								fmt.Print("Digite a quantidade em " + config.BaseCoin + ": ")
								_, err = fmt.Scanln(&global.NextValue)
								if err != nil {
									fmt.Println("Erro, tente digitar somente números: ", err)
									continue
								}
								if global.Value > 0 {
									fmt.Println("Novo valor definido para a próxima ordem.")
									break
								} else {
									fmt.Println("Por favor, digite um valor válido.")
								}
							} // Value
							break
						case "SL":
							for {
								fmt.Println("Qual o Stop Loss que deseja trabalhar em porcentagem (ex: 0.5): ")
								_, err = fmt.Scanln(&global.Stop)
								if err != nil {
									fmt.Println("Erro, tente digitar somente números: ", err)
									continue
								}
								if global.Stop > 0 {
									break
								} else {
									fmt.Println("Stop Loss precisa ser maior que 0")
									continue
								}
							} // StopLoss
							break
						case "SLT":
							for {
								fmt.Println("Qual o Stop Loss TOTAL que deseja trabalhar em porcentagem (ex: 0.5): ")
								_, err = fmt.Scanln(&global.StopLossAll)
								if err != nil {
									fmt.Println("Erro, tente digitar somente números: ", err)
									continue
								}
								if global.StopLossAll > global.Stop {
									break
								} else {
									fmt.Println("Stop Loss TOTAL precisa ser maior que o Stop Loss por Ordem")
									continue
								}
							} // StopLoss Total
							break
						case "TP":
							for {
								fmt.Println("Qual será seu TAKE PROFIT em % total? (Ao atingir o valor a aplicação será encerrada totalmente).")
								_, err = fmt.Scanln(&global.TP)
								if err != nil {
									fmt.Println("Erro, tente digitar somente números: ", err)
									continue
								}
								if global.TP <= 0 {
									fmt.Println("TAKEPROFIT precisa ser maior que 0")
								} else {
									break
								}
							} // TAKEPROFIT Total
							break
						case "ENCERRAR":
							if global.OrdemAtiva && global.CurrentCoin == symbol {
								util.EncerrarHistorico(global.CurrentCoin, side, global.Started, currentPrice, roi)
							} else {
								fmt.Println("Não tem ordem ativa.")
							}
						default:
							log.Println("Comando inválido. Tente: BUY(Entrar em LONG imediatamente), SELL(Entrar em SHORT imediatamente), REVERSE(Trocar de lado imediatamente), STOP(Parar a ordem imeditamente).")
							break
						}
					}
					global.CmdRun = false
				} else {
					log.Println(`Comando Inválido, tente "BUY-BTCUSDT"`)
				}

				global.CmdRun = false
			}

		}
	}()

	// Encerrar a aplicação graciosamente
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("Sinal capturado: %v\n", sig)

		if global.OrdemAtiva && symbol == global.CurrentCoin {
			msg := fmt.Sprintf("Sinal capturado: %v", sig)

			order := ordem.EncerrarOrdem(symbol, side, posSide, quantity)
			if config.Development || order == 200 {
				util.Write(msg+" . Ordem encerrada: "+symbol, symbol)
				util.EncerrarHistorico(symbol, side, global.Started, currentPrice, roi)
			} else {
				msgErr := "Erro ao fechar a ordem de " + symbol + ", encerre manualmente pela binance: " + fmt.Sprint(order)
				util.WriteError(msgErr, err, symbol)
				log.Println(msgErr)
			}
		}
		err = ordem.RemoverCoinDB(symbol, global.Key)
		if err != nil {
			msgErr := "Erro ao remover " + symbol + " do banco de dados: "
			util.WriteError(msgErr, err, symbol)
			log.Println(msgErr, err)
		}
		fmt.Println("Encerrando")
		os.Exit(0)
	}()

	for {
		if global.OrdemAtiva && global.CurrentCoin == symbol || !global.OrdemAtiva {
			if primeiraExec {
				primeiraExec = false
				global.CmdRun = false
				fmt.Println("[" + symbol + "]Primeira execução. Estou lendo os primeiros valores. Aguarde 10s")
				time.Sleep(10 * time.Second)
				fmt.Println("Iniciado!! Aguarde a primeira ordem.")
				if !config.Development {
					allOrders, err = listar.ListarOrdens(symbol)
					if err != nil {
						primeiraExec = true
						global.CmdRun = true
						util.WriteError("Erro ao listar ordens: ", err, symbol)
						continue
					}
					for _, item := range allOrders {
						entryPriceFloat, _ := strconv.ParseFloat(item.EntryPrice, 64)
						if entryPriceFloat > 0 {
							util.Write("Ja possui ordem ativa.", symbol)
							return
						}
					}
					primeiraExec = false
					global.CmdRun = false
				}
			}

			ultimosSaida = listar.ListarUltimosValoresReais(symbol, 1)
			currentPrice, err = strconv.ParseFloat(ultimosSaida[0].Price, 64)
			if err != nil {
				log.Println(err)
			}
			currentPriceStr = fmt.Sprint(currentPrice)
			if !global.OrdemAtiva && !global.Meta { // Não tem ordem ainda e (não atingiu a meta ou perca total)
				canClose = false
				condicaoLossOK = false
				condicaoOK = false
				roiMaximo = 0
				paragem = 0
				time.Sleep(time.Duration(priority) * time.Second)
				side, date, err = util.VerificarHook(symbol, date)
				if side == "BUY" {
					posSide = "LONG"
				} else if side == "SELL" {
					posSide = "SHORT"
				}
				if err != nil {
					log.Println(err)
				}
				if side != "" {
					var o int
					side = strings.ToUpper(side)
					if side == "BUY" {
						start = time.Now()
						timeValue = time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
						global.Started = timeValue.Format("2006-01-02 15:04:05")
						global.Value = global.NextValue
						quantity, priceBuy = util.ConvertBaseCoin(symbol, global.Value*alavancagem)
						if quantity == 0 || priceBuy == 0 {
							util.Write("Valor atual ou Preço de compra é igual a 0", symbol)
							time.Sleep(time.Second)
							continue
						}
						global.ValueCompradoCoin = priceBuy

						o = ordem.ComprarBuy(symbol, fmt.Sprint(quantity), side, posSide, global.Stop)
						switch o {
						case 200:
							global.OrdemAtiva = true
							global.Side = side
							global.CurrentCoin = symbol
							if !config.Development {
								allOrders, err = listar.ListarOrdens(symbol)
								if err != nil {
									util.WriteError(global.Red("ATENÇÃO")+" - Erro ao listar ordem e você está com uma ordem possivelmente ativa: ", err, symbol)
									continue
								}
								for _, item := range allOrders {
									if item.PositionSide == posSide {
										global.ValueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
										if err != nil {
											log.Println("Erro ao buscar valor de entrada: ", err)
										}
										startedTimestamp := item.UpdateTime
										timeStarted := time.Unix(0, startedTimestamp*int64(time.Millisecond))
										global.Started = timeStarted.Format("2006-01-02 15:04:05")
									}
								}
							}
							err := util.SendMessageToDiscord("[" + symbol + "] Entrada em LONG em: " + fmt.Sprintf("%.6f", global.ValueCompradoCoin))
							if err != nil {
								util.Write("Erro ao enviar mensagem para o discord", symbol)
							}
							continue
						case 500:
							return
						default:
							global.OrdemAtiva = false
							continue
						}
					} else if side == "SELL" {
						start = time.Now()
						timeValue = time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
						global.Started = timeValue.Format("2006-01-02 15:04:05")
						global.Value = global.NextValue
						quantity, priceBuy = util.ConvertBaseCoin(symbol, global.Value*alavancagem)
						if quantity == 0 || priceBuy == 0 {
							util.Write("Valor atual ou Preço de compra é igual a 0", symbol)
							time.Sleep(time.Second)
							continue
						}
						global.ValueCompradoCoin = priceBuy

						o = ordem.ComprarSell(symbol, fmt.Sprint(quantity), side, posSide, global.Stop)
						switch o {
						case 200:
							global.OrdemAtiva = true
							global.Side = side
							global.CurrentCoin = symbol
							if !config.Development {
								allOrders, err = listar.ListarOrdens(symbol)
								if err != nil {
									util.WriteError(global.Red("ATENÇÃO")+" - Erro ao listar ordem e você está com uma ordem possivelmente ativa: ", err, symbol)
									continue
								}
								for _, item := range allOrders {
									if item.PositionSide == "SHORT" {
										if !config.Development {
											global.ValueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
											if err != nil {
												log.Println("Erro ao buscar valor de entrada: ", err)
												global.ValueCompradoCoin = priceBuy
											}
											startedTimestamp := item.UpdateTime
											timeStarted := time.Unix(0, startedTimestamp*int64(time.Millisecond))
											global.Started = timeStarted.Format("2006-01-02 15:04:05")
										}
									}
								}
							}
							err := util.SendMessageToDiscord("[" + symbol + "] Entrada em SHORT em: " + fmt.Sprintf("%.6f", global.ValueCompradoCoin))
							if err != nil {
								util.Write("Erro ao enviar mensagem para o discord", symbol)
							}
							continue
						case 500:
							return
						default:
							global.OrdemAtiva = false
							continue
						}
					}
				}
				time.Sleep(time.Second)
				continue
			} else if global.OrdemAtiva && global.CurrentCoin == symbol { // Já possui uma ordem ativa
				global.Side = side
				now = time.Now()
				timeValue = time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
				if posSide == "LONG" {
					posSideText = global.Green(posSide)
					roi = (((currentPrice - global.ValueCompradoCoin) / (global.ValueCompradoCoin / alavancagem)) * 100) - (fee)
				} else if posSide == "SHORT" {
					posSideText = global.Red(posSide)
					roi = (((global.ValueCompradoCoin - currentPrice) / (global.ValueCompradoCoin / alavancagem)) * 100) - (fee)
				} else {
					posSideText = global.Red("ERRO AO OBTER DIREÇÃO")
				}

				if roi > roiMaximo {
					roiMaximo = roi
					if global.StopMovel {
						paragem = roiMaximo
					}
				}
				roiTempoReal = roiAcumulado + roi
				if roi > 0 {
					roiStr = global.Green(fmt.Sprintf("%.4f", roi) + "%")
				} else {
					roiStr = global.Red(fmt.Sprintf("%.4f", roi) + "%")
				}
				if roiTempoReal > 0 {
					roiTempoRealStr = global.Green(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				} else {
					roiTempoRealStr = global.Red(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				}

				if !global.CmdRun {
					util.Write("["+global.CurrentCoin+"] Valor de entrada ("+posSideText+"): "+fmt.Sprintf("%.4f", global.ValueCompradoCoin)+" | "+roiStr+" | "+fmt.Sprint(currentPrice)+" | Roi acumulado: "+roiTempoRealStr, global.CurrentCoin)
				}
				newSide, date, err = util.VerificarHook(global.CurrentCoin, date)
				if err != nil {
					log.Println(err)
				}
				newSide = strings.ToUpper(newSide)

				ultimosSaida15, err = listar.ListarUltimosValores(global.CurrentCoin, 4)
				if err != nil {
					util.WriteError("Erro ao listar ultimos valores, ", err, global.CurrentCoin)
					continue
				}

				if canClose && roiTempoReal < global.TP || condicaoOK {
					roiAcumulado = roiAcumulado + roi

					if roiAcumulado > 0 {
						roiAcumuladoStr = global.Green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = global.Red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Ordem encerrada - TAKE PROFIT atingido :). Roi acumulado: "+roiAcumuladoStr+"\n\n", global.CurrentCoin)
					err = util.SendMessageToDiscord("[" + symbol + "] Ordem encerrada - TAKE PROFIT atingido :)")
					if err != nil {
						util.WriteError("Erro ao enviar mensagem pro Discord: ", err, symbol)
					}
					global.Meta = true
					if !config.Meta {
						global.Meta = false
						roiAcumulado = 0.0
						roiMaximo = 0.0
						roiTempoReal = 0.0
						roiTempoRealStr = ""
						roiAcumuladoStr = ""
					}
					global.OrdemAtiva = false
					order = ordem.EncerrarOrdem(global.CurrentCoin, side, posSide, quantity)
					if config.Development || order == 200 {
						util.EncerrarHistorico(global.CurrentCoin, side, global.Started, currentPrice, roi)
						continue
					} else if order == 400 {
						return
					} else {
						util.Write("Erro ao encerrar ordem. ", global.CurrentCoin)
						continue
					}
				} else if roiTempoReal >= global.TP*2 && roi < roiMaximo {
					roiAcumulado = roiAcumulado + roi

					if roiAcumulado > 0 {
						roiAcumuladoStr = global.Green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = global.Red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Ordem encerrada - TAKE PROFIT atingido :). Roi acumulado: "+roiAcumuladoStr+"\n\n", global.CurrentCoin)
					err = util.SendMessageToDiscord("[" + symbol + "] Ordem encerrada - TAKE PROFIT atingido :)")
					if err != nil {
						util.WriteError("Erro ao enviar mensagem pro Discord: ", err, symbol)
					}
					global.Meta = true
					if !config.Meta {
						global.Meta = false
						roiAcumulado = 0.0
						roiMaximo = 0.0
						roiTempoReal = 0.0
						roiTempoRealStr = ""
						roiAcumuladoStr = ""
					}
					global.OrdemAtiva = false
					order = ordem.EncerrarOrdem(global.CurrentCoin, side, posSide, quantity)
					if config.Development || order == 200 {
						global.OrdemAtiva = false

						util.EncerrarHistorico(global.CurrentCoin, side, global.Started, currentPrice, roi)
						continue
					} else if order == 400 {
						return
					} else {
						util.Write("Erro ao encerrar ordem. ", global.CurrentCoin)
						continue
					}
				} else if roiTempoReal <= 0-(global.StopLossAll) { // STOP LOSS Total
					roiAcumulado = roiAcumulado + roi
					if roiAcumulado > 0 {
						roiAcumuladoStr = global.Green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = global.Red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Stop Loss TOTAL atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", global.CurrentCoin)
					err = util.SendMessageToDiscord("[" + symbol + "] Stop Loss TOTAL atingido.")
					if err != nil {
						util.WriteError("Erro ao enviar mensagem pro Discord: ", err, symbol)
					}
					global.Meta = true
					if !config.Meta {
						global.Meta = false
						roiAcumulado = 0.0
						roiMaximo = 0.0
						roiTempoReal = 0.0
						roiTempoRealStr = ""
						roiAcumuladoStr = ""
					}
					global.OrdemAtiva = false
					order = ordem.EncerrarOrdem(global.CurrentCoin, side, posSide, quantity)
					if config.Development || order == 200 {
						util.EncerrarHistorico(global.CurrentCoin, side, global.Started, currentPrice, roi)
						continue
					} else if order == 400 {
						return
					} else {
						util.Write("Erro ao encerrar ordem.", global.CurrentCoin)
						global.OrdemAtiva = false
						continue
					}
				} else if roi <= paragem-(global.Stop) || condicaoLossOK { // STOP LOSS por ordem
					roiAcumulado = roiAcumulado + roi
					if roiAcumulado > 0 {
						roiAcumuladoStr = global.Green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = global.Red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("StopLoss atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", global.CurrentCoin)
					err = util.SendMessageToDiscord("[" + symbol + "] Stop Loss atingido.")
					order = ordem.EncerrarOrdem(global.CurrentCoin, side, posSide, quantity)
					if config.Development || order == 200 {
						util.EncerrarHistorico(global.CurrentCoin, side, global.Started, currentPrice, roi)
						global.OrdemAtiva = false
					} else if order == 400 {
						return
					} else {
						util.Write("Erro ao encerrar ordem.", global.CurrentCoin)
						time.Sleep(time.Second)
					}
				}

				if side == "BUY" {
					if newSide != side && newSide != "" {
						roiAcumulado = roiAcumulado + roi
						order = ordem.EncerrarOrdem(global.CurrentCoin, side, posSide, quantity)
						if config.Development || order == 200 {
							global.OrdemAtiva = false
							util.EncerrarHistorico(global.CurrentCoin, side, global.Started, currentPrice, roi)
							if newSide == "STOP" {
								util.Write("Ordem encerrada.", global.CurrentCoin)
							} else {
								side = newSide
								posSide = "SHORT"

								start = time.Now()
								timeValue = time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
								global.Started = timeValue.Format("2006-01-02 15:04:05")
								global.Value = global.NextValue
								quantity, priceBuy = util.ConvertBaseCoin(global.CurrentCoin, global.Value*alavancagem)
								if quantity == 0 || priceBuy == 0 {
									util.Write("Valor atual ou Preço de compra é igual a 0", global.CurrentCoin)
									time.Sleep(time.Second)
									continue
								}
								global.ValueCompradoCoin = priceBuy

								o := ordem.ComprarSell(global.CurrentCoin, fmt.Sprint(quantity), side, posSide, global.Stop)
								switch o {
								case 200:
									global.OrdemAtiva = true
									if !config.Development {
										allOrders, err = listar.ListarOrdens(global.CurrentCoin)
										if err != nil {
											util.WriteError(global.Red("ATENÇÃO")+" - Erro ao listar ordem e você está com uma ordem possivelmente ativa: ", err, global.CurrentCoin)
											continue
										}
										for _, item := range allOrders {
											if item.PositionSide == "SHORT" {
												if !config.Development {
													global.ValueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
													if err != nil {
														log.Println("Erro ao buscar valor de entrada: ", err)
														global.ValueCompradoCoin = priceBuy
													}
													startedTimestamp := item.UpdateTime
													timeStarted := time.Unix(0, startedTimestamp*int64(time.Millisecond))
													global.Started = timeStarted.Format("2006-01-02 15:04:05")
												}
											}
										}
										err = util.SendMessageToDiscord("[" + symbol + "] Entrada em SHORT em: " + fmt.Sprintf("%.6f", global.ValueCompradoCoin))
										if err != nil {
											util.WriteError("Erro ao enviar mensagem para o discord, ", err, symbol)
										}
									}
									continue
								case 500:
									return
								default:
									global.OrdemAtiva = false
									continue
								}
							}
						} else if order == 400 {
							return
						} else {
							util.Write("Erro ao encerrar ordem.", global.CurrentCoin)
							time.Sleep(time.Second)
							continue
						}

					}
					if roiTempoReal >= global.TP { // TAKE PROFIT Total
						canClose = true

						ultimoMinuto, err := listar.ListarValorAnterior(global.CurrentCoin)
						ultimoMinutoStr := fmt.Sprint(ultimoMinuto)
						if err != nil {
							util.WriteError("Erro ao buscar valor anterior para compararar: ", err, global.CurrentCoin)
							continue
						}
						util.Write("Valor atual: "+currentPriceStr+" Valor 5min atrás: "+ultimoMinutoStr, global.CurrentCoin)
						if currentPrice < ultimoMinuto {
							condicaoOK = true
							continue
						} else {
							ultimosValores := "| "
							for i := 0; i < 3; i++ {
								condicaoOK = false
								if ultimosSaida15[i].CurrentValue <= ultimosSaida15[i+1].CurrentValue {
									break
								}
								ultimosValores += ultimosSaida15[i].CurrentValue + " | "
								condicaoOK = true
							}
							util.Write("Ultimos valores a cada 15s "+ultimosValores, global.CurrentCoin)
							if condicaoOK {
								continue
							}
						}
					} else if roi <= paragem-(global.Stop/2) && now.Sub(start) >= 5*time.Minute {
						volume1h, err := listar.GetVolumeData(global.CurrentCoin, "1h", 1)
						if err != nil {
							util.WriteError("Erro ao buscar dados do volume, ", err, global.CurrentCoin)
						}
						volume5m, err := listar.GetVolumeData(global.CurrentCoin, "5m", 1)
						if err != nil {
							util.WriteError("Erro ao buscar dados do volume, ", err, global.CurrentCoin)
						}
						util.Write("Ratio Volume 1h: "+fmt.Sprintf("%.4f", volume1h[0].RatioVolume)+", Ratio Volume 5m: "+fmt.Sprintf("%.4f", volume5m[0].RatioVolume), symbol)

						// Volume de venda de 1h 50% superior que o de compra e volume de venda 5m 100% superior que o de compra
						if volume1h[0].RatioVolume <= 1/1.5 && volume5m[0].RatioVolume <= 1/2 {
							condicaoLossOK = true
							util.Write("Volume desfavorável para continuar. Encerrando para evitar perdas maiores.", global.CurrentCoin)
							err = util.SendMessageToDiscord("Volume desfavorável para continuar.")
							if err != nil {
								util.WriteError("Erro ao enviar mensagem para o discord, ", err, symbol)
							}
							continue
						}
						if roiMaximo <= 0 {
							condicaoLossOK = true
							util.Write("Possível tendencia contrária identificado. ", symbol)
							err = util.SendMessageToDiscord("Possível tendencia contrária identificado. ")
							if err != nil {
								util.WriteError("Erro ao enviar mensagem para o discord, ", err, symbol)
							}
							continue
						}

					}
				} else if side == "SELL" {
					if newSide != side && newSide != "" {
						roiAcumulado = roiAcumulado + roi
						order = ordem.EncerrarOrdem(global.CurrentCoin, side, posSide, quantity)
						if config.Development || order == 200 {
							global.OrdemAtiva = false
							util.EncerrarHistorico(global.CurrentCoin, side, global.Started, currentPrice, roi)
							if newSide == "STOP" {
								util.Write("Ordem encerrada.", global.CurrentCoin)
							} else {
								side = newSide
								posSide = "LONG"

								start = time.Now()
								timeValue = time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
								global.Started = timeValue.Format("2006-01-02 15:04:05")
								global.Value = global.NextValue
								quantity, priceBuy = util.ConvertBaseCoin(global.CurrentCoin, global.Value*alavancagem)
								if quantity == 0 || priceBuy == 0 {
									util.Write("Valor atual ou Preço de compra é igual a 0", global.CurrentCoin)
									time.Sleep(time.Second)
									continue
								}
								global.ValueCompradoCoin = priceBuy

								o := ordem.ComprarBuy(global.CurrentCoin, fmt.Sprint(quantity), side, posSide, global.Stop)
								switch o {
								case 200:
									global.OrdemAtiva = true
									if !config.Development {
										allOrders, err = listar.ListarOrdens(global.CurrentCoin)
										if err != nil {
											util.WriteError(global.Red("ATENÇÃO")+" - Erro ao listar ordem e você está com uma ordem possivelmente ativa: ", err, global.CurrentCoin)
											continue
										}
										for _, item := range allOrders {
											if item.PositionSide == posSide {
												global.ValueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
												if err != nil {
													log.Println("Erro ao buscar valor de entrada: ", err)
												}
												startedTimestamp := item.UpdateTime
												timeStarted := time.Unix(0, startedTimestamp*int64(time.Millisecond))
												global.Started = timeStarted.Format("2006-01-02 15:04:05")
											}
										}
									}
									err = util.SendMessageToDiscord("[" + symbol + "] Entrada em LONG em: " + fmt.Sprintf("%.6f", global.ValueCompradoCoin))
									if err != nil {
										util.WriteError("Erro ao enviar mensagem para o discord, ", err, symbol)
									}
									continue
								case 500:
									return
								default:
									global.OrdemAtiva = false
									continue
								}
							}

						} else if order == 400 {
							return
						} else {
							util.Write("Erro ao encerrar ordem. ", global.CurrentCoin)
							global.OrdemAtiva = true
							time.Sleep(time.Second)
							continue
						}

					}
					if roiTempoReal >= global.TP { // TAKE PROFIT Total
						canClose = true

						ultimoMinuto, err := listar.ListarValorAnterior(global.CurrentCoin)
						ultimoMinutoStr := fmt.Sprint(ultimoMinuto)
						if err != nil {
							util.WriteError("Erro ao buscar valor anterior para compararar: ", err, global.CurrentCoin)
							continue
						}
						util.Write("Valor atual: "+currentPriceStr+" Valor 5min atrás: "+ultimoMinutoStr, global.CurrentCoin)
						if currentPrice > ultimoMinuto {
							condicaoOK = true
							continue
						} else {
							ultimosValores := "| "
							for i := 0; i < 3; i++ {
								condicaoOK = false
								if ultimosSaida15[i].CurrentValue >= ultimosSaida15[i+1].CurrentValue {
									break
								}
								ultimosValores += ultimosSaida15[i].CurrentValue + " | "
								condicaoOK = true
							}
							util.Write("Ultimos valores a cada 15s "+ultimosValores, global.CurrentCoin)
							if condicaoOK {
								continue
							}
						}
					} else if roi <= paragem-(global.Stop/2) && now.Sub(start) >= 5*time.Minute {
						volume1h, err := listar.GetVolumeData(global.CurrentCoin, "1h", 1)
						if err != nil {
							util.WriteError("Erro ao buscar dados do volume, ", err, global.CurrentCoin)
						}
						time.Sleep(1 * time.Second)
						volume5m, err := listar.GetVolumeData(global.CurrentCoin, "5m", 1)
						if err != nil {
							util.WriteError("Erro ao buscar dados do volume, ", err, global.CurrentCoin)
						}
						util.Write("Ratio Volume 1h: "+fmt.Sprintf("%.4f", volume1h[0].RatioVolume)+", Ratio Volume 5m: "+fmt.Sprintf("%.4f", volume5m[0].RatioVolume), symbol)

						// Volume de compra de 1h 50% superior que o de venda e volume de compra de 5m 100% superior que o de venda
						if volume1h[0].RatioVolume >= 1.5 && volume5m[0].RatioVolume >= 2 {
							condicaoLossOK = true
							util.Write("Volume desfavorável para continuar. Encerrando para evitar perdas maiores.", global.CurrentCoin)
							continue
						}
						if roiMaximo <= 0 {
							condicaoLossOK = true
							util.Write("Possível tendencia contrária identificado. ", symbol)
							err = util.SendMessageToDiscord("Possível tendencia contrária identificado. ")
							if err != nil {
								util.WriteError("Erro ao enviar mensagem para o discord, ", err, symbol)
							}
							continue
						}

					}
				} else {
					log.Println("Global Side: " + global.Side)
				}
			}
		}
		time.Sleep(global.ForTime)
	}
}
