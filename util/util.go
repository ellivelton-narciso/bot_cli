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

func GetPrecision(number string) int {
	parts := strings.Split(number, ".")
	if len(parts) == 2 {
		if len(parts[1]) > 4 {
			return 4
		} else {
			return len(parts[1])
		}
	}
	return 0
}

func removerZeros(number string) string {
	var newValue string
	foundNonZero := false
	for i := len(number) - 1; i >= 0; i-- {
		if number[i] != '0' {
			foundNonZero = true
		}
		if foundNonZero {
			newValue = string(number[i]) + newValue
		}
	}
	return newValue
}
