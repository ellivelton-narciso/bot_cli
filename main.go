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
		if err := database.DB.Find(&bots).Error; err != nil {
			log.Println("Erro ao buscar dados da tabela v_selected_orders:", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if len(bots) == 0 {
			time.Sleep(1 * time.Second)
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
					executarOrdem.OdemExecucao(bot.Coin, bot.Tend, config.Value, config.Alavancagem, bot.SL, bot.SP, bot.OtherValue)
					return

				} else if bot.Tend == "LONG" {
					executarOrdem.OdemExecucao(bot.Coin, bot.Tend, config.Value, config.Alavancagem, bot.SL, bot.SP, bot.OtherValue)
					return
				}
			}(bot)
		}
		time.Sleep(2 * time.Second)
	}

}
