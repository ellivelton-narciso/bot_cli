package util

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"fmt"
	"gorm.io/gorm"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ConvertBaseCoin(coin string, value float64) (float64, float64) {
	var priceResp []models.PriceResponse
	config.ReadFile()

	priceResp = listar_ordens.ListarUltimosValoresReais(coin, 1)

	if len(priceResp) == 0 {
		return 0, 0
	}

	precision := 0

	if coin == "BTCUSDT" || coin == "ETHUSDT" {
		precision = 3
	}

	price, err := strconv.ParseFloat(priceResp[0].Price, 64)
	if err != nil {
		fmt.Println("Erro ao converter preço para float64: ", err)
	}

	q := value / price
	quantity := math.Round(q*math.Pow(10, float64(precision))) / math.Pow(10, float64(precision))

	return quantity, price
}

func Write(message, coin string) {
	filepath := "logs/log-" + coin

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	log.SetOutput(file)

	log.Println(stripColor(message))
	//fmt.Println(message)
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
	//fmt.Println(message, erro)
}
func WriteError(message string, erro error, coin string) {
	filepath := "logs/log-" + coin

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	log.SetOutput(file)

	log.Println(message, erro)

}

func stripColor(message string) string {
	regex := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return regex.ReplaceAllString(message, "")
}

func DefinirAlavancagem(currentCoin string, alavancagem float64) {
	now := time.Now()
	timestamp := now.UnixMilli()
	apiParams := "symbol=" + currentCoin + "&leverage=" + fmt.Sprint(alavancagem) + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)
	url := config.BaseURL + "fapi/v1/leverage?" + apiParams + "&signature=" + signature

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	Write(string(body), currentCoin)
}

func DefinirMargim(currentCoin, margim string) {
	now := time.Now()
	timestamp := now.UnixMilli()
	margim = strings.ToUpper(margim)
	apiParams := "symbol=" + currentCoin + "&marginType=" + margim + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)
	url := config.BaseURL + "fapi/v1/marginType?" + apiParams + "&signature=" + signature
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	Write(string(body), currentCoin)

}

func Historico(coin, side, started, parametros, currDateTelegram string, currValue, currValueTelegram, entryPrice, roi float64) {
	config.ReadFile()
	basecoin := coin
	count := contagemRows(basecoin, started)

	if count == 1 {
		query := "UPDATE hist_transactions SET " + parametros + " = ?, " + parametros + "_time = NOW(), " + parametros + "_roi = ? WHERE coin = ? AND started_at = ? AND side = ? AND " + parametros + " IS NULL"
		result := database.DB.Exec(query, currValue, roi, basecoin, started, side)
		if result.Error != nil {
			WriteError("Erro ao atualizar os parâmetros na tabela hist_transactions: ", result.Error, basecoin)
			return
		}
	} else {
		query := "INSERT INTO hist_transactions (coin, side, entryPrice, started_at, price_tg, date_tg) VALUES (?, ?, ?, ?, ?, ?)"
		result := database.DB.Exec(query, basecoin, side, entryPrice, started, currValueTelegram, currDateTelegram)
		if result.Error != nil {
			WriteError("Erro ao inserir dados iniciais da moeda na tabela hist_transactions: ", result.Error, basecoin)
			return
		}
	}
}

func EncerrarHistorico(coin, side, started string, currValue, roi float64) {
	count := contagemRows(coin, started)

	if count == 1 {
		query := "UPDATE hist_transactions SET final_price = ?, final_time = NOW(), final_roi = ? WHERE coin = ? AND started_at = ? AND side = ?"
		result := database.DB.Exec(query, currValue, roi, coin, started, side)
		if result.Error != nil {
			WriteError("Erro ao atualizar os parâmetros na tabela hist_transactions: ", result.Error, coin)
			return
		}
	}
}

func contagemRows(basecoin, started string) int {
	query := "SELECT COUNT(*) FROM hist_transactions WHERE coin = ? AND started_at = ?"

	var count int
	result := database.DB.Raw(query, basecoin, started).Scan(&count)
	if result.Error != nil {
		WriteError("Erro ao buscar a quantidade de linhas na tabela historico: ", result.Error, basecoin)
		return 0
	}
	return count
}
func BuscarValoresTelegram(coin string) []models.ResponseQuery {
	var bots []models.ResponseQuery

	database.DB.Where("coin = ?", coin).First(&bots)

	return bots

}

func GetPrecision(str string) int {
	parts := strings.Split(str, ".")

	if len(parts) == 1 {
		return 0
	}

	return len(parts[1])
}
