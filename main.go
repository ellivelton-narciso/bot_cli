package main

import (
	"binance_robot/database"
	"fmt"
)

func main() {

	database.DBCon()

	var (
		currentCoin    string
		value          float64
		margemInferior float64
		margemSuperior float64
	)

	fmt.Print("Digite a moeda (ex: BTC): ")
	fmt.Scanln(&currentCoin)
	fmt.Print("Digite a quantidade em USDT: ")
	fmt.Scanln(&value)

	fmt.Println("Qual sua margem inferior: ")
	fmt.Scanln(&margemInferior)
	fmt.Println("Qual sua margem superior: ")
	fmt.Scanln(&margemSuperior)

	fmt.Println("Para parar as transações pressione Ctrl + C")

}
