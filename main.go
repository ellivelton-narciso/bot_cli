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
	ROI               float64
	order             int
	roiAcumuladoStr   string
	roiTempoRealStr   string
	red               func(a ...interface{}) string
	green             func(a ...interface{}) string
	roiTempoReal      float64
	roiMaximo         float64
	started           string
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

	if config.ApiKey == "" || config.SecretKey == "" || config.BaseURL == "" || config.BaseCoin == "" {
		log.Panic("Arquivo user.json incompleto.")
	}

	red = color.New(color.FgHiRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	ordemAtiva = false
	primeiraExec = true
	valueCompradoCoin = 0.0
	roiAcumulado = 0.0
	roiTempoReal = 0.0
	fee := 0.05 * alavancagem
	roiMaximo = 0

	side = strings.ToUpper(side)
	if side == "LONG" {
		side = "BUY"
	}
	if side == "SHORT" {
		side = "SELL"
	}

	fmt.Println("Para parar as transações pressione Ctrl + C")

	util.DefinirAlavancagem(currentCoin, alavancagem)
	util.DefinirMargim(currentCoin, "ISOLATED")

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
				} else {
					util.Write("Erro ao encerrar ordem. Finalize manulmente no site da Binance.", currentCoin+config.BaseCoin)
				}
			}
		} else {
			fmt.Println(roiAcumulado)
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
			time.Sleep(2 * time.Second)
			primeiraExec = false
		}
		ultimosSaida = listar_ordens.ListarUltimosValores(currentCoin, 1)
		currentPrice, err = strconv.ParseFloat(ultimosSaida[0].Price, 64)

		if err != nil {
			log.Println(err)
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
			if side == "BUY" {
				ROI = (((currentPrice - valueCompradoCoin) / (valueCompradoCoin / alavancagem)) * 100) - (fee * 2)
				if ROI > roiMaximo {
					roiMaximo = ROI
				}
				roiTempoReal = roiAcumulado + ROI
				if roiTempoReal > 0 {
					roiTempoRealStr = green(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				} else {
					roiTempoRealStr = red(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				}
				util.Write("Valor de entrada ("+green("LONG")+"): "+fmt.Sprint(valueCompradoCoin)+" | "+formattedTime+" | "+fmt.Sprint(currentPrice)+" | Roi acumulado: "+roiTempoRealStr, currentCoin+config.BaseCoin)

				if config.Development {
					if ROI >= config.TP3 {
						util.Historico(currentCoin, side, started, "tp3", currentPrice, valueCompradoCoin)
					} else if ROI >= config.TP2 {
						util.Historico(currentCoin, side, started, "tp2", currentPrice, valueCompradoCoin)
					} else if ROI >= config.TP1 {
						util.Historico(currentCoin, side, started, "tp1", currentPrice, valueCompradoCoin)
					}
					if ROI <= config.SL3 {
						util.Historico(currentCoin, side, started, "sl3", currentPrice, valueCompradoCoin)
					} else if ROI <= config.SL2 {
						util.Historico(currentCoin, side, started, "sl2", currentPrice, valueCompradoCoin)
					} else if ROI <= config.SL1 {
						util.Historico(currentCoin, side, started, "sl1", currentPrice, valueCompradoCoin)
					}
				}

				if ROI <= roiMaximo-(stop) { // Stop Loss
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
						os.Exit(1)
					} else {
						util.Write("Erro ao encerrar ordem. Pode a qualquer momento digitar STOP para encerrar a ordem.", currentCoin+config.BaseCoin)
						ordemAtiva = true
					}
				}
			} else if side == "SELL" {
				ROI = (((valueCompradoCoin - currentPrice) / (valueCompradoCoin / alavancagem)) * 100) - (fee * 2)
				if ROI > roiMaximo {
					roiMaximo = ROI
				}
				roiTempoReal := roiAcumulado + ROI
				if roiTempoReal > 0 {
					roiTempoRealStr = green(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				} else {
					roiTempoRealStr = red(fmt.Sprintf("%.4f", roiTempoReal) + "%")
				}
				util.Write("Valor de entrada ("+red("SHORT")+"): "+fmt.Sprint(valueCompradoCoin)+" | "+formattedTime+" | "+currentPriceStr+" | Roi acumulado: "+roiTempoRealStr, currentCoin+config.BaseCoin)

				if config.Development {
					if ROI >= config.TP3 {
						util.Historico(currentCoin, side, started, "tp3", currentPrice, valueCompradoCoin)
					} else if ROI >= config.TP2 {
						util.Historico(currentCoin, side, started, "tp2", currentPrice, valueCompradoCoin)
					} else if ROI >= config.TP1 {
						util.Historico(currentCoin, side, started, "tp1", currentPrice, valueCompradoCoin)
					}
					if ROI <= config.SL3 {
						util.Historico(currentCoin, side, started, "sl3", currentPrice, valueCompradoCoin)
					} else if ROI <= config.SL2 {
						util.Historico(currentCoin, side, started, "sl2", currentPrice, valueCompradoCoin)
					} else if ROI <= config.SL1 {
						util.Historico(currentCoin, side, started, "sl1", currentPrice, valueCompradoCoin)
					}
				}

				if ROI <= roiMaximo-(stop) {
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
	valueCompradoCoin = currentPrice
	now = time.Now()
	timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
	started = timeValue.Format("2006-01-02 15:04:05")
	side = "BUY"
	util.Write("Entrada em LONG: "+currentPriceStr, currentCoin+config.BaseCoin)
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
		util.Historico(currentCoin, side, started, "", valueCompradoCoin, valueCompradoCoin)
	} else {
		side = ""
		util.Write("A ordem de LONG não foi totalmente completada.", currentCoin+config.BaseCoin)
		ordemAtiva = false
		os.Exit(1)

	}
	return order

}

func comprarSell() int {
	currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
	valueCompradoCoin = currentPrice
	now = time.Now()
	timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
	started = timeValue.Format("2006-01-02 15:04:05")
	util.Write("Entrada em SHORT: "+currentPriceStr, currentCoin+config.BaseCoin)
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
		util.Historico(currentCoin, side, started, "", valueCompradoCoin, valueCompradoCoin)
	} else {
		side = ""
		util.Write("A ordem de SHORT não foi totalmente completada. Irei voltar a buscar novas oportunidades. Pode a qualquer momento digitar SELL para entrar em SHORT.", currentCoin+config.BaseCoin)
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
				side = c.String("side")
				alavancagem = c.Float64("alavancagem")
				stop = c.Float64("stop")

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
					Value:   0,
					Usage:   "Quantidade em moeda base (" + config.BaseCoin + ")",
					Aliases: []string{"v"},
				},
				&cli.Float64Flag{
					Name:    "stop",
					Value:   1.5,
					Usage:   "Definir o Stop Loss da Ordem.",
					Aliases: []string{"sl"},
				},
				&cli.StringFlag{
					Name:    "side",
					Usage:   "Direção da ordem (BUY, SELL, LONG, SHORT)",
					Aliases: []string{"s"},
				},
				&cli.Float64Flag{
					Name:    "leverage",
					Value:   1,
					Usage:   "Definir a alavancagem.",
					Aliases: []string{"le"},
				},
			},
		},
	}
}
