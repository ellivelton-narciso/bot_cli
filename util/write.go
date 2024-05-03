package util

import (
	"candles/global"
	"fmt"
	"gorm.io/gorm"
	"log"
	"os"
	"regexp"
	"time"
)

func Write(message, coin string) {
	filepath := "logs/log-" + coin

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	now := time.Now()
	timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
	formattedTime := timeValue.Format("2006-01-02 15:04:05")

	log.SetOutput(file)

	log.Println(stripColor(message))
	fmt.Println(formattedTime + " - " + message)
}
func WriteErrorDB(message string, erro *gorm.DB, coin string) {
	filepath := "logs/log-" + coin

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	log.SetOutput(file)

	log.Println(message, erro)
	if !global.CmdRun {
		fmt.Println(message, erro)
	}
}
func WriteError(message string, erro error, coin string) {
	filepath := "logs/log-" + coin

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	now := time.Now()
	timeValue := time.Unix(0, now.UnixMilli()*int64(time.Millisecond))
	formattedTime := timeValue.Format("2006-01-02 15:04:05")

	log.SetOutput(file)

	log.Println(message, erro)
	if !global.CmdRun {
		fmt.Println(formattedTime+" - "+message, erro)
	}

}
func stripColor(message string) string {
	regex := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return regex.ReplaceAllString(message, "")
}
