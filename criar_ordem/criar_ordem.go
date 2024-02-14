package criar_ordem

import (
	"binance_robot/config"
	"binance_robot/database"
	"binance_robot/listar_ordens"
	"binance_robot/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func CriarOrdem(coin string, side string, positionSide string, quantity string) (string, error) {

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()
	apiParamsOrdem := "symbol=" + coin + "" + config.BaseCoin + "&type=MARKET&side=" + side + "&quantity=" + quantity + "&positionSide=" + positionSide + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)

	urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem

	req, err := http.NewRequest("POST", urlOrdem, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response models.ResponseOrderStruct
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	allOrders, _ := listar_ordens.ListarOrdens(coin)
	var entryPrice string
	for _, item := range allOrders {
		if item.PositionSide == side {
			entryPrice = item.EntryPrice
		}
	}

	return entryPrice, err
}

func EnviarCoinDB(coin string) {
	config.ReadFile()

	basecoin := coin + config.BaseCoin

	var bot models.Bots
	result := database.DB.Where("coin = ?", basecoin).First(&bot)
	if result.RowsAffected > 0 {
		return
	}

	if err := database.DB.Create(&models.Bots{Coin: basecoin}).Error; err != nil {
		fmt.Println("\n Erro ao inserir coin na DB: ", err)
	}

	return
}

func RemoverCoinDB(coin string) error {
	config.ReadFile()

	basecoin := coin + config.BaseCoin

	if err := database.DB.Where("coin = ?", basecoin).Delete(&models.Bots{}).Error; err != nil {
		fmt.Println("\n Erro ao remover coin na DB: ", err)
		return err
	}
	return nil
}
