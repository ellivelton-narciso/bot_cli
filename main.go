package main

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/executarOrdem"
	"binance_robot/models"
	"log"
	"time"
)

var bots []models.ResponseQuery

func main() {
	database.DBCon()
	config.ReadFile()

	if config.ApiKey == "" || config.SecretKey == "" || config.BaseURL == "" {
		log.Panic("Arquivo user.json incompleto.")
	}

	for {
		bots = nil
		if err := database.DB.Raw(`
			SELECT 
				so.hist_date,
				so.coin,
				CASE 
					WHEN tp.total_win / tp.total <= 0.25 THEN (CASE WHEN so.tend = 'LONG' THEN 'SHORT' ELSE 'LONG' END)
					ELSE tend 
				END AS tend,
				so.curr_value,
				so.SP,
				so.SL,
				so.other_value
			FROM v_selected_orders so
				join (SELECT trading_name,
					 TIPO_ALERTA,
					  trend,
					  ROUND(total_win / total * 100, 2) perc_win,
					  total_win,
					  total
				FROM (SELECT TIPO_ALERTA,
							trend,
							trading_name,
							SUM(CASE WHEN status = 'W' THEN 1 ELSE 0 END) AS total_win,
							COUNT(1) AS total
					 FROM (SELECT ROUND(other_value) AS TIPO_ALERTA,
								  trading_name,
								  (CASE WHEN trend_value > 0 THEN 'LONG' ELSE 'SHORT' END) AS trend,
								  status
						   FROM findings_history a
						   WHERE close_date > NOW() - INTERVAL 2 DAY
							AND status IN ('W', 'L')
							AND other_value IN (select ROUND(alert) as alert from bot_control GROUP BY alert)) x
					 GROUP BY TIPO_ALERTA, trading_name, trend
					 HAVING COUNT(1) >= 5) z
						join bot_control bc ON bc.status = 'A'
				WHERE bc.alert = z.TIPO_ALERTA
				 AND bc.side = z.trend
				  AND (total_win / total >= 0.80 OR total_win / total <= 0.25)
				ORDER BY perc_win DESC, total_win DESC) tp ON tp.trading_name = so.coin
			WHERE tp.trend = so.tend
			AND tp.TIPO_ALERTA = ROUND(so.other_value)
		`).Scan(&bots).Error; err != nil {
			log.Println("Erro ao buscar dados da tabela v_selected_orders:", err)
			continue
		}
		if len(bots) == 0 {
			continue
		}

		for _, bot := range bots {
			go func(bot models.ResponseQuery) {
				if bot.SL < 0 {
					bot.SL = -(bot.SL)
				}
				if bot.SP < 0 {
					bot.SP = -(bot.SP)
				}
				if bot.Tend == "SHORT" {
					executarOrdem.OdemExecucao(bot.Coin, bot.Tend, config.Value, config.Alavancagem, bot.SL, bot.SP)
					return

				} else if bot.Tend == "LONG" {
					executarOrdem.OdemExecucao(bot.Coin, bot.Tend, config.Value, config.Alavancagem, bot.SL, bot.SP)
					return
				}
			}(bot)
		}
		time.Sleep(2 * time.Second)
	}

}
