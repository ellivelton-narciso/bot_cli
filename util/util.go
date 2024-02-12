package util

import (
	"binance_robot/config"
	"binance_robot/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
)

func CalcularMargens(precoMin, precoMax float64, numGrids int) ([]float64, []float64) {
	intervaloTotal := precoMax - precoMin
	tamanhoGrid := intervaloTotal / float64(numGrids)

	margensSuperiores := make([]float64, numGrids)
	margensInferiores := make([]float64, numGrids)

	for i := 0; i < numGrids; i++ {
		precoEntrada := precoMin + float64(i)*tamanhoGrid
		margemSuperior := precoEntrada + tamanhoGrid/2.0
		margemInferior := precoEntrada - tamanhoGrid/2.0

		margensSuperiores[i] = margemSuperior
		margensInferiores[i] = margemInferior
	}

	return margensSuperiores, margensInferiores
}

func ConvertBaseCoin(coin string, value float64) (price float64) {

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

	precision := GetPrecision(priceResp.Price)

	price, err = strconv.ParseFloat(priceResp.Price, 64)
	if err != nil {
		fmt.Println("Erro ao converter preço para float64:", err)
		return
	}

	q := value / price
	quantity := math.Round(q*math.Pow(10, float64(precision))) / math.Pow(10, float64(precision))

	return quantity
}

func PrecoAtual(coin string) (price float64, err error) {
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

	priceReturn, err := strconv.ParseFloat(priceResp.Price, 64)
	if err != nil {
		fmt.Println("Erro ao converter string para float:", err)
		return
	}

	return priceReturn, nil
}

func GetPrecision(number string) int {
	parts := strings.Split(number, ".")
	if len(parts) == 2 {
		return len(parts[1])
	}
	return 0
}
