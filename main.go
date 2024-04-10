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
		}
		if control.Ativo == "A" {
			bots = nil
			if err := database.DB.Find(&bots).Error; err != nil {
				log.Println("Erro ao buscar dados da tabela "+config.ViewFiltro+":", err)
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
						executarOrdem.OdemExecucao(bot.Coin, bot.Tend, control.Valor, control.Alavancagem, bot.SL, bot.SP, bot.OtherValue)
						return

					} else if bot.Tend == "LONG" {
						executarOrdem.OdemExecucao(bot.Coin, bot.Tend, control.Valor, control.Alavancagem, bot.SL, bot.SP, bot.OtherValue)
						return
					}
				}(bot)
			}
		}

		time.Sleep(2 * time.Second)
	}

}
