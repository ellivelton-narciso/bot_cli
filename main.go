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
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	currentCoin       string
	side              string
	value             float64
	alavancagem       float64
	currentPrice      float64
	err               error
	currentValue      float64
	currentPriceStr   string
	ordemAtiva        bool
	valueCompradoCoin float64
	primeiraExec      bool
	roiAcumulado      float64
	stop              float64
	allOrders         []models.CryptoPosition
	ultimosSaida      []models.PriceResponse
	now               time.Time
	start             time.Time
	ROI               float64
	order             int
	roiAcumuladoStr   string
	roiTempoRealStr   string
	red               func(a ...interface{}) string
	green             func(a ...interface{}) string
	roiMaximo         float64
	started           string
	takeprofit        float64
)

func main() {
	database.DBCon()
	config.ReadFile()
	app := cli.NewApp()
	setupCommands(app)
	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	if config.ApiKey == "" || config.SecretKey == "" || config.BaseURL == "" {
		log.Panic("Arquivo user.json incompleto.")
	}

	red = color.New(color.FgHiRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	ordemAtiva = false
	primeiraExec = true
	valueCompradoCoin = 0.0
	roiAcumulado = 0.0
	fee := 0.05 * alavancagem
	roiMaximo = 0

	side = strings.ToUpper(side)
	if side == "LONG" {
		side = "BUY"
	}
	if side == "SHORT" {
		side = "SELL"
	}

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
				err = criar_ordem.RemoverCoinDB(currentCoin)
				if err != nil {
					primeiraExec = true
					util.Write("Erro ao remover coin da database", currentCoin)
					continue
				}
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
				o := comprarBuy()
				if config.Development {
					ordemAtiva = true
				} else {
					ordemAtiva = o == 200
				}
			} else if side == "SELL" {
				o := comprarSell()
				if config.Development {
					ordemAtiva = true
				} else {
					ordemAtiva = o == 200
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
					util.Write("Já se passou 1 hora com a operação aberta. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
					util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
					encerrarOrdem()

					util.Historico(currentCoin, side, started, "tp2", currentPrice, valueCompradoCoin, ROI)
					util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
					return

				} else if ROI <= -(stop) { // TODO: ADICIONAR STOP MOVEL NOVAMENTE  -- roiMaximo-(stop)
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("StopLoss atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)

					util.Historico(currentCoin, side, started, "sl1", currentPrice, valueCompradoCoin, ROI)
					util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

					encerrarOrdem()
					return
				} else if ROI >= takeprofit {
					ultimoMinuto := listar_ordens.ListarValorUltimoMinuto(currentCoin)
					if len(ultimoMinuto) == 0 {
						util.Write("Tamanho de variável ultimo minuto é 0", currentCoin)
						continue
					}
					valorUltimoMinuto, _ := strconv.ParseFloat(ultimoMinuto[0].Price, 64)
					util.Historico(currentCoin, side, started, "tp1", currentPrice, valueCompradoCoin, ROI)

					if currentPrice < valorUltimoMinuto {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						util.Write("Take Profit atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)

						util.Historico(currentCoin, side, started, "tp2", currentPrice, valueCompradoCoin, ROI)
						util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

						encerrarOrdem()
						return
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
					util.Write("Já se passou 1 hora com a operação aberta. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)
					util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

					util.Historico(currentCoin, side, started, "tp1", currentPrice, valueCompradoCoin, ROI)
					util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

					encerrarOrdem()
					return

				} else if ROI <= -(stop) { // TODO: ADICIONAR STOP MOVEL NOVAMENTE  -- roiMaximo-(stop)
					roiAcumulado = roiAcumulado + ROI
					if roiAcumulado > 0 {
						roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					} else {
						roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
					}
					util.Write("Ordem encerrada - StopLoss atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)

					util.Historico(currentCoin, side, started, "sl1", currentPrice, valueCompradoCoin, ROI)
					util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

					encerrarOrdem()
					return
				} else if ROI >= takeprofit {
					ultimoMinuto := listar_ordens.ListarValorUltimoMinuto(currentCoin)
					if len(ultimoMinuto) == 0 {
						util.Write("Tamanho de variável ultimo minuto é 0", currentCoin)
						continue
					}
					valorUltimoMinuto, _ := strconv.ParseFloat(ultimoMinuto[0].Price, 64)

					util.Historico(currentCoin, side, started, "tp1", currentPrice, valueCompradoCoin, ROI)
					util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

					if currentPrice > valorUltimoMinuto {
						roiAcumulado = roiAcumulado + ROI
						if roiAcumulado > 0 {
							roiAcumuladoStr = green(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						} else {
							roiAcumuladoStr = red(fmt.Sprintf("%.4f", roiAcumulado) + "%")
						}
						util.Write("Ordem encerrada - Take Profit atingido. Roi acumulado: "+roiAcumuladoStr+"\n\n", currentCoin)

						util.Historico(currentCoin, side, started, "tp2", currentPrice, valueCompradoCoin, ROI)
						util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)

						encerrarOrdem()
						return
					}
				}
			}
		}
		time.Sleep(900 * time.Millisecond)
	}
}

func comprarBuy() int {
	currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
	valueCompradoCoin = currentPrice
	start = time.Now()
	timeValue := time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
	started = timeValue.Format("2006-01-02 15:04:05")
	side = "BUY"
	util.Write("Entrada em LONG: "+currentPriceStr, currentCoin)
	order, err = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue))
	if err != nil {
		log.Println("Erro ao criar conta: ", err)
	}
	if config.Development || order == 200 {
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
		util.Historico(currentCoin, side, started, "", valueCompradoCoin, valueCompradoCoin, ROI)
	} else {
		side = ""
		util.Write("A ordem de LONG não foi totalmente completada.", currentCoin)
		ordemAtiva = false
		os.Exit(1)

	}
	return order

}

func comprarSell() int {
	currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
	valueCompradoCoin = currentPrice
	start = time.Now()
	timeValue := time.Unix(0, start.UnixMilli()*int64(time.Millisecond))
	started = timeValue.Format("2006-01-02 15:04:05")
	util.Write("Entrada em SHORT: "+currentPriceStr, currentCoin)
	side = "SELL"
	order, err = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue))
	if err != nil {
		log.Println("Erro ao criar conta: ", err)
	}
	if config.Development || order == 200 {

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
		util.Historico(currentCoin, side, started, "", valueCompradoCoin, valueCompradoCoin, ROI)
	} else {
		side = ""
		util.Write("A ordem de SHORT não foi totalmente completada. Irei voltar a buscar novas oportunidades. Pode a qualquer momento digitar SELL para entrar em SHORT.", currentCoin)
		ordemAtiva = false
	}
	return order

}

