package deletar_ordens

import (
	"binance_robot/config"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func ClosePosition(symbol, side, quantity string) {
	now := time.Now()
	timestamp := now.UnixMilli()

	apiParamsOrdem := "symbol=" + symbol + config.BaseCoin + "&side=" + side + "&type=MARKET&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(timestamp, 10)

	signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)

	urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem

	req, err := http.NewRequest("POST", urlOrdem, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var responseData models.DeleteResponse
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Resposta da API: %d - %s\n", responseData.Code, responseData.Msg)
	return
}

func CloseAllPositions(coin string, orderType string, stopPrice int) error {
	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()

	longPositionAmt, err := listar_ordens.PositionAmt(coin, "BUY")
	if err != nil {
		return err
	}
	fmt.Println(longPositionAmt)

	shortPositionAmt, err := listar_ordens.PositionAmt(coin, "SELL")
	if err != nil {
		return err
	}
	fmt.Println(shortPositionAmt)

	// Determinar o lado da posição com base nas posições abertas
	var side string
	if longPositionAmt != "" {
		side = "SELL"
	} else if shortPositionAmt != "" {
		side = "BUY"
	} else {
		fmt.Println("Não há posições abertas para fechar.")
		return nil
	}

	apiParams := "symbol=" + coin + config.BaseCoin + "&side=" + side + "&type=" + orderType + "&stopPrice=" + fmt.Sprint(stopPrice) + "&timestamp=" + fmt.Sprint(timestamp)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)

	url := config.BaseURL + "fapi/v1/order?" + apiParams + "&signature=" + signature

	req, _ := http.NewRequest("POST", url, nil)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	// Verifique se a ordem foi bem-sucedida
	if response["orderId"] != nil {
		fmt.Println("Ordem de fechamento de todas as posições enviada com sucesso.")
	} else {
		fmt.Println("Falha ao enviar ordem de fechamento de todas as posições.")
	}

	return nil
}
