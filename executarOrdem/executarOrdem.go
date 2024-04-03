package executarOrdem

import (
	"binance_robot/config"
	"binance_robot/criar_ordem"
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

func OdemExecucao(currentCoin, side string, value, alavancagem, stop, takeprofit float64) {

	var (
		currentPrice        float64
		err                 error
		currentValue        float64
		currentPriceStr     string
		ordemAtiva          bool
		valueCompradoCoin   float64
		primeiraExec        bool
		roiAcumulado        float64
		allOrders           []models.CryptoPosition
		ultimosSaida        []models.HistoricoAll
		now                 time.Time
		start               time.Time
		ROI                 float64
		order               int
		slSeguro            int
		roiAcumuladoStr     string
		roiTempoRealStr     string
		red                 func(a ...interface{}) string
		green               func(a ...interface{}) string
		roiMaximo           float64
		started             string
		currValueTelegram   float64
		currentDateTelegram string
		resposta            string
		precision           int
		forTime             time.Duration
		priceBuy            float64
		condicaoOK          bool
		canClose            bool
	)

	red = color.New(color.FgHiRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	fee := 0.05 * alavancagem
	ordemAtiva = false
	primeiraExec = true
	valueCompradoCoin = 0.0
	roiAcumulado = 0.0
	forTime = 900 * time.Millisecond
	roiMaximo = 0
	condicaoOK = false
	canClose = false

	side = strings.ToUpper(side)
	if side == "LONG" {
		side = "BUY"
	}
	if side == "SHORT" {
		side = "SELL"
	}
	tg := util.BuscarValoresTelegram(currentCoin)
	if len(tg) == 0 {
		util.Write("Erro ao fazer busca do telegram.", currentCoin)
		err = criar_ordem.RemoverCoinDBW(currentCoin)
		if err != nil {
			util.Write("Erro ao remover "+currentCoin+" do banco de dados", currentCoin)
			time.Sleep(2 * time.Second)
			return
		}
		time.Sleep(2 * time.Second)
		return
	}
	currValueTelegram = tg[0].CurrValue
	currentDateTelegram = tg[0].HistDate.Format("2006-01-02 15:04:05")
	stop = (stop * alavancagem) + (fee * 2)
	takeprofit = (takeprofit * alavancagem) - (fee * 2)

	err = util.DefinirAlavancagem(currentCoin, alavancagem)
	if err != nil {
		if !config.Development {
			return
		}
	}
	err = util.DefinirMargim(currentCoin, "ISOLATED")
	if err != nil {
		if !config.Development {
			return
		}
	}
	criar_ordem.EnviarCoinDB(currentCoin)
	precision, err = util.GetPrecision(currentCoin)
	if err != nil {
		precision = 0
		util.WriteError("Erro ao buscar precisao para converter a moeda: ", err, currentCoin)
	}

	// Encerrar a aplicação graciosamente
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan

		if ordemAtiva {
			msg := fmt.Sprintf("Sinal capturado: %v", sig)

			order := encerrarOrdem(currentCoin, side, currentValue)
			time.Sleep(5 * time.Second)
			if config.Development || order == 200 {
				util.Write(msg+" . Ordem encerrada: "+currentCoin, currentCoin)
				util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
				err = criar_ordem.RemoverCoinDBW(currentCoin)
				if err != nil {
					msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
					util.WriteError(msgErr, err, currentCoin)
					fmt.Println(msgErr, err)
					return
				}
				return
			} else {
				msgErr := "Erro ao fechar a ordem de " + currentCoin + ", encerre manualmente pela binance: "
				util.WriteError(msgErr, err, currentCoin)
				fmt.Println(msgErr)
			}
		}

		os.Exit(0)
	}()

	for {
		if primeiraExec {
			if !config.Development {
				allOrders, err = listar_ordens.ListarOrdens(currentCoin)
				if err != nil {
					primeiraExec = true
					util.WriteError("Erro ao listar ordens: ", err, currentCoin)
					continue
				}
				for _, item := range allOrders {
					entryPriceFloat, _ := strconv.ParseFloat(item.EntryPrice, 64)
					if entryPriceFloat > 0 {
						util.Write("Ja possui ordem ativa.", currentCoin)
						ordemAtiva = true
					}
				}
				if ordemAtiva {
					return
				}
				primeiraExec = false
			} else {
				primeiraExec = false
			}
		}

		ultimosSaida, err = listar_ordens.ListarUltimosValores(currentCoin, 4)
		if err != nil {
			util.WriteError("Erro ao listar ultimos valores, ", err, currentCoin)
			continue
		}

		currentPrice, err = strconv.ParseFloat(ultimosSaida[0].CurrentValue, 64)
		if err != nil {
			util.WriteError("Erro no array currentPrice: ", err, currentCoin)
			continue
		}
		currentPriceStr = fmt.Sprint(currentPrice)
		if !ordemAtiva { // Não tem ordem ainda

			if side == "BUY" {
				start = time.Now()
				timeValue := time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
				started = timeValue.Format("2006-01-02 15:04:05")
				currentValue, priceBuy = util.ConvertBaseCoin(currentCoin, value*alavancagem)
				if currentValue == 0 || priceBuy == 0 {
					util.Write("Valor atual ou Preço de compra é igual a 0", currentCoin)
					time.Sleep(1 * time.Second)
					continue
				}
				valueCompradoCoin = priceBuy

				if currValueTelegram < priceBuy*1.002 {
					err = criar_ordem.RemoverCoinDBW(currentCoin)
					if err != nil {
						log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
						time.Sleep(2 * time.Second)
						return
					}
					util.Write("LONG - Valor do Telegram é menor que o preço atual de mercado +  0.2%, Valor Telegram: "+fmt.Sprintf("%.4f", currValueTelegram)+" Valor atual + 0.2%: "+fmt.Sprintf("%.4f", priceBuy*1.002), currentCoin)
					time.Sleep(2 * time.Second)
					return
				}

				order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
				if err != nil {
					log.Println("Erro ao dar entrada m LONG: ", err)
					if !config.Development {
						allOrders, err = listar_ordens.ListarOrdens(currentCoin)
						if err != nil {
							util.WriteError("Erro ao listar ordens: ", err, currentCoin)
							return
						}
						for _, item := range allOrders {
							entryPriceFloat, _ := strconv.ParseFloat(item.EntryPrice, 64)
							if entryPriceFloat > 0 {
								util.Write("Ja possui ordem ativa.", currentCoin)
								ordemAtiva = true
							}
						}
						if !ordemAtiva {
							err = criar_ordem.RemoverCoinDBW(currentCoin)
							if err != nil {
								log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
								time.Sleep(2 * time.Second)
								return
							}
							time.Sleep(2 * time.Second)
							return
						}
						continue
					} else {
						err = criar_ordem.RemoverCoinDBW(currentCoin)
						if err != nil {
							log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
							time.Sleep(2 * time.Second)
							return
						}
						time.Sleep(2 * time.Second)
						return
					}
				}
				if config.Development || order == 200 {
					util.Write("Entrada em LONG: "+currentPriceStr+", TP: "+fmt.Sprintf("%.4f", takeprofit)+", SL: "+fmt.Sprintf("%.4f", stop), currentCoin)
					ordemAtiva = true
					allOrders, err = listar_ordens.ListarOrdens(currentCoin)
					if err != nil {
						log.Println("Erro ao listar ordens: ", err)
					}
					for _, item := range allOrders {
						if item.PositionSide == "BOTH" {
							if !config.Development {
								valueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
								if err != nil {
									log.Println("Erro ao buscar valor de entrada: ", err)
								}
								started_timestamp := item.UpdateTime
								timeStarted := time.Unix(0, started_timestamp*int64(time.Millisecond))
								started = timeStarted.Format("2006-01-02 15:04:05")
							}
						}
					}
					util.Historico(currentCoin, "BUY", started, "", currentDateTelegram, valueCompradoCoin, currValueTelegram, valueCompradoCoin, ROI)
					forTime = 5 * time.Second
					q := valueCompradoCoin * (1 - 0.025*1.2)
					stopSeguro := math.Round(q*math.Pow(10, float64(precision))) / math.Pow(10, float64(precision))
					slSeguro, resposta, err = criar_ordem.CriarSLSeguro(currentCoin, "SELL", fmt.Sprint(stopSeguro))
					if err != nil {
						log.Println("Erro ao criar Stop Loss Seguro para, ", currentCoin, " motivo: ", err)
						util.WriteError("Não foi criada ordem para STOPLOSS, motivo: ", err, currentCoin)
						continue
					}
					if slSeguro != 200 {
						util.Write("Stop Loss Seguro não criado, "+resposta, currentCoin)
						continue
					}
					util.Write("Stop Loss Seguro foi criado.", currentCoin)

				} else {
					util.Write("A ordem de LONG não foi totalmente completada.", currentCoin)
					ordemAtiva = false

				}
			} else if side == "SELL" {
				start = time.Now()
				timeValue := time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
				started = timeValue.Format("2006-01-02 15:04:05")
				currentValue, priceBuy = util.ConvertBaseCoin(currentCoin, value*alavancagem)
				if currentValue == 0 || priceBuy == 0 {
					util.Write("Valor atual ou Preço de compra é igual a 0", currentCoin)
					time.Sleep(1 * time.Second)
					continue
				}
				valueCompradoCoin = priceBuy

				if currValueTelegram > priceBuy*1.002 {
					err = criar_ordem.RemoverCoinDBW(currentCoin)
					if err != nil {
						log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
						time.Sleep(2 * time.Second)
						return
					}
					util.Write("SHORT - Valor do Telegram é maior que o preço atual de mercado +  0.2%, Valor Telegram: "+fmt.Sprintf("%.4f", currValueTelegram)+" Valor atual + 0.2%: "+fmt.Sprintf("%.4f", priceBuy*1.002), currentCoin)
					time.Sleep(2 * time.Second)
					return
				}
				order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
				if err != nil {
					log.Println("Erro ao dar entrada em SHORT: ", err)
					if !config.Development {
						allOrders, err = listar_ordens.ListarOrdens(currentCoin)
						if err != nil {
							util.WriteError("Erro ao listar ordens: ", err, currentCoin)
							return
						}
						for _, item := range allOrders {
							entryPriceFloat, _ := strconv.ParseFloat(item.EntryPrice, 64)
							if entryPriceFloat > 0 {
								util.Write("Ja possui ordem ativa.", currentCoin)
								ordemAtiva = true
							}
						}
						if !ordemAtiva {
							err = criar_ordem.RemoverCoinDBW(currentCoin)
							if err != nil {
								log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
								time.Sleep(2 * time.Second)
								return
							}
							time.Sleep(2 * time.Second)
							return
						}
						continue
					} else {
						err = criar_ordem.RemoverCoinDBW(currentCoin)
						if err != nil {
							log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
							time.Sleep(2 * time.Second)
							return
						}
						time.Sleep(2 * time.Second)
						return
					}
				}
				if config.Development || order == 200 {
					util.Write("Entrada em SHORT: "+currentPriceStr+", TP: "+fmt.Sprintf("%.4f", takeprofit)+", SL: "+fmt.Sprintf("%.4f", stop), currentCoin)
					ordemAtiva = true
					allOrders, err = listar_ordens.ListarOrdens(currentCoin)
					if err != nil {
						log.Println("Erro ao listar ordens: ", err)
					}
					for _, item := range allOrders {
						if item.PositionSide == "BOTH" {
							if !config.Development {
								valueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
								if err != nil {
									log.Println("Erro ao buscar valor de entrada: ", err)
								}
								startedTimestamp := item.UpdateTime
								timeStarted := time.Unix(0, startedTimestamp*int64(time.Millisecond))
								started = timeStarted.Format("2006-01-02 15:04:05")
							}
						}
					}
					util.Historico(currentCoin, "SELL", started, "", currentDateTelegram, valueCompradoCoin, currValueTelegram, valueCompradoCoin, ROI)
					forTime = 5 * time.Second
					stopSeguro := valueCompradoCoin * (1 + 0.025*1.2)
					slSeguro, resposta, err = criar_ordem.CriarSLSeguro(currentCoin, side, fmt.Sprint(stopSeguro))
					if err != nil {
						log.Println("Erro ao criar Stop Loss Seguro para, ", currentCoin, " motivo: ", err)
						util.WriteError("Não foi criada ordem para STOPLOSS, motivo: ", err, currentCoin)
						continue
					}
					if slSeguro != 200 {
						util.Write("Stop Loss Seguro não criado, "+resposta, currentCoin)
						continue
					}
					util.Write("Stop Loss Seguro foi criado.", currentCoin)

				} else {
					util.Write("A ordem de SHORT não foi totalmente completada. Irei voltar a buscar novas oportunidades. Pode a qualquer momento digitar SELL para entrar em SHORT.", currentCoin)
					ordemAtiva = false
				}
			}
		} else { // Já possui uma ordem ativa
			now = time.Now()
			timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
			formattedTime := timeValue.Format("2006-01-02 15:04:05")
			if side == "BUY" && !primeiraExec {
				ROI = (((currentPrice - valueCompradoCoin) / (valueCompradoCoin / alavancagem)) * 100) - (fee * 2)
				if ROI > roiMaximo {
					roiMaximo = ROI
				}
				if ROI > 0 {
					roiTempoRealStr = green(fmt.Sprintf("%.4f", ROI) + "%")
				} else {
					roiTempoRealStr = red(fmt.Sprintf("%.4f", ROI) + "%")
				}
				util.Write("Valor de entrada ("+green("LONG")+"): "+fmt.Sprint(valueCompradoCoin)+" | "+formattedTime+" | "+fmt.Sprint(currentPrice)+" | Roi acumulado: "+roiTempoRealStr, currentCoin)
				// Deverá descer 3 consecutivos para fechar.
				if len(ultimosSaida) >= 4 && ultimosSaida[0].CurrentValue < ultimosSaida[1].CurrentValue && now.Sub(start) >= 45*time.Second && (ROI > 0 && ROI < takeprofit) {
					ultimosValores := "| "
					for i := 0; i < 3; i++ {
						condicaoOK = false
						if ultimosSaida[i].CurrentValue >= ultimosSaida[i+1].CurrentValue {
							break
						}
						ultimosValores += ultimosSaida[i].CurrentValue + " | "
						condicaoOK = true
					}
					if condicaoOK {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						order = encerrarOrdem(currentCoin, side, currentValue)
						if config.Development || order == 200 {
							util.Write(ultimosValores, currentCoin)
							util.Write("Valor desceu 3 vezes nas ultimas leituras. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
							util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
							util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
							err = criar_ordem.RemoverCoinDB(currentCoin)
							if err != nil {
								util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
								return
							}
							return
						} else {
							util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
							continue
						}
					}
				}
				if ROI >= takeprofit {
					ultimoMinuto, err := listar_ordens.ListarValorAnterior(currentCoin)
					ultimoMinutoStr := fmt.Sprint(ultimoMinuto)
					if err != nil {
						util.WriteError("Erro ao buscar valor anterior para compararar: ", err, currentCoin)
						continue
					}
					util.Historico(currentCoin, side, started, "tp1", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)

					util.Write("Valor atual: "+currentPriceStr+" Valor 5min atrás: "+ultimoMinutoStr, currentCoin)
					if currentPrice < ultimoMinuto {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						order = encerrarOrdem(currentCoin, side, currentValue)
						if config.Development || order == 200 {
							util.Write("Take Profit atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
							util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
							util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
							err = criar_ordem.RemoverCoinDB(currentCoin)
							if err != nil {
								util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
								return
							}
							return
						} else {
							util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
							continue
						}
					}

				}
				if ROI <= roiMaximo-(stop) { // TODO: ADICIONAR STOP MOVEL NOVAMENTE  -- roiMaximo-(stop)
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					order = encerrarOrdem(currentCoin, side, currentValue)
					if config.Development || order == 200 {
						util.Write("StopLoss atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.Historico(currentCoin, side, started, "sl1", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
						util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
						err = criar_ordem.RemoverCoinDB(currentCoin)
						if err != nil {
							util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
							return
						}
						return
					} else {
						util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
						continue
					}
				}
				if ROI > 0 && now.Sub(start) >= time.Hour {
					order = encerrarOrdem(currentCoin, side, currentValue)
					if config.Development || order == 200 {
						util.Write("Já se passou 1 hora com a operação aberta. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
						util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
						err = criar_ordem.RemoverCoinDB(currentCoin)
						if err != nil {
							util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
							return
						}
						return
					} else {
						util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
						continue
					}

				}

				// Condição que fecha 2x o TakeProfit
				if canClose && ROI < roiMaximo && ROI >= takeprofit {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					order = encerrarOrdem(currentCoin, side, currentValue)
					if config.Development || order == 200 {
						util.Write("Ordem encerrada. 2x Take Profit atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
						util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
						err = criar_ordem.RemoverCoinDB(currentCoin)
						if err != nil {
							util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
							return
						}
						return
					} else {
						util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
						continue
					}
				} else {
					canClose = false
				}
				if ROI >= takeprofit*2+(fee*2) {
					canClose = true
				}

			} else if side == "SELL" && !primeiraExec {
				ROI = (((valueCompradoCoin - currentPrice) / (valueCompradoCoin / alavancagem)) * 100) - (fee * 2)
				if ROI > roiMaximo {
					roiMaximo = ROI
				}
				if ROI > 0 {
					roiTempoRealStr = green(fmt.Sprintf("%.4f", ROI) + "%")
				} else {
					roiTempoRealStr = red(fmt.Sprintf("%.4f", ROI) + "%")
				}
				util.Write("Valor de entrada ("+red("SHORT")+"): "+fmt.Sprint(valueCompradoCoin)+" | "+formattedTime+" | "+currentPriceStr+" | Roi acumulado: "+roiTempoRealStr, currentCoin)

				// Deverá descer 3 consecutivos para fechar.
				if len(ultimosSaida) >= 4 && ultimosSaida[0].CurrentValue > ultimosSaida[1].CurrentValue && now.Sub(start) >= 45*time.Second && (ROI > 0 && ROI < takeprofit) {
					ultimosValores := "| "
					for i := 0; i < 3; i++ {
						condicaoOK = false
						if ultimosSaida[i].CurrentValue <= ultimosSaida[i+1].CurrentValue {
							break
						}
						ultimosValores += ultimosSaida[i].CurrentValue + " | "
						condicaoOK = true
					}
					if condicaoOK {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						order = encerrarOrdem(currentCoin, side, currentValue)
						if config.Development || order == 200 {
							util.Write(ultimosValores, currentCoin)
							util.Write("Valor subiu 3 vezes nas ultimas leituras. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
							util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
							util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
							err = criar_ordem.RemoverCoinDB(currentCoin)
							if err != nil {
								util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
								return
							}
							return
						} else {
							util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
							continue
						}
					}
				}
				if ROI >= takeprofit {
					ultimoMinuto, err := listar_ordens.ListarValorAnterior(currentCoin)
					ultimoMinutoStr := fmt.Sprint(ultimoMinuto)
					if err != nil {
						util.WriteError("Erro ao buscar valor anterior para compararar: ", err, currentCoin)
						continue
					}
					util.Write("Valor atual: "+currentPriceStr+" Valor 5min atrás: "+ultimoMinutoStr, currentCoin)
					if currentPrice > ultimoMinuto {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						order = encerrarOrdem(currentCoin, side, currentValue)
						if config.Development || order == 200 {
							util.Write("Ordem encerrada - Take Profit atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
							util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
							util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

							err = criar_ordem.RemoverCoinDB(currentCoin)
							if err != nil {
								util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
								return
							}
							return
						} else {
							util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
							continue
						}
					}
				}
				if ROI <= roiMaximo-(stop) { // TODO: ADICIONAR STOP MOVEL NOVAMENTE  -- roiMaximo-(stop)
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					order = encerrarOrdem(currentCoin, side, currentValue)
					if config.Development || order == 200 {
						util.Write("Ordem encerrada - StopLoss atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.Historico(currentCoin, side, started, "sl1", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
						util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

						err = criar_ordem.RemoverCoinDB(currentCoin)
						if err != nil {
							util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
							return
						}
						return
					} else {
						util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
						continue
					}
				}
				if ROI > 0 && now.Sub(start) >= time.Hour {
					order = encerrarOrdem(currentCoin, side, currentValue)
					if config.Development || order == 200 {
						util.Write("Já se passou 1 hora com a operação aberta. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
						util.Historico(currentCoin, side, started, "tp1", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
						err = criar_ordem.RemoverCoinDB(currentCoin)
						if err != nil {
							util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
							return
						}
						return
					} else {
						util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
						continue
					}

				}

				// Condição que fecha 2x o TakeProfit
				if canClose && ROI < roiMaximo && ROI >= takeprofit {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					order = encerrarOrdem(currentCoin, side, currentValue)
					if config.Development || order == 200 {
						util.Write("Ordem encerrada - 2x Take Profit atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)
						util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

						err = criar_ordem.RemoverCoinDB(currentCoin)
						if err != nil {
							util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
							return
						}
						return
					} else {
						util.WriteError("Erro ao fechar a ordem, encerre manualmente pela binance: ", err, currentCoin)
						continue
					}
				} else {
					canClose = false
				}
				if ROI >= takeprofit*2+(fee*2) {
					canClose = true
				}
			}
		}
		time.Sleep(forTime)
	}
}

func encerrarOrdem(currentCoin, side string, currentValue float64) int {

	if !config.Development {
		// Valida se a ordem ja foi encerrada para evitar abrir ordem no sentido contrário.
		all, err := listar_ordens.ListarOrdens(currentCoin)
		if err != nil {
			log.Println("Erro ao listar ordens: ", err)
		}
		for _, item := range all {
			if item.PositionSide == "BOTH" {
				if item.EntryPrice == "0.0" {
					util.Write("Ordem ja foi encerrada anteriormente, manual ou por SL Seguro. Finalizando...", currentCoin)
					return 400
				}
			}
		}
	}

	// Encerra a Ordem
	var opposSide string
	if side == "BUY" {
		opposSide = "SELL"
	} else if side == "SELL" {
		opposSide = "BUY"
	}
	order, err := criar_ordem.CriarOrdem(currentCoin, opposSide, fmt.Sprint(currentValue))
	if err != nil {
		_ = criar_ordem.RemoverCoinDB(currentCoin)
		return 0
	}

	// Cancela o StopLoss Seguro que foi criado.
	_, err = criar_ordem.CancelarSLSeguro(currentCoin)
	if err != nil {
		msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
		fmt.Println(msgError)
		util.WriteError(msgError, err, currentCoin)
	}
	return order
}
