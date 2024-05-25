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
	deleteQry := "DELETE FROM " + config.Tabela
	if err := database.DB.Exec(deleteQry).Error; err != nil {
		log.Println("Erro ao limpar tabela bots", err)
	}

	for {
		var control models.Control
		controle := database.DB.Raw("SELECT * FROM money_bot").First(&control)
		if controle.Error != nil {
			log.Println("Erro ao buscar control do bot")
			continue
		}
		if config.Development {
			control.Ativo = "A"
			control.Valor = config.Value
			control.Alavancagem = config.Alavancagem
			control.Modo = "ISOLATED"
		}
		if control.Ativo == "A" {
			bots = nil
			if err := database.DB.Raw(`
				select hist_date,
					   trading_name                                                             coin,
					   case when trend_value > 0 then 'LONG' else 'SHORT' end                   tend,
					   curr_value,
					   target_perc                                                              SP,
					   target_perc                                                              SL,
					   other_value
				from findings_history
				where other_value >= 200 and other_value < 300
				  and trading_name not in (select symbol from bots_real)
				  and status = 'R'
				  AND hist_date > (NOW() - INTERVAL 1 MINUTE)
				order by hist_date
			`).Scan(&bots).Error; err != nil {
				log.Println("Erro ao buscar dados da query:", err)
				time.Sleep(5 * time.Second)
				continue
			}
			if len(bots) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}
			log.Println("Capturado, ", bots)
			if control.Modo != "ISOLATED" && control.Modo != "CROSSED" {
				control.Modo = "ISOLATED"
			}

			for _, bot := range bots {
				go func(bot models.ResponseQuery) {
					executarOrdem.OdemExecucao(bot.HistDate, bot.Coin, bot.Tend, control.Modo, control.Valor, control.Alavancagem, bot.SL, bot.SP, bot.OtherValue, config.ApiKey, config.SecretKey, true, bot.CurrValue)
					return
				}(bot)
			}
		}

		time.Sleep(1 * time.Second)
	}

}
