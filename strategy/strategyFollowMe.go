package strategy

import (
	"candles/database"
	"candles/models"
	"candles/util"
	"fmt"
	"log"
	"time"
)

func ExecuteStrategyFM() {
	for {

		time.Sleep(1 * time.Minute)
	}
}

func GetCm(symbol, interval string, minuteInterval int) []models.Candle {
	var histCandles []models.Candle

	sqlQuery := fmt.Sprintf(`
		SELECT
            trading_name,
            DATE_FORMAT(DATE_SUB(hist_date,INTERVAL MOD(MINUTE(hist_date), %d) MINUTE),'%%Y-%%m-%%d %%H:%%i:00') AS minute_start,
            SUBSTRING_INDEX(GROUP_CONCAT(curr_value ORDER BY hist_date ASC), ',', 1) AS open,
            MAX(curr_value) AS high,
            MIN(curr_value) AS low,
            SUBSTRING_INDEX(GROUP_CONCAT(curr_value ORDER BY hist_date DESC), ',', 1) AS close
        FROM
            hist_trading_values
        WHERE
            trading_name = ?
          AND hist_date >= NOW() - INTERVAL %s
        GROUP BY
            trading_name, minute_start
	`, minuteInterval, interval)

	candles := database.DB.Raw(sqlQuery, symbol).Scan(&histCandles)

	if candles.Error != nil {
		log.Println("Erro ao buscar valores das velas:", candles.Error)
		return nil
	}

	if len(histCandles) > 2 {
		histCandles = histCandles[1:]
	}
	return histCandles
}

func CalcularMediaDeMovimento(candles []models.Candle) (float64, float64, float64) {
	var sumUp, sumDown float64
	var countUp, countDown int

	for _, candle := range candles {
		change := candle.Close - candle.Open
		if change > 0 {
			sumUp += change
			countUp++
		} else if change < 0 {
			sumDown += change
			countDown++
		}
	}

	var avgUp, avgDown, avgAll float64
	if countUp > 0 {
		avgUp = sumUp / float64(countUp)
	}
	if countDown > 0 {
		avgDown = sumDown / float64(countDown)
	}
	totalCount := countUp + countDown
	if totalCount > 0 {
		avgAll = (sumUp + (-(sumDown))) / float64(totalCount)
	}
	avgUp = util.RoundToPrecision(avgUp, 4)
	avgDown = util.RoundToPrecision(avgDown, 4)
	avgAll = util.RoundToPrecision(avgAll, 4)

	return avgUp, avgDown, avgAll
}
