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
	"strconv"
	"strings"
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
		ultimosSaida        []models.PriceResponse
		now                 time.Time
		start               time.Time
		ROI                 float64
		order               int
		roiAcumuladoStr     string
		roiTempoRealStr     string
		red                 func(a ...interface{}) string
		green               func(a ...interface{}) string
		roiMaximo           float64
		started             string
		currValueTelegram   float64
		currentDateTelegram string
	)

	red = color.New(color.FgHiRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	fee := 0.05 * alavancagem
	ordemAtiva = false
	primeiraExec = true
	valueCompradoCoin = 0.0
	roiAcumulado = 0.0

	roiMaximo = 0

	side = strings.ToUpper(side)
	if side == "LONG" {
		side = "BUY"
	}
	if side == "SHORT" {
		side = "SELL"
	}
	tg := util.BuscarValoresTelegram(currentCoin)
	if len(tg) == 0 {
		return
	}
	currValueTelegram = tg[0].CurrValue
	currentDateTelegram = tg[0].HistDate.Format("2006-01-02 15:04:05")

	util.DefinirAlavancagem(currentCoin, alavancagem)
	util.DefinirMargim(currentCoin, "ISOLATED")
	criar_ordem.EnviarCoinDB(currentCoin)

	for {
		if primeiraExec {
			time.Sleep(2 * time.Second)
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
		}

		ultimosSaida = listar_ordens.ListarUltimosValores(currentCoin, 1)
		if len(ultimosSaida) == 0 {
			util.Write("Erro, array vazio. "+fmt.Sprint(len(ultimosSaida))+"", currentCoin)
			continue
		}
		currentPrice, err = strconv.ParseFloat(ultimosSaida[0].Price, 64)

		if err != nil {
			util.WriteError("Erro no array currentPrice: ", err, currentCoin)
			continue
		}
		currentPriceStr = fmt.Sprint(currentPrice)
		if !ordemAtiva { // Não tem ordem ainda

			if side == "BUY" {
				currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
				valueCompradoCoin = currentPrice
				start = time.Now()
				timeValue := time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
				started = timeValue.Format("2006-01-02 15:04:05")
				order, err = criar_ordem.CriarOrdem(currentCoin, "BUY", fmt.Sprint(currentValue))
				if err != nil {
					log.Println("Erro ao dar entrada m LONG: ", err)
				}
				if config.Development || order == 200 {
					util.Write("Entrada em LONG: "+currentPriceStr, currentCoin)
					ordemAtiva = true
					allOrders, err = listar_ordens.ListarOrdens(currentCoin)
					if err != nil {
						log.Println("Erro ao listar ordens: ", err)
					}
					for _, item := range allOrders {
						if item.PositionSide == "BUY" {
							valueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
							started_timestamp := item.UpdateTime
							timeStarted := time.Unix(0, started_timestamp*int64(time.Millisecond))
							started = timeStarted.Format("2006-01-02 15:04:05")
							if err != nil {
								log.Println("Erro ao buscar valor de entrada: ", err)
							}
						}
					}
					util.Historico(currentCoin, "BUY", started, "", currentDateTelegram, valueCompradoCoin, currValueTelegram, valueCompradoCoin, ROI)
				} else {
					util.Write("A ordem de LONG não foi totalmente completada.", currentCoin)
					ordemAtiva = false

				}
			} else if side == "SELL" {
				currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
				valueCompradoCoin = currentPrice
				start = time.Now()
				timeValue := time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
				started = timeValue.Format("2006-01-02 15:04:05")
				order, err = criar_ordem.CriarOrdem(currentCoin, "SELL", fmt.Sprint(currentValue))
				if err != nil {
					log.Println("Erro ao criar conta: ", err)
				}
				if config.Development || order == 200 {
					util.Write("Entrada em SHORT: "+currentPriceStr, currentCoin)
					ordemAtiva = true
					allOrders, err = listar_ordens.ListarOrdens(currentCoin)
					if err != nil {
						log.Println("Erro ao listar ordens: ", err)
					}
					for _, item := range allOrders {
						if item.PositionSide == "SELL" {
							valueCompradoCoin, err = strconv.ParseFloat(item.EntryPrice, 64)
							started_timestamp := item.UpdateTime
							timeStarted := time.Unix(0, started_timestamp*int64(time.Millisecond))
							started = timeStarted.Format("2006-01-02 15:04:05")
							if err != nil {
								log.Println("Erro ao buscar valor de entrada: ", err)
							}
						}
					}
					util.Historico(currentCoin, "SELL", started, "", currentDateTelegram, valueCompradoCoin, currValueTelegram, valueCompradoCoin, ROI)
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

				} else if ROI <= -(stop) { // TODO: ADICIONAR STOP MOVEL NOVAMENTE  -- roiMaximo-(stop)
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
				} else if ROI >= takeprofit {
					ultimoMinuto := listar_ordens.ListarValorUltimoMinuto(currentCoin)
					if len(ultimoMinuto) == 0 {
						util.Write("Tamanho de variável ultimo minuto é 0", currentCoin)
						continue
					}
					valorUltimoMinuto, _ := strconv.ParseFloat(ultimoMinuto[0].Price, 64)
					util.Historico(currentCoin, side, started, "tp1", currentDateTelegram, currentPrice, currValueTelegram, valueCompradoCoin, ROI)

					if currentPrice < valorUltimoMinuto {
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

				} else if ROI <= -(stop) { // TODO: ADICIONAR STOP MOVEL NOVAMENTE  -- roiMaximo-(stop)
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
				} else if ROI >= takeprofit {
					ultimoMinuto := listar_ordens.ListarValorUltimoMinuto(currentCoin)
					if len(ultimoMinuto) == 0 {
						util.Write("Tamanho de variável ultimo minuto é 0", currentCoin)
						continue
					}
					valorUltimoMinuto, _ := strconv.ParseFloat(ultimoMinuto[0].Price, 64)

					if currentPrice > valorUltimoMinuto {
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
			}
		}
		time.Sleep(900 * time.Millisecond)
	}
}

func encerrarOrdem(currentCoin, side string, currentValue float64) int {
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
	return order
}
