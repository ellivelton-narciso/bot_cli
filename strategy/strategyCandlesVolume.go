package strategy

import (
	"candles/database"
	"candles/global"
	"candles/hac"
	"candles/listar"
	"candles/models"
	"candles/util"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func ExecuteStrategy(symbol, key string, priority int) {
	if priority > 1 {
		time.Sleep(time.Duration(priority+2)*time.Minute + (2 * time.Second))
	}
	allCandles, err := GetAllC(symbol)
	if err != nil {
		log.Fatal("Erro ao buscar histórico de Candles", err)
	}

	for {
		if !global.OrdemAtiva || global.OrdemAtiva && global.CurrentCoin == symbol && !global.Meta {
			now := time.Now()
			nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, now.Second()+priority, 0, now.Location())
			time.Sleep(nextHour.Sub(now))
			// A mimir mais 20s
			time.Sleep(20*time.Second + time.Duration(priority)*time.Second)
			lastCandles, err := getC(symbol, "1h", 2)
			if err != nil {
				log.Fatal("Erro ao buscar histórico de Candles", err)
			}
			allCandles[len(allCandles)-1] = lastCandles[0]
			allCandles = append(allCandles, lastCandles[1])
			for try := 0; try < 3; try++ {
				haCandles := getHAC(symbol, 12, allCandles)
				if len(haCandles) < 2 {
					util.Write("["+symbol+"] Não há velas suficientes para executar a estratégia", symbol)
					break
				}
				previousCandle := haCandles[len(haCandles)-2]

				volumes, err := listar.GetVolumeData(symbol, "1h", 2)
				if err != nil {
					util.WriteError("["+symbol+"] Erro ao buscar dados do volume, ", err, symbol)
					break
				}

				util.Write("["+symbol+"] RatioAnterior: "+fmt.Sprintf("%.4f", volumes[0].RatioVolume), symbol+"-alert")
				util.Write("["+symbol+"] RatioAtual: "+fmt.Sprintf("%.4f", volumes[1].RatioVolume), symbol+"-alert")

				if previousCandle.Close > previousCandle.Open && volumes[0].RatioVolume > 1 && volumes[1].RatioVolume > 1 && previousCandle.Low == previousCandle.Open {
					if isTrendPositive(symbol, 5, allCandles) {
						if (global.OrdemAtiva && global.Side == "SELL") || !global.OrdemAtiva {
							util.Write("Vela Anterior HAC - OPEN: "+fmt.Sprintf("%.4f", previousCandle.Open)+", CLOSE: "+fmt.Sprintf("%.4f", previousCandle.Close)+", HIGH: "+fmt.Sprintf("%.4f", previousCandle.High)+", LOW: "+fmt.Sprintf("%.4f", previousCandle.Low), symbol)
							openLongPosition(symbol, key)
							break
						}
					}
				} else if previousCandle.Close < previousCandle.Open && volumes[0].RatioVolume < 1 && volumes[1].RatioVolume < 1 && previousCandle.High == previousCandle.Open {
					if isTrendNegative(symbol, 5, allCandles) {
						if (global.OrdemAtiva && global.Side == "BUY") || !global.OrdemAtiva {
							util.Write("Vela Anterior HAC - OPEN: "+fmt.Sprintf("%.4f", previousCandle.Open)+", CLOSE: "+fmt.Sprintf("%.4f", previousCandle.Close)+", HIGH: "+fmt.Sprintf("%.4f", previousCandle.High)+", LOW: "+fmt.Sprintf("%.4f", previousCandle.Low), symbol)
							openShortPosition(symbol, key)
							break
						}

					}
				}
				util.Write("["+symbol+"] Abr VelaAnterior: "+fmt.Sprintf("%.4f", previousCandle.Open)+", Fech Vela Anterior: "+fmt.Sprintf("%.4f", previousCandle.Close)+", High Vela Anterior: "+fmt.Sprintf("%.4f", previousCandle.High)+", Low Vela Anterior: "+fmt.Sprintf("%4.f", previousCandle.Low), symbol+"-alert")
				util.Write("["+symbol+"] RatioAnterior: "+fmt.Sprintf("%.4f", volumes[0].RatioVolume)+", VolumeCompra Anterior: "+fmt.Sprintf("%.4f", volumes[0].BuyVolume)+", VolumeVenda Anterior: "+fmt.Sprintf("%.4f", volumes[0].SellVolume), symbol+"-alert")
				util.Write("["+symbol+"] RatioAtual: "+fmt.Sprintf("%.4f", volumes[1].RatioVolume)+", VolumeCompra Atual: "+fmt.Sprintf("%.4f", volumes[1].BuyVolume)+", VolumeVenda Atual: "+fmt.Sprintf("%.4f", volumes[1].SellVolume), symbol+"-alert")
				time.Sleep(3 * time.Minute)
			}
			util.Write("["+symbol+"] Aguardando a proxima hora", symbol+"-alert")
		}

	}
}

