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
				select
					fh.hist_date AS hist_date,
					fh.trading_name AS coin,
					(CASE WHEN (fh.trend_value > 0) THEN 'LONG' ELSE 'SHORT' END) AS tend,
					fh.curr_value AS curr_value,
					fh.target_perc AS SP,
					fh.target_perc AS SL,
					fh.other_value AS other_value
				from findings_history fh
				where fh.other_value IN (20, 21)
				  and fh.status = 'R'
				  and fh.trading_name NOT IN (SELECT bots.coin FROM bots)
				  and fh.hist_date > (NOW() - INTERVAL 2 MINUTE)
				order by fh.hist_date desc
			`).Scan(&bots).Error; err != nil {
				log.Println("Erro ao buscar dados da tabela "+config.ViewFiltro+":", err)
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
					if bot.SL < 0 {
						bot.SL = -(bot.SL)
					}
					if bot.SP < 0 {
						bot.SP = -(bot.SP)
					}
					if bot.Tend == "SHORT" {
						executarOrdem.OdemExecucao(bot.Coin, bot.Tend, control.Modo, control.Valor, control.Alavancagem, bot.SL, bot.SP, bot.OtherValue)
						return

					} else if bot.Tend == "LONG" {
						executarOrdem.OdemExecucao(bot.Coin, bot.Tend, control.Modo, control.Valor, control.Alavancagem, bot.SL, bot.SP, bot.OtherValue)
						return
					}
				}(bot)
			}
		}

		time.Sleep(2 * time.Second)
	}

}
