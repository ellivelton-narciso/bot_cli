package criar_ordem

import (
	"binance_robot/config"
	"binance_robot/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"
)

func CriarOrdem(coin string, side string, quantity string) (string, error) {
	var side2 string

	if side == "BUY" {
		side2 = "LONG"
	} else if side == "SELL" {
		side2 = "SHORT"
	}

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()
	apiParamsOrdem := "symbol=" + coin + "" + config.BaseCoin + "&type=MARKET&side=" + side + "&quantity=" + quantity + "&positionSide=" + side2 + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)

	urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem

	req, err := http.NewRequest("POST", urlOrdem, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response models.ResponseOrderStruct
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	fmt.Println(string(body))

	return strconv.FormatInt(response.OrderId, 10), nil
}

func limitarCasasDecimais(numero float64, casasDecimais int) float64 {
	multiplicador := math.Pow(10, float64(casasDecimais))
	return math.Round(numero*multiplicador) / multiplicador
}

func calcularROIAlavancado(roi float64, alavancagem float64) float64 {
	fatorAlavancagem := 1 / alavancagem
	roiAjustado := roi * fatorAlavancagem
	return roiAjustado
}

func FecharOrdem(coin string, side string, quantity float64, stopPrice float64, orderType string) (string, error) {

	now := time.Now()
	timestamp := now.UnixMilli()

	var side2 string
	if side == "BUY" {
		side2 = "LONG"
	} else if side == "SELL" {
		side2 = "SHORT"
	}
	var sideReverse string

	if side == "BUY" {
		sideReverse = "SELL"
	}
	if side == "SELL" {
		sideReverse = "BUY"
	}

	apiParamsProfit := "symbol=" + coin + "" + config.BaseCoin + "&side=" + sideReverse + "&positionSide=" + side2 + "&quantity=" + fmt.Sprint(quantity) + "&type=" + orderType + "&stopPrice=" + fmt.Sprint(limitarCasasDecimais(stopPrice, 2)) + "&timestamp=" + strconv.FormatInt(timestamp, 10)

	signatureProfit := config.ComputeHmacSha256(config.SecretKey, apiParamsProfit)

	urlProfit := config.BaseURL + "fapi/v1/order?" + apiParamsProfit + "&signature=" + signatureProfit

	reqProfit, err := http.NewRequest("POST", urlProfit, nil)
	if err != nil {
		return "Segunda Ordem: ", err
	}

	reqProfit.Header.Add("Content-Type", "application/json")
	reqProfit.Header.Add("X-MBX-APIKEY", config.ApiKey)

	resProfit, err := http.DefaultClient.Do(reqProfit)
	if err != nil {
		return "Segunda Ordem: ", err
	}
	defer resProfit.Body.Close()

	_, err = ioutil.ReadAll(resProfit.Body)
	if err != nil {
		return "Segunda Ordem: ", err
	}

	return "", nil
}
