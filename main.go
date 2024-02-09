package main

import (
	"binance_robot/criar_ordem"
	"binance_robot/deletar_ordens"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"binance_robot/util"
	"fmt"
	"log"
	"time"
)

func main() {

	var (
		currentCoin     string
		value           float64
		margemInferior  float64
		margemSuperior  float64
		quantidadeGrids int
	)

	fmt.Print("Digite a moeda (ex: BTC): ")
	fmt.Scanln(&currentCoin)
	fmt.Print("Digite a quantidade em USDT: ")
	fmt.Scanln(&value)

	fmt.Println("Qual sua margem inferior: ")
	fmt.Scanln(&margemInferior)
	fmt.Println("Qual sua margem superior: ")
	fmt.Scanln(&margemSuperior)
	fmt.Println("Qual a quantidade de grids: ")
	fmt.Scanln(&quantidadeGrids)

	fmt.Println("Para parar as transações pressione Ctrl + C")

	margensSuperiores, margensInferiores := util.CalcularMargens(margemInferior, margemSuperior, quantidadeGrids)

	precoAtual, err := util.PrecoAtual(currentCoin)
	if err != nil {
		fmt.Println("Erro ao obter o preço atual de ", currentCoin)
	}

	fmt.Println(margensSuperiores)
	fmt.Println(margensInferiores)

	currentQuantity, _ := util.ConvertBaseCoin(currentCoin, value)
	fmt.Println(precoAtual)

	for i := 0; i < quantidadeGrids; i++ {
		if precoAtual < margensInferiores[i] {
			fmt.Printf("Entrar em posição longa na grid %d\n", i+1)
			if i == 9 {
				fmt.Println("O ativo atingiu o limite máximo definido")
				break
			}
			_, err := criar_ordem.CriarOrdem(currentCoin, "BUY", "TAKE_PROFIT", currentQuantity/float64(quantidadeGrids), margensInferiores[i], margensInferiores[i+1])
			if err != nil {
				log.Println("Erro ao criar Ordem de Compra: ", err)
			}
			break
		}
	}
	for i := 0; i < quantidadeGrids; i++ {
		if precoAtual < margensSuperiores[i] {
			fmt.Printf("Entrar em posição curta na grid %d\n", i+1)
			if i == 0 {
				fmt.Println("O ativo atingiu o limite mínimo definido")
				break
			}
			_, err := criar_ordem.CriarOrdem(currentCoin, "SELL", "TAKE_PROFIT", currentQuantity/float64(quantidadeGrids), margensSuperiores[i], margensSuperiores[i-1])
			if err != nil {
				log.Println("Erro ao criar Ordem de Compra: ", err)
			}
			break

		}
	}

	for {

		allOpenOrders, err := listar_ordens.ListarOrdens(currentCoin)
		if err != nil {
			fmt.Println("Erro ao consultar as ordens abertas: ", err)
		}
		var filteredOrders *models.CryptoPosition
		for _, order := range allOpenOrders {
			if order.EntryPrice == "0.0" {
				filteredOrders = &order
				break
			}
		}
		var positionSide string
		if filteredOrders != nil {
			precoAtual, err = util.PrecoAtual(currentCoin)
			if filteredOrders.PositionSide == "SHORT" {
				positionSide = "BUY"
				fmt.Println("Ordem concluída, ", filteredOrders.PositionSide)

				_, err := deletar_ordens.DeletarOrdens(currentCoin)
				if err != nil {
					return
				}
				_, err = deletar_ordens.CloseAllPosition(currentCoin, positionSide, fmt.Sprint(precoAtual))
				if err != nil {
					return
				}

				for i := 0; i < quantidadeGrids; i++ {
					if precoAtual < margensSuperiores[i] {
						fmt.Printf("Entrar em posição curta na grid %d\n", i+1)
						if i == 0 {
							fmt.Println("O ativo atingiu o limite mínimo definido")
							break
						}
						_, err := criar_ordem.CriarOrdem(currentCoin, "BUY", "TAKE_PROFIT", currentQuantity/float64(quantidadeGrids), margensSuperiores[i], margensSuperiores[i-1])
						if err != nil {
							log.Println("Erro ao criar Ordem de Compra: ", err)
						}
						break

					}
				}
				for i := 0; i < quantidadeGrids; i++ {
					if precoAtual < margensInferiores[i] {
						fmt.Printf("Entrar em posição longa na grid %d\n", i+1)
						if i == 9 {
							fmt.Println("O ativo atingiu o limite máximo definido")
							break
						}
						_, err := criar_ordem.CriarOrdem(currentCoin, "SELL", "TAKE_PROFIT", currentQuantity/float64(quantidadeGrids), margensInferiores[i], margensInferiores[i+1])
						if err != nil {
							log.Println("Erro ao criar Ordem de Compra: ", err)
						}
						break
					}
				}

			} else if filteredOrders.PositionSide == "LONG" {
				positionSide = "SELL"
				fmt.Println("Ordem concluída: ", filteredOrders.PositionSide)

				_, err := deletar_ordens.DeletarOrdens(currentCoin)
				if err != nil {
					return
				}
				_, err = deletar_ordens.CloseAllPosition(currentCoin, positionSide, fmt.Sprint(precoAtual))
				if err != nil {
					return
				}

				for i := 0; i < quantidadeGrids; i++ {
					if precoAtual < margensSuperiores[i] {
						fmt.Printf("Entrar em posição curta na grid %d\n", i+1)
						if i == 0 {
							fmt.Println("O ativo atingiu o limite mínimo definido")
							break
						}
						_, err := criar_ordem.CriarOrdem(currentCoin, "BUY", "TAKE_PROFIT", currentQuantity/float64(quantidadeGrids), margensSuperiores[i], margensSuperiores[i-1])
						if err != nil {
							log.Println("Erro ao criar Ordem de Compra: ", err)
						}
						break

					}
				}
				for i := 0; i < quantidadeGrids; i++ {
					if precoAtual < margensInferiores[i] {
						fmt.Printf("Entrar em posição longa na grid %d\n", i+1)
						if i == 9 {
							fmt.Println("O ativo atingiu o limite máximo definido")
							break
						}
						_, err := criar_ordem.CriarOrdem(currentCoin, "SELL", "TAKE_PROFIT", currentQuantity/float64(quantidadeGrids), margensInferiores[i], margensInferiores[i+1])
						if err != nil {
							log.Println("Erro ao criar Ordem de Compra: ", err)
						}
						break
					}
				}

			}

			if err != nil {
				fmt.Println("Erro ao criar Ordem: ", err)
			}
		}

		time.Sleep(1 * time.Second)
	}
}

/*func main() {

	atual, err := util.PrecoAtual("BTC")
	if err != nil {
		fmt.Println(err)
	}
	_, err = deletar_ordens.CloseAllPosition("BTC", "BUY", fmt.Sprint(atual))
	if err != nil {
		fmt.Println(err)
	}
}*/
