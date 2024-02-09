package deletar_ordens

import (
	"binance_robot/config"
	"net/http"
	"strconv"
	"time"
)

func DeletarOrdens(coin string) (string, error) {

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()

	apiParamsOrdem := "symbol=" + coin + config.BaseCoin + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)
	urlOrdem := config.BaseURL + "fapi/v1/allOpenOrders?" + apiParamsOrdem + "&signature=" + signatureOrdem

	req, err := http.NewRequest("DELETE", urlOrdem, nil)
	if err != nil {
		return "Erro ao deletar ordens: ", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "Erro ao deletar ordens: ", err
	}
	defer res.Body.Close()

	return "", nil
}

func CloseAllPosition(coin string, side string, stopPrice string) (string, error) {
	config.ReadFile()

	var side2 string

	if side == "BUY" {
		side2 = "LONG"
	} else if side == "SELL" {
		side2 = "SHORT"
	}

	now := time.Now()
	timestamp := now.UnixMilli()

	apiParamsOrdem := "symbol=" + coin + config.BaseCoin + "&side=" + side + "&type=TAKE_PROFIT_MARKET&positionSide=" + side2 + "&stopPrice=" + stopPrice + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)
	urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem
	req, err := http.NewRequest("POST", urlOrdem, nil)
	if err != nil {
		return "Aqui", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "Erro aq", err
	}
	defer res.Body.Close()
	return "Foi", nil

}
