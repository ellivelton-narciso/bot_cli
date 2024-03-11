package criar_ordem

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/models"
	"binance_robot/util"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func CriarOrdem(coin string, side string, quantity string) (int, error) {

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()
	apiParamsOrdem := "symbol=" + coin + "&type=MARKET&side=" + side + "&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)

	urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem

	req, err := http.NewRequest("POST", urlOrdem, nil)
	if err != nil {
		return 500, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 500, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 500, err
	}

	if res.StatusCode != 200 {
		util.Write(string(body), coin)
	}
	//fmt.Println(string(body))
	//fmt.Println(res.StatusCode)

	var response models.ResponseOrderStruct
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 500, err
	}
	return res.StatusCode, err
}

func EnviarCoinDB(coin string) {
	var bot models.Bots
	result := database.DB.Where("coin = ?", coin).First(&bot)
	if result.RowsAffected > 0 {
		return
	}

	if err := database.DB.Create(&models.Bots{Coin: coin}).Error; err != nil {
		fmt.Println("\n Erro ao inserir coin na DB: ", err)
	}

	return
}

func RemoverCoinDB(coin string) error {
	if err := database.DB.Where("coin = ?", coin).Delete(&models.Bots{}).Error; err != nil {
		fmt.Println("\n Erro ao remover coin na DB: ", err)
		return err
	}
	return nil
}