func isTrendPositive(symbol string, intervals int, allC []models.Candle) bool {
	for i := 0; i < intervals; i++ {
		lastCandlesHAC := getHAC(symbol, 1, allC)
		volumes, err := listar.GetVolumeData(symbol, "5m", 1)
		if err != nil {
			util.WriteError("["+symbol+"] Erro ao buscar dados do volume, ", err, symbol)
		}
		util.Write("["+symbol+"] Pos Trend "+fmt.Sprint(i)+" - Abertura HAC: "+fmt.Sprintf("%.4f", lastCandlesHAC[0].Open)+", Fechamento HAC: "+fmt.Sprintf("%.4f", lastCandlesHAC[0].Close)+", Volume Ratio: "+fmt.Sprintf("%.4f", volumes[0].RatioVolume), symbol+"-alert")

		if lastCandlesHAC[0].Close < lastCandlesHAC[0].Open || volumes[0].RatioVolume < 1/1.25 {
			util.Write("["+symbol+"] Cancelado Pos Trend", symbol+"-alert")
			return false
		}
		time.Sleep(time.Minute)
	}
	util.Write("["+symbol+"] Pos Trend - "+fmt.Sprint(intervals)+" consecutivas", symbol)
	return true
}

func isTrendNegative(symbol string, intervals int, allC []models.Candle) bool {
	for i := 0; i < intervals; i++ {
		lastCandlesHAC := getHAC(symbol, 1, allC)
		volumes, err := listar.GetVolumeData(symbol, "5m", 1)
		if err != nil {
			util.WriteError("["+symbol+"] Erro ao buscar dados do volume, ", err, symbol)
		}
		util.Write("["+symbol+"] Neg Trend "+fmt.Sprint(i)+" - Abertura HAC: "+fmt.Sprintf("%.4f", lastCandlesHAC[0].Open)+", Fechamento HAC: "+fmt.Sprintf("%.4f", lastCandlesHAC[0].Close)+", Volume Ratio: "+fmt.Sprintf("%.4f", volumes[0].RatioVolume), symbol+"-alert")
		if lastCandlesHAC[0].Close > lastCandlesHAC[0].Open || volumes[0].RatioVolume > 1.25 {
			util.Write("["+symbol+"] Cancelado Neg Trend", symbol+"-alert")
			return false
		}
		time.Sleep(time.Minute)
	}
	util.Write("["+symbol+"] Neg Trend - "+fmt.Sprint(intervals)+" consecutivas", symbol)
	return true
}

func openLongPosition(symbol, key string) {
	if err := InsertAlert(key, symbol, "BUY"); err != nil {
		util.WriteError("["+symbol+"] Erro ao enviar alerta LONG para "+symbol+" key :"+key+" - ", err, symbol)
	}
}

func openShortPosition(symbol, key string) {
	if err := InsertAlert(key, symbol, "SELL"); err != nil {
		util.WriteError("["+symbol+"] Erro ao enviar alerta SHORT para "+symbol+" key :"+key+" - ", err, symbol)
	}
}

func InsertAlert(key, symbol, side string) error {
	err := database.DB.Exec("INSERT INTO alerts (user, symbol, side) VALUES (?, ?, ?)", key, symbol, side).Error
	if err != nil {
		util.WriteError("["+symbol+"] Erro ao inserir lerta de "+side+", ", err, symbol)
		return err
	}
	util.Write("["+symbol+"] Alerta de "+side+" inserido.", symbol)
	return nil
}

func getHAC(symbol string, limit int, allC []models.Candle) []models.Candle {
	lastCandle, _ := getC(symbol, "1h", 1)
	allC[len(allC)-1] = lastCandle[0]
	histCandles := allC
	var (
		haCandles []models.Candle
		haCandle  models.Candle
	)
	haCandle = hac.FirstHeikinAshi(histCandles[0])
	haCandles = append(haCandles, haCandle)

	for i := 1; i < len(histCandles); i++ {
		haCandle = hac.HeikinAshi(haCandles[i-1], histCandles[i])
		haCandles = append(haCandles, haCandle)
	}
	if limit < len(haCandles) {
		return haCandles[len(haCandles)-limit:]
	}
	return haCandles
}

func getC(symbol, interval string, limit int) ([]models.Candle, error) {
	klines, err := listar.GetKlineData(symbol, interval, limit)
	if err != nil {
		return nil, err
	}

	var candles []models.Candle
	for _, kline := range klines {
		candle := models.Candle{
			Open:  kline.Open,
			High:  kline.High,
			Low:   kline.Low,
			Close: kline.Close,
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

func GetAllC(symbol string) ([]models.Candle, error) {
	var candles []models.Candle
	var startTime int64
	var klines []models.KlineData

	startTime = 631152000000
	for {
		url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/klines?symbol=%s&interval=1h&limit=1500&startTime=%d", symbol, startTime)
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
			return nil, err
		}

		if len(rawData) == 0 {
			break
		}

		for _, data := range rawData {
			kline := models.KlineData{
				OpenTime:                 int64(data[0].(float64)),
				Open:                     listar.StringToFloat64(data[1].(string)),
				High:                     listar.StringToFloat64(data[2].(string)),
				Low:                      listar.StringToFloat64(data[3].(string)),
				Close:                    listar.StringToFloat64(data[4].(string)),
				Volume:                   listar.StringToFloat64(data[5].(string)),
				CloseTime:                int64(data[6].(float64)),
				QuoteAssetVolume:         listar.StringToFloat64(data[7].(string)),
				NumberOfTrades:           int(data[8].(float64)),
				TakerBuyBaseAssetVolume:  listar.StringToFloat64(data[9].(string)),
				TakerBuyQuoteAssetVolume: listar.StringToFloat64(data[10].(string)),
			}
			klines = append(klines, kline)
		}

		for _, kline := range klines {
			candle := models.Candle{
				Open:  kline.Open,
				High:  kline.High,
				Low:   kline.Low,
				Close: kline.Close,
			}
			candles = append(candles, candle)
		}
		startTime = klines[len(klines)-1].CloseTime + 1
		time.Sleep(15 * time.Second)
	}

	return candles, nil
}
