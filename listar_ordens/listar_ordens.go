package listar_ordens

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/models"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

func ListarOrdens(coin, apiKey, secretKey string) ([]models.CryptoPosition, error) {

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()

	apiParams := "symbol=" + coin + "" + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(secretKey, apiParams)

	url := config.BaseURL + "fapi/v2/positionRisk?" + apiParams + "&signature=" + signature

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", apiKey)

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

	return response, err

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

func ListarValorAnterior(coin, interval string) (float64, error) {
	config.ReadFile()

	var historicos []models.HistoricoAll
	query := `SELECT * FROM hist_trading_values WHERE trading_name = ? AND hist_date >= NOW() - INTERVAL ` + interval + ` MINUTE ORDER BY hist_date LIMIT 1`

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

func GetKlineData(symbol, interval string, limit int) ([]models.KlineData, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/klines?symbol=%s&interval=%s&limit=%d", symbol, interval, limit)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawData [][]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		fmt.Println(body)
		return nil, err
	}

	var klines []models.KlineData
	for _, data := range rawData {
		kline := models.KlineData{
			OpenTime:                 int64(data[0].(float64)),
			Open:                     StringToFloat64(data[1].(string)),
			High:                     StringToFloat64(data[2].(string)),
			Low:                      StringToFloat64(data[3].(string)),
			Close:                    StringToFloat64(data[4].(string)),
			Volume:                   StringToFloat64(data[5].(string)),
			CloseTime:                int64(data[6].(float64)),
			QuoteAssetVolume:         StringToFloat64(data[7].(string)),
			NumberOfTrades:           int(data[8].(float64)),
			TakerBuyBaseAssetVolume:  StringToFloat64(data[9].(string)),
			TakerBuyQuoteAssetVolume: StringToFloat64(data[10].(string)),
		}
		klines = append(klines, kline)
	}
	return klines, nil
}

func GetVolumeData(symbol, interval string, limit int) ([]models.VolumeData, error) {
	klines, err := GetKlineData(symbol, interval, limit)
	if err != nil {
		return nil, err
	}

	var volumeDataList []models.VolumeData
	for _, kline := range klines {
		sellVolume := kline.Volume - kline.TakerBuyBaseAssetVolume
		ratioVolume := 0.0
		if sellVolume != 0 { // Evita divisão por zero
			ratioVolume = (kline.TakerBuyBaseAssetVolume + 0.01) / sellVolume
		}

		volumeData := models.VolumeData{
			Volume:      kline.Volume,
			BuyVolume:   kline.TakerBuyBaseAssetVolume,
			SellVolume:  sellVolume,
			RatioVolume: ratioVolume,
		}
		volumeDataList = append(volumeDataList, volumeData)
	}

	return volumeDataList, nil
}

func StringToFloat64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatalf("Erro ao converter string para float64: %v", err)
	}
	return f
}
