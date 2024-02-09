package main

import (
	"binance_robot/config"
	"binance_robot/criar_ordem"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	var currentCoin string
	var value float64
	var currentAlavancagem int
	var porcentagemLucro float64

	fmt.Print("Digite a moeda (ex: BTC): ")
	fmt.Scanln(&currentCoin)
	fmt.Print("Digite a quantidade em USDT: ")
	fmt.Scanln(&value)
	fmt.Print("Digite a alavancagem: ")
	fmt.Scanln(&currentAlavancagem)
	fmt.Print("Qual seu Take Profit (ROI %): ")
	fmt.Scanln(&porcentagemLucro)

	fmt.Println("Para parar as transações pressione Ctrl + C")

	currentQuantity, _ := convertUSDT(currentCoin, value)
	_, err := criar_ordem.CriarOrdem(currentCoin, "SELL", "TAKE_PROFIT_MARKET", currentQuantity, 0, porcentagemLucro, currentAlavancagem)
	if err != nil {
		log.Println("Erro ao criar Ordem de Compra: ", err)
	}
	_, err = criar_ordem.CriarOrdem(currentCoin, "BUY", "TAKE_PROFIT_MARKET", currentQuantity, 0, porcentagemLucro, currentAlavancagem)
	if err != nil {
		log.Println("Erro ao criar Ordem de Compra: ", err)
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
			if filteredOrders.PositionSide == "SHORT" {
				positionSide = "SELL"
				fmt.Println("Ordem concluída, ", filteredOrders.PositionSide)
			} else if filteredOrders.PositionSide == "LONG" {
				positionSide = "BUY"
				fmt.Println("Ordem concluída: ", filteredOrders.PositionSide)
			}
			_, err := criar_ordem.CriarOrdem(currentCoin, positionSide, "TAKE_PROFIT_MARKET", currentQuantity, 0, porcentagemLucro, currentAlavancagem)
			if err != nil {
				fmt.Println("Erro ao criar Ordem: ", err)
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func convertUSDT(coin string, value float64) (price float64, err error) {

	config.ReadFile()

	url := config.BaseURL + "fapi/v1/ticker/price?symbol=" + coin + config.BaseCoin
	req, _ := http.NewRequest("GET", url, nil)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Erro ao acessar a API para converter: ", err)
		return
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var priceResp models.PriceResponse
	err = json.Unmarshal(body, &priceResp)
	if err != nil {
		fmt.Println("Erro ao decodificar JSON:", err)
		return
	}

	precision := getPrecision(priceResp.Price)

	price, err = strconv.ParseFloat(priceResp.Price, 64)
	if err != nil {
		fmt.Println("Erro ao converter preço para float64:", err)
		return
	}

	q := value / price
	quantity := math.Round(q*math.Pow(10, float64(precision))) / math.Pow(10, float64(precision))

	return quantity, nil
}

func getPrecision(number string) int {
	parts := strings.Split(number, ".")
	if len(parts) == 2 {
		return len(parts[1])
	}
	return 0
}
