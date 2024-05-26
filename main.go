package main

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/executarOrdem"
	"binance_robot/models"
	"binance_robot/util"
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
			userKey := config.ApiKey[:5]
			bots = nil
			bots = util.BuscarValoresTelegram(userKey)
			if len(bots) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}
			log.Println("Capturado, ", bots)
			if control.Modo != "ISOLATED" && control.Modo != "CROSSED" {
				control.Modo = "ISOLATED"
			}

			for _, bot := range bots {
				go executarOrdem.OdemExecucao(bot.Coin, bot.Tend, control.Modo, control.Valor, control.Alavancagem, bot.SL, bot.SP, bot.OtherValue, config.ApiKey, config.SecretKey, userKey, true, true)
			}
		}

		time.Sleep(2 * time.Second)
	}

}
