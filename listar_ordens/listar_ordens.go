package listar_ordens

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

func ListarOrdens(coin string) ([]models.CryptoPosition, error) {

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()

	apiParams := "symbol=" + coin + "" + config.BaseCoin + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)

	url := config.BaseURL + "fapi/v2/positionRisk?" + apiParams + "&signature=" + signature

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response []models.CryptoPosition
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil

}

func PositionAmt(coin string, side string) (string, error) {
	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()

	apiParams := "timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)

	url := config.BaseURL + "fapi/v2/positionRisk?" + apiParams + "&signature=" + signature

	req, _ := http.NewRequest("GET", url, nil)

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

	var response []models.CryptoPosition
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if side == "BUY" {
		side = "LONG"
	} else if side == "SELL" {
		side = "SHORT"
	}

	var positionAmt string
	for _, pos := range response {
		if pos.Symbol == coin+config.BaseCoin && pos.PositionSide == side {
			positionAmt = pos.PositionAmt
			break
		}
	}

	return positionAmt, nil
}

func ListarUltimosValores(coin string, count int64) []models.PriceResponse {
	config.ReadFile()

	rows, err := database.DB.Queryx("SELECT * FROM historico")
	if err != nil {
		log.Fatal(err)
	}
	var historicos []models.Historico
	for rows.Next() {
		var historico models.Historico
		err := rows.StructScan(&historico)
		if err != nil {
			fmt.Println("\n Erro ao buscar historico da DB - ", err)
			continue
		}
		historicos = append(historicos, historico)
	}
	defer rows.Close()

	ultimos := historicos[60-count:]

	var priceRespAll []models.PriceResponse
	for _, item := range ultimos {
		var data []models.PriceResponse
		if err := json.Unmarshal([]byte(item.Value), &data); err != nil {
			fmt.Println("\n Erro ao decodificar JSON - ", err)
			continue
		}
		priceRespAll = append(priceRespAll, data...)
	}

	var priceResp []models.PriceResponse
	for _, item := range priceRespAll {
		if item.Symbol == coin+config.BaseCoin {
			priceResp = append(priceResp, item)
		}
	}

	return priceResp
}
