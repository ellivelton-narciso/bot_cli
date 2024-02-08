package main

import (
	"binance_robot/criar_ordem"
	"fmt"
)

func main() {
	fmt.Println(criar_ordem.CriarOrdem("BTC", "BUY", "TAKE_PROFIT_MARKET", 0.05, 0, 1.3))
	fmt.Println(criar_ordem.CriarOrdem("BTC", "SELL", "TAKE_PROFIT_MARKET", 0.05, 0, 1.3))

	//fmt.Println(listar_ordens.ListarOrdens("BTC"))
}
