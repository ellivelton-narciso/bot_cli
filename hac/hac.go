package hac

import (
	"candles/models"
	"math"
)

func HeikinAshi(previousHACandle models.Candle, currentCandle models.Candle) models.Candle {
	haClose := (currentCandle.Open + currentCandle.High + currentCandle.Low + currentCandle.Close) / 4
	haOpen := (previousHACandle.Open + previousHACandle.Close) / 2
	haHigh := math.Max(math.Max(currentCandle.High, haOpen), haClose)
	haLow := math.Min(math.Min(currentCandle.Low, haOpen), haClose)

	return models.Candle{
		Open:  haOpen,
		High:  haHigh,
		Low:   haLow,
		Close: haClose,
	}
}

func FirstHeikinAshi(firstCandle models.Candle) models.Candle {
	return HeikinAshi(firstCandle, firstCandle)
}
