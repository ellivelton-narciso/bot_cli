package main

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/executarOrdem"
	"binance_robot/models"
	"binance_robot/util"
	"fmt"
	"log"
	"time"
)

var bots []models.ResponseQuery

func main() {
	database.DBCon()
	config.ReadFile()
	userKey := config.ApiKey[:5]

	if config.ApiKey == "" || config.SecretKey == "" || config.BaseURL == "" {
		log.Panic("Arquivo user.json incompleto.")
	}
	deleteQry := "DELETE FROM " + config.Tabela + " WHERE user = ?"
	if err := database.DB.Exec(deleteQry, userKey).Error; err != nil {
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
			bots = util.BuscarValoresTelegram(userKey)
			if len(bots) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}
			if control.Modo != "ISOLATED" && control.Modo != "CROSSED" {
				control.Modo = "ISOLATED"
			}

			for _, bot := range bots {
				go executarOrdem.OdemExecucao(bot.Coin, bot.Tend, control.Modo, control.Valor, control.Alavancagem, bot.SL, bot.SP, bot.OtherValue, config.ApiKey, config.SecretKey, userKey, true, true, bot.CurrValue, bot.HistDate, config.UrlDisc)
				err := util.SendMessageToDiscord("["+bot.Coin+"] Entrada em "+bot.Tend+", "+fmt.Sprintf("%.6f", bot.CurrValue)+" - "+fmt.Sprintf("%.1f", bot.OtherValue), config.UrlDisc)
				if err != nil {
					log.Println("Erro ao iniicar.")
				}
			}
		}

		time.Sleep(2 * time.Second)
	}

}
