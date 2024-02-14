package main

import (
	"binance_robot/config"
	"binance_robot/criar_ordem"
	"binance_robot/database"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"binance_robot/util"
	"fmt"
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
	fmt.Println("Qual sua alavancagem: ")
	fmt.Scanln(&alavancagem)

	fmt.Println("Qual o ROI que deseja trabalhar: ")
	fmt.Scanln(&roi)

	fmt.Println("Qual sua margem inferior: ")
	fmt.Scanln(&margemInferior)
	fmt.Println("Qual sua margem superior: ")
	fmt.Scanln(&margemSuperior)

	fmt.Println("Para parar as transações pressione Ctrl + C")
	var currentValue float64

	var ordemAtiva bool
	var valueComprado float64
	var primeiraExec bool
	var entryPrice string
	ordemAtiva = false
	primeiraExec = true
	valueComprado = 0.0

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
					currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
					valueComprado = currentValue
					fmt.Println("Entrada em LONG: " + ultimos[4].Price)
					entryPrice, _ = criar_ordem.CriarOrdem(currentCoin, side, "LONG", fmt.Sprint(currentValue))
					ordemAtiva = true
					fmt.Println(entryPrice)
				}

			} else if side == "SELL" {
				if ultimos[4].Price <= ultimos[3].Price && ultimos[3].Price <= ultimos[2].Price && ultimos[2].Price <= ultimos[1].Price && ultimos[1].Price <= ultimos[0].Price {
					currentValue = util.ConvertBaseCoin(currentCoin, value*alavancagem)
					valueComprado = currentValue
					fmt.Println("Entrada em SHORT: " + ultimos[4].Price)
					entryPrice, _ = criar_ordem.CriarOrdem(currentCoin, side, "SHORT", fmt.Sprint(currentValue))
					ordemAtiva = true
					fmt.Println(entryPrice)

				}
			}
		} else {
			if side == "BUY" {
				currentPrice, _ := strconv.ParseFloat(ultimos[4].Price, 64)
				ROI := ((currentPrice - valueComprado) / valueComprado) * 100

				if ultimos[4].Price < ultimos[3].Price && ultimos[3].Price < ultimos[2].Price {
					fmt.Println("Ordem encerrada - desceu 2 consecutivos: " + ultimos[4].Price)
					_, err := criar_ordem.CriarOrdem(currentCoin, "SELL", "LONG", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false

				} else if ultimos[4].Price >= fmt.Sprint(margemSuperior) {
					fmt.Println("Ordem encerrada - Atingiu a Margem Superior: " + ultimos[4].Price)
					_, err := criar_ordem.CriarOrdem(currentCoin, "SELL", "LONG", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				} else if ultimos[4].Price <= fmt.Sprint(margemInferior) {
					fmt.Println("Ordem encerrada - atingiu a margem inferior: " + ultimos[4].Price)
					_, err := criar_ordem.CriarOrdem(currentCoin, "SELL", "LONG", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				} else if ROI >= roi {
					fmt.Println("Ordem encerrada - ROI atingido: " + ultimos[4].Price)
					_, err := criar_ordem.CriarOrdem(currentCoin, "SELL", "LONG", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				}
			} else if side == "SELL" {
				currentPrice, _ := strconv.ParseFloat(ultimos[4].Price, 64)
				ROI := ((valueComprado - currentPrice) / valueComprado) * 100

				if ultimos[4].Price > ultimos[3].Price && ultimos[3].Price > ultimos[2].Price {
					fmt.Println("Ordem encerrada - subiu 2 consecutivos: " + ultimos[4].Price)
					_, err := criar_ordem.CriarOrdem(currentCoin, "BUY", "SHORT", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false

				} else if ultimos[4].Price >= fmt.Sprint(margemSuperior) {
					fmt.Println("Ordem encerrada - atingiu a margem superior: " + ultimos[4].Price)
					_, err := criar_ordem.CriarOrdem(currentCoin, "BUY", "SHORT", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				} else if ultimos[4].Price <= fmt.Sprint(margemInferior) {
					fmt.Println("Ordem encerrada - atingiu a margem inferior: " + ultimos[4].Price)
					_, err := criar_ordem.CriarOrdem(currentCoin, "BUY", "SHORT", fmt.Sprint(valueComprado))
					if err != nil {
						return
					}
					ordemAtiva = false
				} else if ROI >= roi {
					fmt.Println("Ordem encerrada - atingiu roi: " + ultimos[4].Price)
					_, err := criar_ordem.CriarOrdem(currentCoin, "BUY", "SHORT", fmt.Sprint(valueComprado))
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
