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
	)

	fmt.Print("Digite a moeda (ex: BTC): ")
	fmt.Scanln(&currentCoin)
	currentCoin = strings.ToUpper(currentCoin)
	fmt.Print("Digite a quantidade em " + config.BaseCoin + ": ")
	fmt.Scanln(&value)

	for {
		fmt.Println("Irá trabalhar em LONG ou SHORT? (ex: BUY, SELL)")
		fmt.Scanln(&side)
		side = strings.ToUpper(side)
		if side == "LONG" || side == "SHORT" || side == "BUY" || side == "SELL" {
			break
		} else {
			fmt.Println("Deve entrar somente em LONG ou SHORT")
		}
	}

	fmt.Println("Qual sua margem inferior: ")
	fmt.Scanln(&margemInferior)
	fmt.Println("Qual sua margem superior: ")
	fmt.Scanln(&margemSuperior)

	fmt.Println("Para parar as transações pressione Ctrl + C")
	var currentValue float64

	var ordemAtiva bool
	var primeiraExecStop bool
	var primeiraExec bool
	var entryPrice string
	ordemAtiva = false
	primeiraExec = true
	primeiraExecStop = true

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
				if ultimos[4].Price >= ultimos[3].Price && ultimos[3].Price >= ultimos[2].Price && ultimos[2].Price >= ultimos[1].Price && ultimos[1].Price >= ultimos[0].Price {
					currentValue = util.ConvertBaseCoin(currentCoin, value)
					fmt.Println(currentValue)
					entryPrice, _ = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue))
					ordemAtiva = true
					fmt.Println("Entrada em LONG")
					fmt.Println(entryPrice)

					if primeiraExecStop {
						primeiraExecStop = false
						err := criar_ordem.FecharOrdem(currentCoin, side, currentValue, margemSuperior, "TAKE_PROFIT_MARKET")
						if err != nil {
							log.Panic(err)
						}
						err = criar_ordem.FecharOrdem(currentCoin, side, currentValue, margemInferior, "STOP_MARKET")
						if err != nil {
							log.Panic(err)
						}
					}

				}

			} else if side == "SELL" {
				if ultimos[4].Price <= ultimos[3].Price && ultimos[3].Price <= ultimos[2].Price && ultimos[2].Price <= ultimos[1].Price && ultimos[1].Price <= ultimos[0].Price {
					currentValue = util.ConvertBaseCoin(currentCoin, value)
					fmt.Println(currentValue)
					entryPrice, _ = criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue))
					ordemAtiva = true
					fmt.Println("Entrada em SHORT")
					fmt.Println(entryPrice)

					if primeiraExecStop {
						primeiraExecStop = false
						err := criar_ordem.FecharOrdem(currentCoin, side, currentValue, margemInferior, "TAKE_PROFIT_MARKET")
						if err != nil {
							log.Panic(err)
						}
						err = criar_ordem.FecharOrdem(currentCoin, side, currentValue, margemSuperior, "STOP_MARKET")
						if err != nil {
							log.Panic(err)
						}
					}

				}
			}
		} else {
			if side == "BUY" {
				if ultimos[4].Price < ultimos[3].Price && ultimos[3].Price < ultimos[2].Price && ultimos[2].Price < ultimos[1].Price {
					currentValue = util.ConvertBaseCoin(currentCoin, value)
					stopPrice, _ := strconv.ParseFloat(ultimos[4].Price, 64)
					fmt.Println(stopPrice)
					// Falta calcular se valor de stop é maior que o valor de entrada
					err := criar_ordem.FecharOrdem(currentCoin, side, currentValue, stopPrice, "TAKE_PROFIT_MARKET")
					if err != nil {
						fmt.Println(err)
						return
					}
					ordemAtiva = false
				}
			} else if side == "SELL" {
				if ultimos[4].Price > ultimos[3].Price && ultimos[3].Price > ultimos[2].Price && ultimos[2].Price > ultimos[1].Price {
					currentValue = util.ConvertBaseCoin(currentCoin, value)
					fmt.Println(currentValue)
					stopPrice, _ := strconv.ParseFloat(ultimos[4].Price, 64)
					err := criar_ordem.FecharOrdem(currentCoin, side, currentValue, stopPrice, "TAKE_PROFIT_MARKET")
					if err != nil {
						fmt.Println(err)
						return
					}
					ordemAtiva = false
				}
			}
		}

		time.Sleep(1 * time.Second)
	}

}

/*func main() {
	currentValue := util.ConvertBaseCoin("BTC", 2000)
	criar_ordem.FecharOrdem("BTC", "BUY", currentValue, 50000, "STOP_MARKET")

	database.DBCon()
	fmt.Println(listar_ordens.ListarUltimosValores("BTC", 5))
}*/
