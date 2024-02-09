package listar_ordens

import (
	"binance_robot/config"
	"binance_robot/models"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func ListarOrdens(coin string) ([]models.CryptoPosition, error) {

	config.ReadFile()

	now := time.Now()
	timestamp := now.UnixMilli()

	apiParams := "symbol=" + coin + "" + config.BaseCoin + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(config.SecretKey, apiParams)

	url := config.BaseURL + "fapi/v2/positionRisk?" + apiParams + "&signature=" + signature

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response []models.CryptoPosition
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil

}
