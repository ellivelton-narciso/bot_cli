package criar_ordem

import (
	"binance_robot/config"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func CriarOrdem(coin string, side string, orderType string, quantity float64, price float64, roi float64) (string, error) {

	if orderType != "STOP" && orderType != "STOP_MARKET" && orderType != "TAKE_PROFIT" && orderType != "TAKE_PROFIT_MARKET" {
		return "", errors.New("orderType deve ser 'STOP', 'TAKE_PROFIT', 'STOP_MARKET', 'TAKE_PROFIT_MARKET', 'LIIMT' ou 'MARKET'")
	}

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()
	apiParamsOrdem := "symbol=" + coin + "" + config.BaseCoin + "&side=" + side + "&quantity=" + fmt.Sprint(quantity)

	if orderType == "STOP" || orderType == "TAKE_PROFIT" { // Ordem Limit
		apiParamsOrdem += "&type=LIMIT&price=" + fmt.Sprint(price)
	}

	if orderType == "STOP_MARKET" || orderType == "TAKE_PROFIT_MARKET" { // Ordem Market
		apiParamsOrdem += "&type=MARKET"
	}

	apiParamsOrdem += "&timestamp=" + strconv.FormatInt(timestamp, 10)

	signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)

	urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem

	req, _ := http.NewRequest("POST", urlOrdem, nil)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	fmt.Println(res)
	fmt.Println(string(body))

	var sideReverse string

	if side == "BUY" {
		sideReverse = "SELL"
	}
	if side == "SELL" {
		sideReverse = "BUY"
	}

	stopPrice := price + (price / 100 * roi)

	apiParamsProfit := "symbol=" + coin + "" + config.BaseCoin + "&side=" + sideReverse + "&quantity=" + fmt.Sprint(quantity) + "&type=" + orderType + "&stopPrice=" + fmt.Sprint(stopPrice) + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signatureProfit := config.ComputeHmacSha256(config.SecretKey, apiParamsProfit)

	urlProfit := config.BaseURL + "fapi/v1/order?" + apiParamsProfit + "&signature=" + signatureProfit

	reqProfit, _ := http.NewRequest("POST", urlProfit, nil)

	reqProfit.Header.Add("Content-Type", "application/json")
	reqProfit.Header.Add("X-MBX-APIKEY", config.ApiKey)

	resProfit, _ := http.DefaultClient.Do(reqProfit)

	defer resProfit.Body.Close()
	bodyProfit, _ := ioutil.ReadAll(resProfit.Body)

	fmt.Println(resProfit)
	fmt.Println(string(bodyProfit))

	return string(body), nil
}
