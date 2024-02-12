package main

import (
	"binance_robot/config"
	"binance_robot/criar_ordem"
	"binance_robot/util"
	"fmt"
	"strings"
)

func main() {

	//database.DBCon()

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

	currentValue := util.ConvertBaseCoin(currentCoin, value)

	fmt.Println(criar_ordem.CriarOrdem(currentCoin, side, fmt.Sprint(currentValue)))
}
