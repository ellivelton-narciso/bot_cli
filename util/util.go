package util

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/models"
	"encoding/json"
	"fmt"
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

func ConvertBaseCoin(coin string, value float64) float64 {

	config.ReadFile()

	url := config.BaseURL + "fapi/v1/ticker/price?symbol=" + coin + config.BaseCoin
	req, _ := http.NewRequest("GET", url, nil)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Erro ao acessar a API para converter: ", err)

	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
	}

	var priceResp models.PriceResponse
	err = json.Unmarshal(body, &priceResp)
	if err != nil {
		fmt.Println("Erro ao decodificar JSON:", err)
	}

	precision := 0

	if coin == "BTC" || coin == "ETH" {
		precision = 3
	}

	price, err := strconv.ParseFloat(priceResp.Price, 64)
	if err != nil {
		fmt.Println("Erro ao converter preço para float64:", err)
	}

	q := value / price
	quantity := math.Round(q*math.Pow(10, float64(precision))) / math.Pow(10, float64(precision))

	return quantity
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

func stripColor(message string) string {
	regex := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return regex.ReplaceAllString(message, "")
}

func DefinirAlavancagem(currentCoin string, alavancagem float64) {
	now := time.Now()
	timestamp := now.UnixMilli()
	apiParams := "symbol=" + currentCoin + config.BaseCoin + "&leverage=" + fmt.Sprint(alavancagem) + "&timestamp=" + strconv.FormatInt(timestamp, 10)
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
	fmt.Println(string(body))
}

func DefinirMargim(currentCoin, margim string) {
	now := time.Now()
	timestamp := now.UnixMilli()
	margim = strings.ToUpper(margim)
	apiParams := "symbol=" + currentCoin + config.BaseCoin + "&marginType=" + margim + "&timestamp=" + strconv.FormatInt(timestamp, 10)
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
	fmt.Println(string(body))

}

func Historico(coin, side, started, parametros string, currValue, entryPrice float64) {
	config.ReadFile()
	if config.Development {
		basecoin := coin + config.BaseCoin

		query := "SELECT COUNT(*) FROM hist_transactions WHERE coin = ? AND started_at = ?"
		var count int
		err := database.DB.Exec(query, basecoin, started).Scan(&count)
		if err != nil {
			log.Fatal(err)
		}

		if count > 0 {
			query = "UPDATE hist_transactions SET " + parametros + " = ?, " + parametros + "_time = NOW() WHERE coin = ? AND started_at = ? AND side = ?"
			err = database.DB.Exec(query, currValue, basecoin, started, side)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			query = "INSERT INTO hist_transactions (coin, side, entryPrice, started_at) VALUES (?, ?, ?, ?)"
			err = database.DB.Exec(query, basecoin, side, entryPrice, started)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
