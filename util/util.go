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

	precision := GetPrecision(priceResp.Price)

	price, err := strconv.ParseFloat(priceResp.Price, 64)
	if err != nil {
		fmt.Println("Erro ao converter preço para float64:", err)
	}

	q := value / price
	quantity := math.Round(q*math.Pow(10, float64(precision))) / math.Pow(10, float64(precision))

	return quantity
}

func GetPrecision(number string) int {
	parts := strings.Split(number, ".")
	if len(parts) == 2 {
		if len(parts[1]) > 4 {
			return 4
		} else {
			return len(parts[1])
		}
	}
	return 0
}

func removerZeros(number string) string {
	var newValue string
	foundNonZero := false
	for i := len(number) - 1; i >= 0; i-- {
		if number[i] != '0' {
			foundNonZero = true
		}
		if foundNonZero {
			newValue = string(number[i]) + newValue
		}
	}
	return newValue
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
	fmt.Println(message)
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

func SalvarHistorico(coin string, command string, commandParams string, currValue float64, accumRoi float64) error {

	if !config.Development {
		config.ReadFile()
		basecoin := coin + config.BaseCoin

		query := fmt.Sprintf("INSERT INTO bot_history (account_key, hist_date, curr_value, command, commmand_params, accum_roi, trading_name) VALUES ('%s', NOW(), %f, '%s', '%s', %f, '%s')",
			config.ApiKey, currValue, command, commandParams, accumRoi, basecoin)
		if err := database.DB.Exec(query).Error; err != nil {
			return err
		}

		return nil
	} else {
		return nil
	}
}
