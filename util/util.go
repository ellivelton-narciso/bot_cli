package util

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"encoding/json"
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

	price, err := strconv.ParseFloat(priceResp[0].Price, 64)
	if err != nil {
		WriteError("Erro ao converter preço para float64: ", err, coin)
	}

	precision, err := GetPrecision(coin)
	if err != nil {
		precision = 0
		WriteError("Erro ao buscar precisao para converter a moeda: ", err, coin)
	}

	if coin == "BTCUSDT" || coin == "ETHUSDT" || coin == "YFIUSDT" {
		precision = 3
	} else if coin == "BNBUSDT" {
		precision = 2
	} else if coin == "BSVUSDT" || coin == "ARUSDT" || price > 10 {
		precision = 1
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

func DefinirAlavancagem(currentCoin string, alavancagem float64) error {
	now := time.Now()
	timestamp := now.UnixMilli()
	apiParams := "symbol=" + currentCoin + "&leverage=" + fmt.Sprint(alavancagem) + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)
	url := config.BaseURL + "fapi/v1/leverage?" + apiParams + "&signature=" + signature

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	if res.StatusCode != 200 {
		Write(string(body), currentCoin)
	}
	return nil
}

func DefinirMargim(currentCoin, margim string) error {
	now := time.Now()
	timestamp := now.UnixMilli()
	margim = strings.ToUpper(margim)
	apiParams := "symbol=" + currentCoin + "&marginType=" + margim + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)
	url := config.BaseURL + "fapi/v1/marginType?" + apiParams + "&signature=" + signature
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	if res.StatusCode != 200 {
		Write(string(body), currentCoin)
	}
	return nil
}

func Historico(coin, side, started, parametros, currDateTelegram string, currValue, currValueTelegram, entryPrice, roi float64) {
	config.ReadFile()
	basecoin := coin
	count := contagemRows(basecoin, started)

	if count == 1 {
		query := "UPDATE " + config.TabelaHist + " SET " + parametros + " = ?, " + parametros + "_time = NOW(), " + parametros + "_roi = ? WHERE coin = ? AND started_at = ? AND side = ? AND " + parametros + " IS NULL"
		result := database.DB.Exec(query, currValue, roi, basecoin, started, side)
		if result.Error != nil {
			WriteError("Erro ao atualizar os parâmetros na tabela hist_transactions: ", result.Error, basecoin)
			return
		}
	} else {
		query := "INSERT INTO " + config.TabelaHist + " (coin, side, entryPrice, started_at, price_tg, date_tg) VALUES (?, ?, ?, ?, ?, ?)"
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
		query := "UPDATE " + config.TabelaHist + " SET final_price = ?, final_time = NOW(), final_roi = ? WHERE coin = ? AND started_at = ? AND side = ?"
		result := database.DB.Exec(query, currValue, roi, coin, started, side)
		if result.Error != nil {
			WriteError("Erro ao atualizar os parâmetros na tabela hist_transactions: ", result.Error, coin)
			return
		}
	}
}

func contagemRows(basecoin, started string) int {
	query := "SELECT COUNT(*) FROM " + config.TabelaHist + " WHERE coin = ? AND started_at = ?"

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

func GetPrecision(currentCoin string) (int, error) {
	url := "https://fapi.binance.com/fapi/v1/ticker/bookTicker?symbol=" + currentCoin
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	Write(string(body), currentCoin)

	var response models.ResponseBookTicker
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, err
	}
	parts := strings.Split(response.BidQty, ".")
	if len(parts) == 1 {
		return 0, nil
	}
	precision := len(parts[1])
	return precision, nil
}
