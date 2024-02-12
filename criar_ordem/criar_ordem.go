package criar_ordem

import (
	"binance_robot/config"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"
)

func CriarOrdem(coin string, side string, orderType string, quantity float64, price float64, stopPrice float64) (string, error) {
	if orderType != "STOP" && orderType != "STOP_MARKET" && orderType != "TAKE_PROFIT" && orderType != "TAKE_PROFIT_MARKET" {
		return "", errors.New("orderType deve ser 'STOP', 'TAKE_PROFIT', 'STOP_MARKET', 'TAKE_PROFIT_MARKET', 'LIIMT' ou 'MARKET'")
	}

	var side2 string

	if side == "BUY" {
		side2 = "LONG"
	} else if side == "SELL" {
		side2 = "SHORT"
	}

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()
	apiParamsOrdem := "symbol=" + coin + "" + config.BaseCoin + "&side=" + side + "&quantity=" + fmt.Sprint(quantity) + "&positionSide=" + side2

	var currentType string

	if orderType == "STOP" || orderType == "TAKE_PROFIT" { // Ordem Limit
		currentType = "LIMIT"
		apiParamsOrdem += "&price=" + fmt.Sprint(price) + "&type=" + currentType + "&timeInForce=GTC"
	}

	if orderType == "STOP_MARKET" || orderType == "TAKE_PROFIT_MARKET" { // Ordem Market
		currentType = "MARKET"
		apiParamsOrdem += "&type=" + currentType
	}

	apiParamsOrdem += "&timestamp=" + strconv.FormatInt(timestamp, 10)

	signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)

	urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem

	req, err := http.NewRequest("POST", urlOrdem, nil)
	if err != nil {
		return "Primeira Ordem: ", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "Primeira Ordem: ", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "Primeira Ordem: ", err
	}

	var response models.ResponseOrderStruct
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "Primeira Ordem: ", err
	}

	ordensAbertas, err := listar_ordens.ListarOrdens(coin)
	if err != nil {
		return "", err
	}

	var ordemFiltrada *models.CryptoPosition
	for _, ordem := range ordensAbertas {
		if ordem.PositionSide == side2 {
			ordemFiltrada = &ordem
			break
		}
	}

	var priceLastOrder string
	if ordemFiltrada == nil {
		fmt.Println("Ordem não encontrada: ")
	} else {
		priceLastOrder = ordemFiltrada.EntryPrice
		fmt.Println("Entrada em "+side2+", preço de entrada: "+priceLastOrder+" "+config.BaseCoin+". Quantidade em "+coin+" adquirida: ", quantity)
	}

	var sideReverse string

	if side == "BUY" {
		sideReverse = "SELL"
	}
	if side == "SELL" {
		sideReverse = "BUY"
	}

	apiParamsProfit := "symbol=" + coin + "" + config.BaseCoin + "&side=" + sideReverse + "&positionSide=" + side2 + "&quantity=" + fmt.Sprint(quantity) + "&type=" + orderType + "&stopPrice=" + fmt.Sprint(limitarCasasDecimais(stopPrice, 2)) + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	if orderType == "STOP" || orderType == "TAKE_PROFIT" { // Ordem Limit
		apiParamsProfit += "&price=" + fmt.Sprint(price) + "&timeInForce=GTC"
	}
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
