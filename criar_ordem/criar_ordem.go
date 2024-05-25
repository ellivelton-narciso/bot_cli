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

func CriarOrdem(coin, side, quantity, posSide, apiKey, secretKey string) (int, error) {
	config.ReadFile()
	now := time.Now()
	timestamp := now.UnixMilli()
	apiParamsOrdem := "symbol=" + coin + "&type=MARKET&side=" + side + "&positionSide=" + posSide + "&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signatureOrdem := config.ComputeHmacSha256(secretKey, apiParamsOrdem)

	urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem

	req, err := http.NewRequest("POST", urlOrdem, nil)
	if err != nil {
		return 500, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", apiKey)

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
	return res.StatusCode, nil
}

func EnviarCoinDB(coin string) {
	var bot models.Bots
	result := database.DB.Where("symbol = ?", coin).First(&bot)
	if result.RowsAffected > 0 {
		return
	}

	if err := database.DB.Create(&models.Bots{Symbol: coin}).Error; err != nil {
		fmt.Println("\n Erro ao inserir coin na DB: ", err)
	}
	util.Write("Inserido na tabela bots", coin)
	return
}

func RemoverCoinDB(coin string, tempo time.Duration) error {
	time.Sleep(tempo)
	if err := database.DB.Where("symbol = ?", coin).Delete(&models.Bots{}).Error; err != nil {
		util.WriteError("\n Erro ao remover coin na DB: ", err, coin)
		return err
	}
	util.Write("Removido da tabela bots", coin)
	return nil
}

func CriarSLSeguro(coin, side, stop, posSide, apiKey, secretKey string) (int, string, error) {
	config.ReadFile()

	if !config.Development {
		now := time.Now()
		timestamp := now.UnixMilli()
		apiParamsOrdem := "symbol=" + coin + "&type=STOP_MARKET&side=" + side + "&positionSide=" + posSide + "&closePosition=true&stopPrice=" + stop + "&timestamp=" + strconv.FormatInt(timestamp, 10)
		signatureOrdem := config.ComputeHmacSha256(secretKey, apiParamsOrdem)

		urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem

		req, err := http.NewRequest("POST", urlOrdem, nil)
		if err != nil {
			return 500, "", err
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("X-MBX-APIKEY", apiKey)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return 500, "", err
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return 500, string(body), err
		}

		if res.StatusCode != 200 {
			return res.StatusCode, string(body), nil
		}
		//fmt.Println(string(body))
		//fmt.Println(res.StatusCode)

		var response models.ResponseOrderStruct
		err = json.Unmarshal(body, &response)
		if err != nil {
			return 500, string(body), err
		}
		return res.StatusCode, string(body), nil
	} else {
		return 200, "", nil
	}
}

func CancelarSLSeguro(coin, apiKey, secretKey string) (int, error) {
	config.ReadFile()

	if !config.Development {
		now := time.Now()
		timestamp := now.UnixMilli()
		apiParamsOrdem := "symbol=" + coin + "&timestamp=" + strconv.FormatInt(timestamp, 10)
		signatureOrdem := config.ComputeHmacSha256(secretKey, apiParamsOrdem)

		urlOrdem := config.BaseURL + "fapi/v1/allOpenOrders?" + apiParamsOrdem + "&signature=" + signatureOrdem
		req, err := http.NewRequest("DELETE", urlOrdem, nil)
		if err != nil {
			return 500, err
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("X-MBX-APIKEY", apiKey)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return 500, err
		}

		if res.StatusCode != 200 {
			return res.StatusCode, nil
		}
		return res.StatusCode, nil
	} else {
		return 200, nil
	}
}
