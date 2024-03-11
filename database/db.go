package database

import (
	"binance_robot/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
)

var (
	err error
	DB  *gorm.DB
)

func DBCon() {
	//fmt.Println("\nConectando ao MySQL...")
	config.ReadFile()
	con := config.User + ":" + config.Pass + "@tcp(" + config.Host + ":" + config.Port + ")/" + config.DBname + "?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(con), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Panic("Erro ao conectar com o banco de dados.")
	}

	//fmt.Println("Conexão com MySQL efetuada com sucesso!")
}

func GetDatabase() *gorm.DB {
	return DB
}
