package listar_ordens

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/models"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func ListarOrdens(coin string) ([]models.CryptoPosition, error) {

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()

	apiParams := "symbol=" + coin + "" + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)

	url := config.BaseURL + "fapi/v1/positionRisk?" + apiParams + "&signature=" + signature

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

func ListarUltimosValores(coin string, count int64) ([]models.HistoricoAll, error) {
	config.ReadFile()

	var historicos []models.HistoricoAll
	query := `SELECT * FROM hist_trading_values WHERE trading_name = ? ORDER BY hist_date desc  LIMIT ?`

	err := database.DB.Raw(query, coin, count).Scan(&historicos).Error
	if err != nil {
		return nil, err
	}
	if len(historicos) == 0 {
		return nil, errors.New("hist_trading_values retornou um array vazio")
	}
	return historicos, nil
}

func ListarUltimosValoresReais(coin string, count int64) []models.PriceResponse {
	config.ReadFile()

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

	return priceResp
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
