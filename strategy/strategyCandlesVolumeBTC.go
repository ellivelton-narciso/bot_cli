package strategy

import (
	"candles/global"
	"candles/listar"
	"candles/models"
	"candles/util"
	"fmt"
	"log"
	"time"
)

func ExecuteStrategy2(symbol, key string, priority int) {
	if priority > 1 {
		time.Sleep(time.Duration(priority+2)*time.Minute + (2 * time.Second))
	}
	allCandles, err := GetAllC(symbol)
	if err != nil {
		log.Fatal("Erro ao buscar histórico de Candles", err)
	}
	if priority == 1 {
		global.AllCandlesBTC, err = GetAllC("BTCUSDT")
		if err != nil {
			log.Fatal("Erro ao buscar histórico de Candles", err)
		}
	}

	for {

		now := time.Now()
		nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, now.Second()+priority, 0, now.Location())
		time.Sleep(nextHour.Sub(now))
		// A mimir mais 20s
		time.Sleep(20 * time.Second)
		lastCandles, err := getC(symbol, "1h", 2)
		if err != nil {
			log.Fatal("Erro ao buscar histórico de Candles", err)
		}
		allCandles[len(allCandles)-1] = lastCandles[0]
		allCandles = append(allCandles, lastCandles[1])

		lastCandlesBTC, err := getC("BTCUSDT", "1h", 2)
		if err != nil {
			log.Fatal("Erro ao buscar histórico de Candles", err)
		}
		if priority == 1 {
			global.AllCandlesBTC[len(global.AllCandlesBTC)-1] = lastCandlesBTC[0]
			global.AllCandlesBTC = append(global.AllCandlesBTC, lastCandlesBTC[1])
		}
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

			lastCandlesBTC, err = getC("BTCUSDT", "1h", 1)
			if err != nil {
				log.Println("Erro ao buscar histórico de Candles", err)
				break
			}
			volumesBTC, err := listar.GetVolumeData(symbol, "1h", 2)
			if err != nil {
				util.WriteError("["+symbol+"] Erro ao buscar dados do volume, ", err, symbol)
				break
			}

			util.Write("["+symbol+"] RatioAnterior: "+fmt.Sprintf("%.4f", volumes[0].RatioVolume), symbol+"-alert")
			util.Write("["+symbol+"] RatioAtual: "+fmt.Sprintf("%.4f", volumes[1].RatioVolume), symbol+"-alert")

			if previousCandle.Close > previousCandle.Open && volumes[0].RatioVolume > 1 && volumes[1].RatioVolume > 1 && previousCandle.Low == previousCandle.Open && lastCandlesBTC[0].Close > lastCandlesBTC[0].Open && volumesBTC[0].RatioVolume > 1 && volumesBTC[1].RatioVolume > 1 {
				if isTrendPositiveBTC(symbol, 5, allCandles) {
					lastCandles, err = getC(symbol, "1h", 1)
					if err != nil {
						log.Fatal("Erro ao buscar histórico de Candles", err)
					}
					util.Write("Vela Anterior HAC - OPEN: "+fmt.Sprintf("%.4f", previousCandle.Open)+", CLOSE: "+fmt.Sprintf("%.4f", previousCandle.Close)+", HIGH: "+fmt.Sprintf("%.4f", previousCandle.High)+", LOW: "+fmt.Sprintf("%.4f", previousCandle.Low), symbol)
					openLongPosition(symbol, key)
					break
				}
			} else if previousCandle.Close < previousCandle.Open && volumes[0].RatioVolume < 1 && volumes[1].RatioVolume < 1 && previousCandle.High == previousCandle.Open && lastCandlesBTC[0].Close < lastCandlesBTC[0].Open && volumesBTC[0].RatioVolume < 1 && volumesBTC[1].RatioVolume < 1 {
				if isTrendNegativeBTC(symbol, 5, allCandles) {
					lastCandles, err = getC(symbol, "1h", 1)
					if err != nil {
						log.Fatal("Erro ao buscar histórico de Candles", err)
					}
					util.Write("Vela Anterior HAC - OPEN: "+fmt.Sprintf("%.4f", previousCandle.Open)+", CLOSE: "+fmt.Sprintf("%.4f", previousCandle.Close)+", HIGH: "+fmt.Sprintf("%.4f", previousCandle.High)+", LOW: "+fmt.Sprintf("%.4f", previousCandle.Low), symbol)
					openShortPosition(symbol, key)
					break
				}
			}
			util.Write("["+symbol+"] Abr VelaAnterior: "+fmt.Sprintf("%.4f", previousCandle.Open)+", Fech Vela Anterior: "+fmt.Sprintf("%.4f", previousCandle.Close)+", High Vela Anterior: "+fmt.Sprintf("%.4f", previousCandle.High)+", Low Vela Anterior: "+fmt.Sprintf("%4.f", previousCandle.Low), symbol+"-alert")
			util.Write("["+symbol+"] RatioAnterior: "+fmt.Sprintf("%.4f", volumes[0].RatioVolume)+", VolumeCompra Anterior: "+fmt.Sprintf("%.4f", volumes[0].BuyVolume)+", VolumeVenda Anterior: "+fmt.Sprintf("%.4f", volumes[0].SellVolume), symbol+"-alert")
			util.Write("["+symbol+"] RatioAtual: "+fmt.Sprintf("%.4f", volumes[1].RatioVolume)+", VolumeCompra Atual: "+fmt.Sprintf("%.4f", volumes[1].BuyVolume)+", VolumeVenda Atual: "+fmt.Sprintf("%.4f", volumes[1].SellVolume), symbol+"-alert")
			time.Sleep(3 * time.Minute)
		}
		//message.SendAll(fmt.Sprintf("[" + symbol + "] Aguardando a proxima hora"))
		util.Write("["+symbol+"] Aguardando a proxima hora", symbol+"-alert")

	}
}