func setupCommands(app *cli.App) {
	app.Commands = []*cli.Command{
		{
			Name:  "config",
			Usage: "Configurar parâmetros de inicialização",
			Action: func(c *cli.Context) error {
				currentCoin = c.String("coin")
				value = c.Float64("value")
				stop = c.Float64("stop")
				side = c.String("side")
				alavancagem = c.Float64("leverage")
				takeprofit = c.Float64("takeprofit")

				return nil
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "coin",
					Usage:   "Moeda (ex: BTC)",
					Aliases: []string{"c"},
				},
				&cli.Float64Flag{
					Name:    "value",
					Value:   10,
					Usage:   "Quantidade em moeda base",
					Aliases: []string{"v"},
				},
				&cli.Float64Flag{
					Name:    "stop",
					Value:   1.5,
					Usage:   "Definir o Stop Loss da Ordem.",
					Aliases: []string{"s"},
				},
				&cli.StringFlag{
					Name:    "side",
					Usage:   "Direção da ordem (BUY, SELL, LONG, SHORT)",
					Aliases: []string{"i"},
				},
				&cli.Float64Flag{
					Name:    "leverage",
					Value:   1,
					Usage:   "Definir a alavancagem.",
					Aliases: []string{"l"},
				},
				&cli.Float64Flag{
					Name:    "takeprofit",
					Value:   1.5,
					Usage:   "Definir o Take Profit",
					Aliases: []string{"t"},
				},
			},
		},
	}
}

func encerrarOrdem() {
	var opposSide string
	if side == "BUY" {
		opposSide = "SELL"
	} else if side == "SELL" {
		opposSide = "BUY"
	}
	order, err = criar_ordem.CriarOrdem(currentCoin, opposSide, fmt.Sprint(currentValue))
	if err != nil {
		log.Println("Erro ao fechar a ordem, encerre manualmente pela binance: ", err)
		util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
		_ = criar_ordem.RemoverCoinDB(currentCoin)
		return
	}
	if config.Development || order == 200 {
		ordemAtiva = false
		util.EncerrarHistorico(currentCoin, side, started, currentPrice, ROI)
		_ = criar_ordem.RemoverCoinDB(currentCoin)
		return
	} else {
		util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin)
		ordemAtiva = true
	}
}
