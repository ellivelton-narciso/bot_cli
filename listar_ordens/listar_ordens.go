package listar_ordens

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/models"
	"binance_robot/util"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

func ListarOrdens(coin string) ([]models.CryptoPosition, error) {

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()

	apiParams := "symbol=" + coin + "" + "&timestamp=" + strconv.FormatInt(timestamp, 10)
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

func ListarUltimosValores(coin string) ([]models.HistoricoAll, error) {
	config.ReadFile()

	var historicos []models.HistoricoAll
	query := `SELECT * FROM hist_trading_values WHERE trading_name = ? ORDER BY hist_date desc  LIMIT 1`

	err := database.DB.Raw(query, coin).Scan(&historicos).Error
	if err != nil {
		return nil, err
	}
	if len(historicos) == 0 {
		return nil, errors.New("hist_trading_values retornou um array vazio")
	}
	return historicos, nil
}

func ListarUltimosValoresBuy(coin string, count int64) ([]models.PriceResponse, float64) {

	config.ReadFile()

	url := config.BaseURL + "fapi/v1/ticker/price?symbol=" + coin
	req, _ := http.NewRequest("GET", url, nil)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		util.WriteError("Erro ao acessar a API para converter: ", err, coin)
		os.Exit(1)

	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
	}

	var priceR models.PriceResponse
	err = json.Unmarshal(body, &priceR)
	if err != nil {
		fmt.Println("Erro ao decodificar JSON:", err)
	}

	if priceR.Price == "" {
		fmt.Println("Preço é vazio, provavelmente devido a algum erro na requisição em momento de compra, , StatusCode: ", response.StatusCode, " Coin: ", coin)
		var historicos []models.Historico
		database.DB.Order("created_at DESC").Limit(int(count)).Find(&historicos)

		var priceRespAll []models.PriceResponse
		for _, historico := range historicos {
			var data []models.PriceResponse
			if err := json.Unmarshal([]byte(historico.Value), &data); err != nil {
				fmt.Println("\n Erro ao decodificar JSON - ", err)
				continue
			}
			priceRespAll = append(priceRespAll, data...)
		}

		var priceResp []models.PriceResponse
		for _, item := range priceRespAll {
			if item.Symbol == coin {
				priceResp = append(priceResp, item)
			}
		}

		return priceResp, 0
	}

	price, err := strconv.ParseFloat(priceR.Price, 64)

	if err != nil {
		fmt.Println("Erro ao converter preço para float64 em momento de compra: ", err)
		return nil, 0
	}

	return nil, price

}

func ListarValorAnterior(coin string) (float64, error) {
	config.ReadFile()

	var historicos []models.HistoricoAll
	query := `SELECT * FROM hist_trading_values WHERE trading_name = ? AND hist_date >= NOW() - INTERVAL 5 MINUTE ORDER BY hist_date LIMIT 1`

	err := database.DB.Raw(query, coin).Scan(&historicos).Error
	if err != nil {
		return 0.0, err
	}
	if len(historicos) == 0 {
		return 0.0, errors.New("hist_trading_values retornou um array vazio")
	}
	priceFloat, err := strconv.ParseFloat(historicos[0].CurrentValue, 64)
	if err != nil {
		return 0.0, err
	}
	return priceFloat, nil
}