func isTrendPositiveBTC(symbol string, intervals int, allC []models.Candle) bool {
	for i := 0; i < intervals; i++ {
		lastCandlesHAC := getHAC(symbol, 1, allC)
		volumes, err := listar.GetVolumeData(symbol, "5m", 1)
		if err != nil {
			util.WriteError("["+symbol+"] Erro ao buscar dados do volume, ", err, symbol)
		}

		lastCandlesBTC := getHAC("BTCUSDT", 1, global.AllCandlesBTC)
		volumesBTC, err := listar.GetVolumeData(symbol, "5m", 1)
		if err != nil {
			util.WriteError("["+symbol+"] Erro ao buscar dados do volume, ", err, symbol)
			break
		}

		util.Write("["+symbol+"] Pos Trend "+fmt.Sprint(i)+" - Abertura HAC: "+fmt.Sprintf("%.4f", lastCandlesHAC[0].Open)+", Fechamento HAC: "+fmt.Sprintf("%.4f", lastCandlesHAC[0].Close)+", Volume Ratio: "+fmt.Sprintf("%.4f", volumes[0].RatioVolume), symbol+"-alert")
		util.Write("[BTCUSDT - Comp] Neg Trend "+fmt.Sprint(i)+" - Abertura HAC: "+fmt.Sprintf("%.4f", lastCandlesBTC[0].Open)+", Fechamento HAC: "+fmt.Sprintf("%.4f", lastCandlesBTC[0].Close)+", Volume Ratio: "+fmt.Sprintf("%.4f", volumesBTC[0].RatioVolume), symbol+"-alert")
		if lastCandlesHAC[0].Close < lastCandlesHAC[0].Open || volumes[0].RatioVolume < 1/1.25 || lastCandlesBTC[0].Close < lastCandlesBTC[0].Open || volumesBTC[0].RatioVolume < 1/1.25 {
			util.Write("["+symbol+"] Cancelado Pos Trend", symbol+"-alert")
			return false
		}
		time.Sleep(time.Minute)
	}
	util.Write("["+symbol+"] Pos Trend - "+fmt.Sprint(intervals)+" consecutivas", symbol)
	return true
}

func isTrendNegativeBTC(symbol string, intervals int, allC []models.Candle) bool {
	for i := 0; i < intervals; i++ {
		lastCandlesHAC := getHAC(symbol, 1, allC)
		volumes, err := listar.GetVolumeData(symbol, "5m", 1)
		if err != nil {
			util.WriteError("["+symbol+"] Erro ao buscar dados do volume, ", err, symbol)
		}

		lastCandlesBTC := getHAC("BTCUSDT", 1, global.AllCandlesBTC)
		volumesBTC, err := listar.GetVolumeData(symbol, "5m", 1)
		if err != nil {
			util.WriteError("["+symbol+"] Erro ao buscar dados do volume, ", err, symbol)
			break
		}

		util.Write("["+symbol+"] Neg Trend "+fmt.Sprint(i)+" - Abertura HAC: "+fmt.Sprintf("%.4f", lastCandlesHAC[0].Open)+", Fechamento HAC: "+fmt.Sprintf("%.4f", lastCandlesHAC[0].Close)+", Volume Ratio: "+fmt.Sprintf("%.4f", volumes[0].RatioVolume), symbol+"-alert")
		util.Write("[BTCUSDT - Comp] Neg Trend "+fmt.Sprint(i)+" - Abertura HAC: "+fmt.Sprintf("%.4f", lastCandlesBTC[0].Open)+", Fechamento HAC: "+fmt.Sprintf("%.4f", lastCandlesBTC[0].Close)+", Volume Ratio: "+fmt.Sprintf("%.4f", volumesBTC[0].RatioVolume), symbol+"-alert")
		if lastCandlesHAC[0].Close > lastCandlesHAC[0].Open || volumes[0].RatioVolume > 1.25 || lastCandlesBTC[0].Close > lastCandlesBTC[0].Open || volumesBTC[0].RatioVolume > 1.25 {
			util.Write("["+symbol+"] Cancelado Neg Trend", symbol+"-alert")
			return false
		}
		time.Sleep(time.Minute)
	}
	util.Write("["+symbol+"] Neg Trend - "+fmt.Sprint(intervals)+" consecutivas", symbol)
	return true
}
