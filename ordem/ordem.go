package ordem

import (
	"candles/config"
	"candles/database"
	"candles/global"
	"candles/listar"
	"candles/models"
	"candles/util"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

func criarOrdem(coin, side, quantity, posSide string) (int, error) {
	config.ReadFile()
	if !config.Development {
		now := time.Now()
		timestamp := now.UnixMilli()
		apiParamsOrdem := "symbol=" + coin + "&type=MARKET&side=" + side + "&positionSide=" + posSide + "&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(timestamp, 10)
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
		return res.StatusCode, nil
	} else {
		return 200, nil
	}
}

func ComprarBuy(symbol, quantity, side, posSide string, stop float64) int {
	config.ReadFile()
	order, err := criarOrdem(symbol, side, quantity, posSide)
	if err != nil {
		util.WriteError("Erro ao dar entrada m LONG: ", err, symbol)

		if !config.Development {
			allOrders, err := listar.ListarOrdens(symbol)
			if err != nil {
				util.WriteError("Erro ao listar ordens: ", err, symbol)
				time.Sleep(time.Second)
				return 500
			}
			for _, item := range allOrders {
				entryPriceFloat, _ := strconv.ParseFloat(item.EntryPrice, 64)
				if entryPriceFloat > 0 {
					util.Write("Ja possui ordem ativa.", symbol)
					return 500
				}
			}
			time.Sleep(time.Second)
			return 0
		}
		time.Sleep(time.Second)
		return 0
	}
	if config.Development || order == 200 {
		util.Write("Entrada em LONG: "+fmt.Sprint(global.ValueCompradoCoin), symbol)

		util.Historico(symbol, side, global.Started, global.ValueCompradoCoin)
		global.ForTime = 5 * time.Second

		precisionSymbol, err := util.GetPrecisionSymbol(symbol)

		q := global.ValueCompradoCoin * (1 - (((stop / global.Alavancagem) / 100) * 1.1))
		stopSeguro := math.Round(q*math.Pow(10, float64(precisionSymbol))) / math.Pow(10, float64(precisionSymbol))
		slSeguro, resposta, err := criarSLSeguro(symbol, "SELL", fmt.Sprint(stopSeguro), posSide)
		if err != nil {
			log.Println("Erro ao criar Stop Loss Seguro para, ", symbol, " motivo: ", err)
			util.WriteError("Não foi criada ordem para STOPLOSS, motivo: ", err, symbol)
			return 200
		}
		if slSeguro != 200 {
			util.Write("Stop Loss Seguro não criado, "+resposta, symbol)
			return 200
		}
		util.Write("Stop Loss Seguro foi criado.", symbol)
		return 200
	} else {
		util.Write("A ordem de LONG não foi totalmente completada.", symbol)
		time.Sleep(time.Second)
		return 0
	}
}

func ComprarSell(symbol, quantity, side, posSide string, stop float64) int {
	config.ReadFile()
	order, err := criarOrdem(symbol, side, quantity, posSide)
	if err != nil {
		util.WriteError("Erro ao dar entrada em SHORT: ", err, symbol)
		if !config.Development {
			allOrders, err := listar.ListarOrdens(symbol)
			if err != nil {
				util.WriteError("Erro ao listar ordens: ", err, symbol)
				return 500
			}
			for _, item := range allOrders {
				entryPriceFloat, _ := strconv.ParseFloat(item.EntryPrice, 64)
				if entryPriceFloat > 0 {
					util.Write("Ja possui ordem ativa.", symbol)
					return 500
				}
			}
			time.Sleep(time.Second)
			return 0
		}
		time.Sleep(time.Second)
		return 0
	}
	if config.Development || order == 200 {
		util.Write("Entrada em SHORT: "+fmt.Sprint(global.ValueCompradoCoin), symbol)
		util.Historico(symbol, side, global.Started, global.ValueCompradoCoin)
		global.ForTime = 5 * time.Second

		precisionSymbol, err := util.GetPrecisionSymbol(symbol)

		q := global.ValueCompradoCoin * (1 + (((stop / global.Alavancagem) / 100) * 1.1))
		stopSeguro := math.Round(q*math.Pow(10, float64(precisionSymbol))) / math.Pow(10, float64(precisionSymbol))
		slSeguro, resposta, err := criarSLSeguro(symbol, "BUY", fmt.Sprint(stopSeguro), posSide)
		if err != nil {
			log.Println("Erro ao criar Stop Loss Seguro para, ", symbol, " motivo: ", err)
			util.WriteError("Não foi criada ordem para STOPLOSS, motivo: ", err, symbol)
			util.Write(resposta, symbol)
			time.Sleep(time.Second)
			return 200
		}
		if slSeguro != 200 {
			util.Write("Stop Loss Seguro não criado, "+resposta, symbol)
			time.Sleep(time.Second)
			return 200
		}
		util.Write("Stop Loss Seguro foi criado.", symbol)
		return 200

	} else {
		util.Write("A ordem de SHORT não foi totalmente completada.", symbol)
		time.Sleep(time.Second)
		return 0
	}
}

func EncerrarOrdem(currentCoin, side, posSide string, quantity float64) int {
	config.ReadFile()
	if !config.Development {
		// Valida se a ordem ja foi encerrada para evitar abrir ordem no sentido contrário.
		all, err := listar.ListarOrdens(currentCoin)
		if err != nil {
			log.Println("Erro ao listar ordens: ", err)
		}
		for _, item := range all {
			if item.PositionSide == posSide {
				if item.EntryPrice == "0.0" {
					util.Write("Ordem ja foi encerrada anteriormente, manual ou por SL Seguro. Finalizando...", currentCoin)
					return 200
				}
			}
		}
	}

	// Encerra a Ordem
	var opposSide string
	if side == "BUY" {
		opposSide = "SELL"
	} else if side == "SELL" {
		opposSide = "BUY"
	} else {
		fmt.Println("SIDE não é nem BUY nem SELL, Side: ", side)
		return 0
	}
	order, err := criarOrdem(currentCoin, opposSide, fmt.Sprint(quantity), posSide)
	if err != nil {
		_ = RemoverCoinDB(currentCoin, global.Key)
		global.CurrentCoin = ""
		return 0
	}

	// Cancela o StopLoss Seguro que foi criado.
	_, err = cancelarSLSeguro(currentCoin)
	if err != nil {
		msgError := "Erro ao cancelar Stop Loss Seguro de " + currentCoin
		fmt.Println(msgError)
		util.WriteError(msgError, err, currentCoin)
	}
	global.CurrentCoin = ""
	return order
}

func EnviarCoinDB(coin, key string) {
	var bot models.Bots
	result := database.DB.Where("symbol = ? AND user = ?", coin, key).First(&bot)
	if result.RowsAffected > 0 {
		return
	}

	if err := database.DB.Create(&models.Bots{Symbol: coin, User: key}).Error; err != nil {
		fmt.Println("\n Erro ao inserir coin na DB: ", err)
	}
	util.Write("Inserido na tabela bots", coin)
	time.Sleep(10 * time.Second)
	return
}

func RemoverCoinDB(coin, key string) error {
	if err := database.DB.Where("symbol = ? AND user = ?", coin, key).Delete(&models.Bots{}).Error; err != nil {
		util.WriteError("\n Erro ao remover coin na DB: ", err, coin)
		return err
	}
	util.Write("Removido da tabela bots", coin)
	return nil
}

func criarSLSeguro(coin, side, stop, posSide string) (int, string, error) {
	config.ReadFile()

	if !config.Development {
		now := time.Now()
		timestamp := now.UnixMilli()
		apiParamsOrdem := "symbol=" + coin + "&type=STOP_MARKET&side=" + side + "&positionSide=" + posSide + "&closePosition=true&stopPrice=" + stop + "&timestamp=" + strconv.FormatInt(timestamp, 10)
		signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)

		urlOrdem := config.BaseURL + "fapi/v1/order?" + apiParamsOrdem + "&signature=" + signatureOrdem

		req, err := http.NewRequest("POST", urlOrdem, nil)
		if err != nil {
			return 500, "", err
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("X-MBX-APIKEY", config.ApiKey)

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

func cancelarSLSeguro(coin string) (int, error) {
	config.ReadFile()

	if !config.Development {
		now := time.Now()
		timestamp := now.UnixMilli()
		apiParamsOrdem := "symbol=" + coin + "&timestamp=" + strconv.FormatInt(timestamp, 10)
		signatureOrdem := config.ComputeHmacSha256(config.SecretKey, apiParamsOrdem)

		urlOrdem := config.BaseURL + "fapi/v1/allOpenOrders?" + apiParamsOrdem + "&signature=" + signatureOrdem
		req, err := http.NewRequest("DELETE", urlOrdem, nil)
		if err != nil {
			return 500, err
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("X-MBX-APIKEY", config.ApiKey)

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
