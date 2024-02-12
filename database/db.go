package database

import (
	"binance_robot/config"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"log"
)

var (
	err error
	DB  *sqlx.DB
)

func DBCon() {
	fmt.Println("\nConectando ao MySQL...")
	config.ReadFile()
	con := config.User + ":" + config.Pass + "@tcp(" + config.Host + ":" + config.Port + ")/" + config.DBname
	DB, err = sqlx.Connect("mysql", con)
	if err != nil {
		log.Panic("Erro ao conectar com o banco de dados.")
	}

	err = DB.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Conexão com MySQL efetuada com sucesso!")
}

func GetDatabase() *sqlx.DB {
	return DB
}
