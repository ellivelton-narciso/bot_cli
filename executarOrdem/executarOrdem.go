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

func OdemExecucao(currentCoin, posSide, modo string, value, alavancagem, stop, takeprofit, tipoAlerta float64, apiKey, secretKey, user string, enviarDB, historico bool, currValueTelegram float64, currentDateTelegram, urlDisc string) {

	var (
		currentPrice      float64
		err               error
		currentValue      float64
		currentPriceStr   string
		ordemAtiva        bool
		valueCompradoCoin float64
		primeiraExec      bool
		roiAcumulado      float64
		allOrders         []models.CryptoPosition
		ultimosSaida      []models.HistoricoAll
		now               time.Time
		start             time.Time
		ROI               float64
		order             int
		slSeguro          int
		roiAcumuladoStr   string
		roiTempoRealStr   string
		red               func(a ...interface{}) string
		green             func(a ...interface{}) string
		roiMaximo         float64
		started           string
		resposta          string
		//precision           int
		forTime     time.Duration
		priceBuy    float64
		canClose    bool
		side        string
		posSideText string
		newWin      float64
	)

	red = color.New(color.FgHiRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	fee := (0.05 * alavancagem) * 2
	ordemAtiva = false
	primeiraExec = true
	valueCompradoCoin = 0.0
	roiAcumulado = 0.0
	forTime = 900 * time.Millisecond
	roiMaximo = 0
	canClose = false
	newWin = 0

	side = strings.ToUpper(side)
	if posSide == "LONG" {
		side = "BUY"
	}
	if posSide == "SHORT" {
		side = "SELL"
	}
	if enviarDB {
		criar_ordem.EnviarCoinDB(currentCoin, user)
	}
	stop = (stop * alavancagem)
	takeprofit = (takeprofit * alavancagem)

	if !config.Development {
		err = util.DefinirAlavancagem(currentCoin, alavancagem, apiKey, secretKey)
		if err != nil {
			if enviarDB {
				err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
				if err != nil {
					msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
					util.WriteError(msgErr, err, currentCoin)
					fmt.Println(msgErr, err)
					return
				}
			}
			return

		}

		err = util.DefinirMargim(currentCoin, modo, apiKey, secretKey)
		if err != nil {
			if enviarDB {
				err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
				if err != nil {
					msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
					util.WriteError(msgErr, err, currentCoin)
					fmt.Println(msgErr, err)
					return
				}
			}
			return
		}
	}

	/*precision, err = util.GetPrecision(currentCoin)
	if err != nil {
		precision = 0
		util.WriteError("Erro ao buscar precisao para converter a moeda: ", err, currentCoin)
	}*/

	// Encerrar a aplicação graciosamente
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan

		if ordemAtiva {
			msg := fmt.Sprintf("Sinal capturado: %v", sig)

			order := encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
			time.Sleep(5 * time.Second)
			if config.Development || order == 200 {
				util.Write(msg+" . Ordem encerrada: "+currentCoin, currentCoin)
				util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
				err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
				if err != nil {
					msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
					util.WriteError(msgErr, err, currentCoin)
					fmt.Println(msgErr, err)
					return
				}
				return
			} else {
				msgErr := "Erro ao fechar a ordem de " + currentCoin + ", encerre manualmente pela binance: " + fmt.Sprint(order)
				util.WriteError(msgErr, err, currentCoin)
				fmt.Println(msgErr)
				return
			}
		}

		os.Exit(0)
	}()

	for {
		if primeiraExec {
			time.Sleep(500 * time.Millisecond)
			primeiraExec = false
			if !config.Development {
				allOrders, err = listar_ordens.ListarOrdens(currentCoin, apiKey, secretKey)
				if err != nil {
					primeiraExec = true
					util.WriteError("Erro ao listar ordens: ", err, currentCoin)
					continue
				}
				for _, item := range allOrders {
					entryPriceFloat, _ := strconv.ParseFloat(item.EntryPrice, 64)
					if entryPriceFloat > 0 {
						util.Write("Ja possui ordem ativa.", currentCoin)
						util.RegistroLogs(currentCoin, side, currentDateTelegram, 2, currValueTelegram)
						ordemAtiva = true
					}
				}
				if ordemAtiva {
					return
				}
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
				currentValue, priceBuy = util.ConvertBaseCoin(currentCoin, value*alavancagem, apiKey)
				if currentValue == 0 || priceBuy == 0 {
					util.Write("Valor atual ou Preço de compra é igual a 0", currentCoin)
					err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
					if err != nil {
						util.Write("Erro ao remover "+currentCoin+" do banco de dados", currentCoin)
						return
					}
					return
				}
				valueCompradoCoin = priceBuy
				order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue), posSide, apiKey, secretKey)
				if err != nil {
					log.Println("Erro ao dar entrada m LONG: ", err)
					err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
					if err != nil {
						log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
					}

					if !config.Development {
						allOrders, err = listar_ordens.ListarOrdens(currentCoin, apiKey, secretKey)
						if err != nil {
							util.WriteError("Erro ao listar ordens: ", err, currentCoin)
							return
						}
						for _, item := range allOrders {
							entryPriceFloat, _ := strconv.ParseFloat(item.EntryPrice, 64)
							if entryPriceFloat > 0 {
								util.Write("Ja possui ordem ativa.", currentCoin)

							}
						}
						return
					}
					return
				}
				if config.Development || order == 200 {
					util.Write("Entrada em LONG: "+fmt.Sprint(valueCompradoCoin)+", TP: "+fmt.Sprintf("%.4f", takeprofit)+", SL: "+fmt.Sprintf("%.4f", stop)+"ALERTA: "+fmt.Sprint(tipoAlerta), currentCoin)
					ordemAtiva = true
					allOrders, err = listar_ordens.ListarOrdens(currentCoin, apiKey, secretKey)
					if err != nil {
						log.Println("Erro ao listar ordens: ", err)
					}
					for _, item := range allOrders {
						if item.PositionSide == "LONG" {
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
					util.Historico(currentCoin, "BUY", started, "tp1", currentDateTelegram, valueCompradoCoin, currValueTelegram, valueCompradoCoin, ROI, historico)
					forTime = 5 * time.Second
					precisionSymbol, err := util.GetPrecisionSymbol(currentCoin, apiKey)
					q := valueCompradoCoin * (1 - (((stop / alavancagem) / 100) * 1.1))
					stopSeguro := math.Round(q*math.Pow(10, float64(precisionSymbol))) / math.Pow(10, float64(precisionSymbol))
					slSeguro, resposta, err = criar_ordem.CriarSLSeguro(currentCoin, "SELL", fmt.Sprint(stopSeguro), posSide, apiKey, secretKey)
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
					util.RegistroLogs(currentCoin, side, currentDateTelegram, 4, currValueTelegram)
					ordemAtiva = false
					err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
					if err != nil {
						util.Write("Erro ao remover "+currentCoin+" do banco de dados", currentCoin)
					}
					return
				}

			} else if side == "SELL" {
				start = time.Now()
				timeValue := time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
				started = timeValue.Format("2006-01-02 15:04:05")
				currentValue, priceBuy = util.ConvertBaseCoin(currentCoin, value*alavancagem, apiKey)
				if currentValue == 0 || priceBuy == 0 {
					util.Write("Valor atual ou Preço de compra é igual a 0", currentCoin)
					err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
					if err != nil {
						util.Write("Erro ao remover "+currentCoin+" do banco de dados", currentCoin)
						return
					}
					return
				}
				valueCompradoCoin = priceBuy
				order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue), posSide, apiKey, secretKey)
				if err != nil {
					log.Println("Erro ao dar entrada em SHORT: ", err)
					if !config.Development {
						allOrders, err = listar_ordens.ListarOrdens(currentCoin, apiKey, secretKey)
						if err != nil {
							util.WriteError("Erro ao listar ordens: ", err, currentCoin)
							err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
							if err != nil {
								log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
								return
							}
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
							err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
							if err != nil {
								log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
								return
							}
							return
						}
						continue
					} else {
						err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
						if err != nil {
							log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
							return
						}
						return
					}
				}
				if config.Development || order == 200 {
					util.Write("Entrada em SHORT: "+fmt.Sprint(valueCompradoCoin)+", TP: "+fmt.Sprintf("%.4f", takeprofit)+", SL: "+fmt.Sprintf("%.4f", stop)+"ALERTA: "+fmt.Sprint(tipoAlerta), currentCoin)
					ordemAtiva = true
					allOrders, err = listar_ordens.ListarOrdens(currentCoin, apiKey, secretKey)
					if err != nil {
						log.Println("Erro ao listar ordens: ", err)
					}
					for _, item := range allOrders {
						if item.PositionSide == "SHORT" {
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
					util.Historico(currentCoin, "SELL", started, "tp1", currentDateTelegram, valueCompradoCoin, currValueTelegram, valueCompradoCoin, ROI, historico)
					forTime = 5 * time.Second
					precisionSymbol, err := util.GetPrecisionSymbol(currentCoin, apiKey)
					q := valueCompradoCoin * (1 + (((stop / alavancagem) / 100) * 1.1))
					stopSeguro := math.Round(q*math.Pow(10, float64(precisionSymbol))) / math.Pow(10, float64(precisionSymbol))
					slSeguro, resposta, err = criar_ordem.CriarSLSeguro(currentCoin, "BUY", fmt.Sprint(stopSeguro), posSide, apiKey, secretKey)
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
					util.Write("A ordem de SHORT não foi totalmente completada.", currentCoin)
					util.RegistroLogs(currentCoin, side, currentDateTelegram, 4, currValueTelegram)
					ordemAtiva = false
					err = criar_ordem.RemoverCoinDB(currentCoin, user, 2*time.Second)
					if err != nil {
						util.Write("Erro ao remover "+currentCoin+" do banco de dados", currentCoin)
					}
				}

			}
		} else { // Já possui uma ordem ativa
			now = time.Now()
			timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
			formattedTime := timeValue.Format("2006-01-02 15:04:05")
			if posSide == "LONG" {
				posSideText = green(posSide)
				ROI = (((currentPrice - valueCompradoCoin) / (valueCompradoCoin / alavancagem)) * 100) - (fee)
			} else if posSide == "SHORT" {
				posSideText = red(posSide)
				ROI = (((valueCompradoCoin - currentPrice) / (valueCompradoCoin / alavancagem)) * 100) - (fee)
			} else {
				posSideText = red("ERRO AO OBTER DIREÇÃO")
			}
			if ROI > roiMaximo {
				roiMaximo = ROI
			}
			if ROI > 0 {
				roiTempoRealStr = green(fmt.Sprintf("%.4f", ROI) + "%")
			} else {
				roiTempoRealStr = red(fmt.Sprintf("%.4f", ROI) + "%")
			}
			util.Write("Valor de entrada ("+posSideText+"): "+fmt.Sprint(valueCompradoCoin)+" | "+formattedTime+" | "+fmt.Sprint(currentPrice)+" | Roi acumulado: "+roiTempoRealStr, currentCoin)

			if canClose && ROI < takeprofit {
				roiAcumulado = roiAcumulado + ROI
				if roiAcumulado > 0 {
					roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
				} else {
					roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
				}
				err := util.SendMessageToDiscord("Take Profit atingido. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado), urlDisc)
				if err != nil {
					util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
				}
				order = encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
				if config.Development || order == 200 {
					util.Write("Ordem encerrada. Take Profit atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
					util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI, historico)
					util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
					err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
					if err != nil {
						msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
						util.WriteError(msgErr, err, currentCoin)
						fmt.Println(msgErr, err)
						return
					}
					return
				} else {
					util.Write("Erro ao fechar a ordem, encerre manualmente pela binance: "+fmt.Sprint(order), currentCoin)
					_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
					if err != nil {
						msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
						fmt.Println(msgError)
						util.WriteError(msgError, err, currentCoin)
					}
					err = criar_ordem.RemoverCoinDB(currentCoin, user, 3*time.Minute)
					if err != nil {
						util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
						return
					}
					return
				}
			}

			if side == "BUY" && !primeiraExec {
				// Deverá descer 3 consecutivos para fechar.
				if ROI >= takeprofit || (ROI > 0 && now.Sub(start) >= 2*time.Minute) || canClose {
					canClose = true
					ultimoMinuto, err := listar_ordens.ListarValorAnterior(currentCoin, "1")
					ultimoMinutoStr := fmt.Sprint(ultimoMinuto)
					if err != nil {
						util.WriteError("Erro ao buscar valor anterior para compararar: ", err, currentCoin)
						continue
					}

					if newWin == 0 || currentPrice > newWin {
						newWin = currentPrice
					}

					util.Write("Valor atual: "+currentPriceStr+" Valor 5min atrás: "+ultimoMinutoStr, currentCoin)
					if currentPrice < ultimoMinuto || currentPrice < newWin {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						err := util.SendMessageToDiscord("Take Profit atingido. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado), urlDisc)
						if err != nil {
							util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
						}
						order = encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
						if config.Development || order == 200 {
							util.Write("Take Profit atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
							util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI, historico)
							util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
							err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
							if err != nil {
								msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
								util.WriteError(msgErr, err, currentCoin)
								fmt.Println(msgErr, err)
								return
							}
							return
						} else {
							util.Write("Erro ao fechar a ordem, encerre manualmente pela binance: "+fmt.Sprint(order), currentCoin)
							_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
							if err != nil {
								msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
								fmt.Println(msgError)
								util.WriteError(msgError, err, currentCoin)
							}
							err = criar_ordem.RemoverCoinDB(currentCoin, user, 3*time.Minute)
							if err != nil {
								log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
								return
							}
							return
						}
					}

				}
				if ROI <= 0-(stop) || (ROI < 0 && now.Sub(start) >= 2*time.Minute) {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					err := util.SendMessageToDiscord("StopLoss atingido. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado), urlDisc)
					if err != nil {
						util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
					}
					order = encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
					if config.Development || order == 200 {
						util.Write("StopLoss atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.Historico(currentCoin, side, started, "sl1", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI, historico)
						util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
						err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
						if err != nil {
							msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
							util.WriteError(msgErr, err, currentCoin)
							fmt.Println(msgErr, err)
							return
						}
						return
					} else {
						util.Write("Erro ao fechar a ordem, encerre manualmente pela binance: "+fmt.Sprint(order), currentCoin)
						_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
						if err != nil {
							msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
							fmt.Println(msgError)
							util.WriteError(msgError, err, currentCoin)
						}
						return
					}
				}
				if (ROI >= takeprofit/4 && now.Sub(start) >= 2*time.Hour) || (ROI >= takeprofit/6 && now.Sub(start) >= 3*time.Hour) || (ROI > 0 && now.Sub(start) >= 6*time.Hour) {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					err := util.SendMessageToDiscord("Já se passou muito tempo com a operação aberta. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado), urlDisc)
					if err != nil {
						util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
					}
					order = encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
					if config.Development || order == 200 {
						util.Write("Já se passou muito tempo com a operação aberta. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI, historico)
						util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
						err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
						if err != nil {
							msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
							util.WriteError(msgErr, err, currentCoin)
							fmt.Println(msgErr, err)
							return
						}
						return
					} else {
						util.Write("Erro ao fechar a ordem, encerre manualmente pela binance: "+fmt.Sprint(order), currentCoin)
						_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
						if err != nil {
							msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
							fmt.Println(msgError)
							util.WriteError(msgError, err, currentCoin)
						}
						err = criar_ordem.RemoverCoinDB(currentCoin, user, 3*time.Minute)
						if err != nil {
							log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
							return
						}
						return
					}
				}
				if ROI <= 0-(stop/2) && now.Sub(start) >= 20*time.Minute {
					var condicaoLossOK bool
					condicaoLossOK = false

					volume1h, err := listar_ordens.GetVolumeData(currentCoin, "1h", 1)
					if err != nil {
						util.WriteError("Erro ao buscar dados do volume, ", err, currentCoin)
					}
					volume5m, err := listar_ordens.GetVolumeData(currentCoin, "5m", 1)
					if err != nil {
						util.WriteError("Erro ao buscar dados do volume, ", err, currentCoin)
					}
					util.Write("Ratio Volume 1h: "+fmt.Sprintf("%.4f", volume1h[0].RatioVolume)+", Ratio Volume 5m: "+fmt.Sprintf("%.4f", volume5m[0].RatioVolume), currentCoin)

					// Volume de venda de 1h 50% superior que o de compra e volume de venda 5m 100% superior que o de compra
					if volume1h[0].RatioVolume <= 1/1.5 && volume5m[0].RatioVolume <= 1/2 {
						condicaoLossOK = true
						util.Write("Volume desfavorável para continuar. Encerrando para evitar perdas maiores.", currentCoin)
						err := util.SendMessageToDiscord("Volume desfavorável para continuar. Encerrando para evitar perdas maiores.", urlDisc)
						if err != nil {
							util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
						}

					}
					if roiMaximo <= 0 {
						condicaoLossOK = true
						util.Write("Possível tendencia contrária identificado. ", currentCoin)
						err := util.SendMessageToDiscord("Possível tendencia contrária identificado.", urlDisc)
						if err != nil {
							util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
						}
					}
					if condicaoLossOK {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						err := util.SendMessageToDiscord("Ordem encerrada. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado), urlDisc)
						if err != nil {
							util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
						}
						order = encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
						if config.Development || order == 200 {
							util.Write("Ordem encerrada. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
							util.Historico(currentCoin, side, started, "sl1", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI, historico)
							util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
							err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
							if err != nil {
								msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
								util.WriteError(msgErr, err, currentCoin)
								fmt.Println(msgErr, err)
								return
							}
							return
						} else {
							util.Write("Erro ao fechar a ordem, encerre manualmente pela binance: "+fmt.Sprint(order), currentCoin)
							_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
							if err != nil {
								msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
								fmt.Println(msgError)
								util.WriteError(msgError, err, currentCoin)
							}

							err = criar_ordem.RemoverCoinDB(currentCoin, user, 3*time.Minute)
							if err != nil {
								util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
								return
							}
							return
						}
					}

				}

			} else if side == "SELL" && !primeiraExec {
				if ROI >= takeprofit || (ROI > 0 && now.Sub(start) >= 2*time.Minute) || canClose {
					canClose = true
					ultimoMinuto, err := listar_ordens.ListarValorAnterior(currentCoin, "1")
					ultimoMinutoStr := fmt.Sprint(ultimoMinuto)
					if err != nil {
						util.WriteError("Erro ao buscar valor anterior para compararar: ", err, currentCoin)
						continue
					}
					util.Write("Valor atual: "+currentPriceStr+" Valor 5min atrás: "+ultimoMinutoStr, currentCoin)

					if newWin == 0 || currentPrice < newWin {
						newWin = currentPrice
					}

					if currentPrice > ultimoMinuto || currentPrice > newWin {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						err := util.SendMessageToDiscord("Take Profit atingido. Roi acumulado:  "+fmt.Sprintf("%.4f", roiAcumulado), urlDisc)
						if err != nil {
							util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
						}
						order = encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
						if config.Development || order == 200 {
							util.Write("Ordem encerrada - Take Profit atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
							util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI, historico)
							util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
							err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
							if err != nil {
								msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
								util.WriteError(msgErr, err, currentCoin)
								fmt.Println(msgErr, err)
								return
							}

							return
						} else {
							util.Write("Erro ao fechar a ordem, encerre manualmente pela binance: "+fmt.Sprint(order), currentCoin)
							_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
							if err != nil {
								msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
								fmt.Println(msgError)
								util.WriteError(msgError, err, currentCoin)
							}

							err = criar_ordem.RemoverCoinDB(currentCoin, user, 3*time.Minute)
							if err != nil {
								util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
								return
							}
							return
						}
					}
				}
				if ROI <= 0-(stop) || (ROI < 0 && now.Sub(start) >= 2*time.Minute) { // TODO: ADICIONAR STOP MOVEL NOVAMENTE  -- roiMaximo-(stop)
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					err := util.SendMessageToDiscord("StopLoss atingido. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado), urlDisc)
					if err != nil {
						util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
					}
					order = encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
					if config.Development || order == 200 {
						util.Write("Ordem encerrada - StopLoss atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.Historico(currentCoin, side, started, "sl1", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI, historico)
						util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
						err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
						if err != nil {
							msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
							util.WriteError(msgErr, err, currentCoin)
							fmt.Println(msgErr, err)
							return
						}
						return
					} else {
						util.Write("Erro ao fechar a ordem, encerre manualmente pela binance: "+fmt.Sprint(order), currentCoin)
						_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
						if err != nil {
							msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
							fmt.Println(msgError)
							util.WriteError(msgError, err, currentCoin)
						}

						err = criar_ordem.RemoverCoinDB(currentCoin, user, 3*time.Minute)
						if err != nil {
							util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
							return
						}
						return
					}
				}
				if (ROI >= takeprofit/4 && now.Sub(start) >= 2*time.Hour) || (ROI >= takeprofit/6 && now.Sub(start) >= 3*time.Hour) || (ROI > 0 && now.Sub(start) >= 6*time.Hour) {
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					err := util.SendMessageToDiscord("Já se passou muito tempo com a operação aberta. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado), urlDisc)
					if err != nil {
						util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
					}
					order = encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
					if config.Development || order == 200 {
						util.Write("Já se passou muito tempo com a operação aberta. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
						util.Historico(currentCoin, side, started, "tp2", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI, historico)
						util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
						err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
						if err != nil {
							msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
							util.WriteError(msgErr, err, currentCoin)
							fmt.Println(msgErr, err)
							return
						}
						return
					} else {
						util.Write("Erro ao fechar a ordem, encerre manualmente pela binance: "+fmt.Sprint(order), currentCoin)
						_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
						if err != nil {
							msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
							fmt.Println(msgError)
							util.WriteError(msgError, err, currentCoin)
						}
						err = criar_ordem.RemoverCoinDB(currentCoin, user, 3*time.Minute)
						if err != nil {
							log.Println("Não foi possível remover ", currentCoin, " da tabela bots")
							return
						}
						return
					}
				}
				if ROI <= 0-(stop/2) && now.Sub(start) >= 20*time.Minute {
					var condicaoLossOK bool
					condicaoLossOK = false

					volume1h, err := listar_ordens.GetVolumeData(currentCoin, "1h", 1)
					if err != nil {
						util.WriteError("Erro ao buscar dados do volume, ", err, currentCoin)
					}
					time.Sleep(1 * time.Second)
					volume5m, err := listar_ordens.GetVolumeData(currentCoin, "5m", 1)
					if err != nil {
						util.WriteError("Erro ao buscar dados do volume, ", err, currentCoin)
					}
					util.Write("Ratio Volume 1h: "+fmt.Sprintf("%.4f", volume1h[0].RatioVolume)+", Ratio Volume 5m: "+fmt.Sprintf("%.4f", volume5m[0].RatioVolume), currentCoin)

					// Volume de compra de 1h 50% superior que o de venda e volume de compra de 5m 100% superior que o de venda
					if volume1h[0].RatioVolume >= 1.5 && volume5m[0].RatioVolume >= 2 {
						condicaoLossOK = true
						util.Write("Volume desfavorável para continuar. Encerrando para evitar perdas maiores.", currentCoin)
						err := util.SendMessageToDiscord("Volume desfavorável para continuar. Encerrando para evitar perdas maiores.", urlDisc)
						if err != nil {
							util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
						}
					}
					if roiMaximo <= 0 {
						condicaoLossOK = true
						util.Write("Possível tendencia contrária identificado. ", currentCoin)
						err := util.SendMessageToDiscord("Possível tendencia contrária identificado. ", urlDisc)
						if err != nil {
							util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
						}
					}
					if condicaoLossOK {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						err := util.SendMessageToDiscord("Ordem encerrada. Roi acumulado: "+fmt.Sprintf("%.4f", roiAcumulado), urlDisc)
						if err != nil {
							util.WriteError("Erro ao enviar mensagem no discord.", err, currentCoin)
						}
						order = encerrarOrdem(currentCoin, side, posSide, currentValue, apiKey, secretKey, user, enviarDB)
						if config.Development || order == 200 {
							util.Write("Ordem encerrada. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
							util.Historico(currentCoin, side, started, "sl1", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI, historico)
							util.EncerrarHistorico(currentCoin, side, currentDateTelegram, currentPrice, ROI/alavancagem, historico)
							err = criar_ordem.RemoverCoinDB(currentCoin, user, 5*time.Minute)
							if err != nil {
								msgErr := "Erro ao remover " + currentCoin + " do banco de dados: "
								util.WriteError(msgErr, err, currentCoin)
								fmt.Println(msgErr, err)
								return
							}
							return
						} else {
							util.Write("Erro ao fechar a ordem, encerre manualmente pela binance: "+fmt.Sprint(order), currentCoin)
							_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
							if err != nil {
								msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
								fmt.Println(msgError)
								util.WriteError(msgError, err, currentCoin)
							}

							err = criar_ordem.RemoverCoinDB(currentCoin, user, 3*time.Minute)
							if err != nil {
								util.WriteError("Erro ao remover ativo do banco de dados: ", err, currentCoin)
								return
							}
							return
						}
					}

				}
			}
		}
		time.Sleep(forTime)
	}
}

func encerrarOrdem(currentCoin, side, posSide string, currentValue float64, apiKey, secretKey, user string, enviarDB bool) int {

	if !config.Development {
		// Valida se a ordem ja foi encerrada para evitar abrir ordem no sentido contrário.
		all, err := listar_ordens.ListarOrdens(currentCoin, apiKey, secretKey)
		if err != nil {
			log.Println("Erro ao listar ordens: ", err)
		}
		for _, item := range all {
			if item.PositionSide == posSide {
				if item.EntryPrice == "0.0" {
					util.Write("Ordem ja foi encerrada anteriormente, manual ou por SL Seguro. Finalizando...", currentCoin)
					return 400
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
		order, err := criar_ordem.CriarOrdem(currentCoin, opposSide, fmt.Sprint(currentValue), posSide, apiKey, secretKey)
		if err != nil {
			if enviarDB {
				err = criar_ordem.RemoverCoinDB(currentCoin, user, 3*time.Minute)
				if err != nil {
					util.WriteError("Erro ao remover coin da Tabela, ", err, currentCoin)
				}
			}
			return 0
		}
		// Cancela o StopLoss Seguro que foi criado.
		_, err = criar_ordem.CancelarSLSeguro(currentCoin, apiKey, secretKey)
		if err != nil {
			msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
			fmt.Println(msgError)
			util.WriteError(msgError, err, currentCoin)
		}
		return order
	}
	return 0
}
