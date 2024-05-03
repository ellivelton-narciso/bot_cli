package util

import (
	"candles/config"
	"candles/global"
	"log"
	"time"
)

func ControlRoutine() {
	config.ReadFile()
	for {
		now := time.Now()
		if now.Hour() == config.WakeUp && !global.OrdemAtiva {
			global.Meta = false
			global.ValueCompradoCoin = 0.0
			global.ForTime = time.Second
			global.NextValue = global.Value
			log.Println("Acordei!")
		}
		nextDay := now.AddDate(0, 0, 1)
		nextDay = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), config.WakeUp, 0, 0, 0, now.Location())
		time.Sleep(nextDay.Sub(now))
	}
}

func ControlSleep() {
	config.ReadFile()
	for {
		now := time.Now()
		if now.Hour() >= config.Sleep && now.Hour() < config.WakeUp &&
			!global.OrdemAtiva && !global.Meta &&
			config.WakeUp != config.Sleep {
			global.Meta = true
			log.Println("Dormindo...")
		}
		time.Sleep(time.Hour)
	}
}
