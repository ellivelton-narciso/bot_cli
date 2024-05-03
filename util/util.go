package util

import (
	"candles/config"
	"candles/database"
	"candles/global"
	"candles/listar"
	"candles/models"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func ConvertBaseCoin(coin string, value float64) (float64, float64) {
	var priceResp []models.PriceResponse
	config.ReadFile()

	priceResp = listar.ListarUltimosValoresReais(coin, 1)

	if len(priceResp) == 0 {
		Write("Erro ao listar ultimos valores", coin)
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
	Write("Quantidade: "+fmt.Sprintf("%.4f", quantity)+" - Preço: "+fmt.Sprintf("%.4f", price), coin)
	return quantity, price
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

func Historico(coin, side, started string, entryPrice float64) {
	config.ReadFile()
	basecoin := coin
	count := contagemRows(basecoin, started)

	switch count {
	case 0:
		query := "INSERT INTO " + config.TabelaHist + " (user, symbol, side, entry_price, started_at) VALUES (?, ?, ?, ?, ?)"
		result := database.DB.Exec(query, global.Key, basecoin, side, entryPrice, started)
		if result.Error != nil {
			WriteError("Erro ao inserir dados iniciais da moeda na tabela "+config.TabelaHist+": ", result.Error, basecoin)
			return
		}
		break
	default:
		query := "UPDATE " + config.TabelaHist + " SET symbol = ?, side = ?, entry_price = ?, started_at = ? WHERE symbol = ? AND started_at = ? AND side = ? AND user = ? AND final_time IS NULL"
		result := database.DB.Exec(query, basecoin, side, entryPrice, started, basecoin, started, side, global.Key)
		if result.Error != nil {
			WriteError("Erro ao atualizar os parâmetros na tabela "+config.TabelaHist+": ", result.Error, basecoin)
			return
		}
		break
	}
}

func EncerrarHistorico(coin, side, started string, currValue, roi float64) {
	count := contagemRows(coin, started)

	if count >= 1 {
		query := "UPDATE " + config.TabelaHist + " SET final_price = ?, final_time = NOW(), final_roi = ? WHERE symbol = ? AND started_at = ? AND side = ? AND user = ?"
		result := database.DB.Exec(query, currValue, roi, coin, started, side, global.Key)
		if result.Error != nil {
			WriteError("Erro ao atualizar os parâmetros na tabela "+config.TabelaHist+": ", result.Error, coin)
			return
		}
	}
}

func contagemRows(basecoin, started string) int {
	query := "SELECT COUNT(*) FROM " + config.TabelaHist + " WHERE symbol = ? AND started_at = ? and user = ?"

	var count int
	result := database.DB.Raw(query, basecoin, started, global.Key).Scan(&count)
	if result.Error != nil {
		WriteError("Erro ao buscar a quantidade de linhas na tabela historico: ", result.Error, basecoin)
		return 0
	}
	return count
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

func GetPrecisionSymbol(currentCoin string) (int, error) {
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
	parts := strings.Split(response.BidPrice, ".")
	if len(parts) == 1 {
		return 0, nil
	}
	precision := len(parts[1])
	return precision, nil
}

func VerificarHook(coin, date string) (action, dateResp string, err error) {
	config.ReadFile()
	var hookAlerts []models.HookAlerts
	basecoin := coin

	if err := database.DB.Order("created_at DESC").Find(&hookAlerts, "user = ? AND symbol = ? AND created_at > NOW() - INTERVAL 30 SECOND", global.Key, basecoin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", err
		}
		return "", "", err
	}
	if len(hookAlerts) == 0 {
		return "", "", nil
	} else if len(hookAlerts) >= 2 {
		return "STOP", "", nil
	} else {
		if date != hookAlerts[0].CreatedAt {
			return hookAlerts[0].Side, hookAlerts[0].CreatedAt, nil
		}
	}
	return "", date, nil
}

func RoundToPrecision(n float64, precision int) float64 {
	shift := math.Pow(10, float64(precision))
	return math.Round(n*shift) / shift
}
