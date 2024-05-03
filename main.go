package main

import (
	"candles/config"
	"candles/database"
	"candles/exec"
	"candles/global"
	"candles/ordem"
	"candles/strategy"
	"candles/util"
	"fmt"
	"github.com/fatih/color"
	"log"
	"strings"
	"time"
)

func main() {
	config.ReadFile()
	database.DBCon()

	var (
		currentCoin string
		value       float64
		alavancagem float64
		fee         float64
		modo        string
		stop        float64
		stopLossAll float64
		roi         float64
		err         error
	)
	fee = 0.05 * 2
	modo = "CROSSED"
	global.Key = config.ApiKey[:5]

	global.Red = color.New(color.FgHiRed).SprintFunc()
	global.Green = color.New(color.FgGreen).SprintFunc()
	global.OrdemAtiva = false

	for {
		fmt.Print("Digite a moeda em ordem de prioridade (ex: BTC,ETH,ADA): ")
		_, err = fmt.Scanln(&currentCoin)
		if err != nil {
			fmt.Println("Erro, tente digitar somente letras: ", err)
			continue
		}
		currentCoin = strings.ToUpper(currentCoin)
		if len(currentCoin) > 0 {
			break
		} else {
			fmt.Println("Por favor, insira uma moeda válida.")
		}
	} // currentCoin
	for {
		fmt.Print("Digite a quantidade em " + config.BaseCoin + ": ")
		_, err = fmt.Scanln(&value)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if value > 0 {
			break
		} else {
			fmt.Println("Por favor, digite um valor válido.")
		}
	} // value
	for {
		fmt.Println("Qual sua alavancagem (1 - 20): ")
		_, err = fmt.Scanln(&alavancagem)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		} else if alavancagem <= 0 {
			alavancagem = 1
			fee = fee * alavancagem
			value = value * alavancagem
			fmt.Println("Alavancagem menor que 0 definido como 1.")
			break
		}
		fee = fee * alavancagem
		break
	} // alavancagem
	if alavancagem > 1 {
		for {
			fmt.Println("Isolado ou Cruzado?: (I, C)")
			_, err = fmt.Scanln(&modo)
			if err != nil {
				fmt.Println("Erro, tente digitar somente letras: ", err)
				continue
			}
			modo = strings.ToUpper(modo)
			if modo != "I" && modo != "C" {
				fmt.Println("Erro, tente digitar somente I para Isolado e C para Cruzado: ", err)
				continue
			} else if modo == "I" {
				modo = "ISOLATED"
			} else if modo == "C" {
				modo = "CROSSED"
			}
			break
		} // alavancagem
	}
	for {
		fmt.Println("Qual o Stop Loss por ORDEM que deseja trabalhar em porcentagem (ex: 0.5): ")
		_, err = fmt.Scanln(&stop)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if stop > 0 {
			break
		} else {
			fmt.Println("Stop Loss por ORDEM precisa ser maior que 0")
			continue
		}
	} // stopLoss
	for {
		fmt.Println("Qual o Stop Loss TOTAL que deseja trabalhar em porcentagem (ex: 0.5): ")
		_, err = fmt.Scanln(&stopLossAll)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if stopLossAll > stop {
			break
		} else {
			fmt.Println("Stop Loss TOTAL precisa ser maior que o Stop Loss por Ordem")
			continue
		}
	} // stopLoss Total
	for {
		fmt.Println("Qual será seu TAKE PROFIT em % total? (Ao atingir o valor a aplicação será encerrada totalmente).")
		_, err = fmt.Scanln(&roi)
		if err != nil {
			fmt.Println("Erro, tente digitar somente números: ", err)
			continue
		}
		if roi <= 0 {
			fmt.Println("TAKEPROFIT precisa ser maior que 0")
		} else {
			break
		}
	} // TAKEPROFIT Total
	for {
		var resposta string
		fmt.Println("Quer o Stop Loss seja dinamico e acompanhe o ganho maximo ? (S/N): ")
		_, err = fmt.Scanln(&resposta)
		if err != nil {
			fmt.Println("Erro, tente digitar somente letras: ", err)
			continue
		}
		resposta = strings.ToUpper(resposta)
		if resposta == "S" || resposta == "N" || resposta == "SIM" || resposta == "NAO" || resposta == "NÃO" {
			if resposta == "S" {
				global.StopMovel = true
			} else {
				global.StopMovel = false
			}
			break
		} else {
			fmt.Println("Digite somente S ou N")
			continue
		}
	} // STOP Dinamico

	global.SliceCurrentCoin = strings.Split(currentCoin, ",")
	global.Value = value
	global.TP = roi
	global.Stop = stop
	global.StopLossAll = stopLossAll
	global.Meta = false

	go util.ControlRoutine()
	go util.ControlSleep()
	err = util.SendMessageToDiscord("Iniciado, " + currentCoin)
	if err != nil {
		log.Println("Erro ao enviar mensagem para discord de alerta")
	}
	priority := 0
	for _, symbol := range global.SliceCurrentCoin {
		priority++
		symbol = symbol + config.BaseCoin
		ordem.EnviarCoinDB(symbol, global.Key)
		go strategy.ExecuteStrategy2(symbol, global.Key, priority)
		//go strategy.ExecuteStrategy(symbol, global.Key, priority)

		go exec.ExecutarOrdem(symbol, modo, alavancagem, priority)

	}
	for {
		time.Sleep(15 * time.Second)
	}

	/*err = ordem.RemoverCoinDB(exec.CurrentCoin, global.Key)
	if err != nil {
		log.Println("Erro ao remover coin da DB")
	}*/
}
