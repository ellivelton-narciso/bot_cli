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

func ConvertBaseCoin(coin string, value float64, apiKey string) (float64, float64) {
	var priceResp []models.HistoricoAll
	config.ReadFile()

	priceResp, _ = listar_ordens.ListarUltimosValores(coin, 1)

	if len(priceResp) == 0 {
		Write("Erro ao listar ultimos valores", coin)
		return 0, 0
	}

	price, err := strconv.ParseFloat(priceResp[0].CurrentValue, 64)
	if err != nil {
		WriteError("Erro ao converter preço para float64: ", err, coin)
	}

	precision, err := GetPrecision(coin, apiKey)
	if err != nil {
		precision = 0
		WriteError("Erro ao buscar precisao para converter a moeda: ", err, coin)
	}

	q := value / price
	quantity := math.Round(q*math.Pow(10, float64(precision))) / math.Pow(10, float64(precision))
	Write("Quantidade: "+fmt.Sprintf("%.4f", quantity)+" - Preço: "+fmt.Sprintf("%.4f", price), coin)
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

func DefinirAlavancagem(currentCoin string, alavancagem float64, apikey, secretKey string) error {
	now := time.Now()
	timestamp := now.UnixMilli()
	apiParams := "symbol=" + currentCoin + "&leverage=" + fmt.Sprint(alavancagem) + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(secretKey, apiParams)
	url := config.BaseURL + "fapi/v1/leverage?" + apiParams + "&signature=" + signature

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", apikey)

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

func DefinirMargim(currentCoin, margim, apiKey, secretKey string) error {
	now := time.Now()
	timestamp := now.UnixMilli()
	margim = strings.ToUpper(margim)
	apiParams := "symbol=" + currentCoin + "&marginType=" + margim + "&timestamp=" + strconv.FormatInt(timestamp, 10)
	signature := config.ComputeHmacSha256(secretKey, apiParams)
	url := config.BaseURL + "fapi/v1/marginType?" + apiParams + "&signature=" + signature
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", apiKey)

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

func RegistroLogs(basecoin, side, currDateTelegram string, motivo int64, currValue float64) {
	var modo bool
	modo = false
	if modo {
		config.ReadFile()
		count := contagemRows(basecoin, currDateTelegram)

		switch count {
		case 0:
			query := "INSERT INTO " + config.TabelaHist + " (coin, side, started_at, date_tg, price_tg, trigger_tg) VALUES (?, ?, NOW(), ?, ?)"
			result := database.DB.Exec(query, basecoin, side, currDateTelegram, motivo)
			if result.Error != nil {
				WriteError("Erro ao inserir dados na tabela de "+config.TabelaHist+" de logs, motivo: ", result.Error, basecoin)
				return
			}
			break
		case 1:
			query := "UPDATE " + config.TabelaHist + " SET trigger_tg = ? WHERE coin = ? AND side = ? AND date_tg = ?"
			result := database.DB.Exec(query, motivo, basecoin, side, currDateTelegram)
			if result.Error != nil {
				WriteError("Erro ao atualizar dados na tabela de historico de logs, motivo: ", result.Error, basecoin)
				return
			}
			break
		default:
			Write("Na tabela posui mais de uma ordem para o mesmo alerta", basecoin)
			break

		}
	}
}

func Historico(coin, side, started, parametros, currDateTelegram string, currValue, currValueTelegram, entryPrice, roi float64) {
	var modo bool
	modo = false
	if modo {
		config.ReadFile()
		basecoin := coin
		count := contagemRows(basecoin, currDateTelegram)

		switch count {
		case 0:
			query := "INSERT INTO " + config.TabelaHist + " (coin, side, entryPrice, started_at, price_tg, date_tg) VALUES (?, ?, ?, ?, ?, ?)"
			result := database.DB.Exec(query, basecoin, side, entryPrice, started, currValueTelegram, currDateTelegram)
			if result.Error != nil {
				WriteError("Erro ao inserir dados iniciais da moeda na tabela "+config.TabelaHist+": ", result.Error, basecoin)
				return
			}
			break
		default:
			query := "UPDATE " + config.TabelaHist + " SET " + parametros + " = ?, " + parametros + "_time = NOW(), " + parametros + "_roi = ?, coin = ?, side = ?, entryPrice = ?, started_at = ?, price_tg = ?, date_tg = ?, trigger_tg = -1 WHERE coin = ? AND started_at = ? AND side = ? AND " + parametros + " IS NULL"
			result := database.DB.Exec(query, currValue, roi, basecoin, side, entryPrice, started, currValueTelegram, currDateTelegram, basecoin, started, side)
			if result.Error != nil {
				WriteError("Erro ao atualizar os parâmetros na tabela "+config.TabelaHist+": ", result.Error, basecoin)
				return
			}
			break
		}
	}
}

func EncerrarHistorico(coin, side, started string, currValue, roi float64) {
	var modo bool
	modo = false
	if modo {
		count := contagemRows(coin, started)

		if count >= 1 {
			query := "UPDATE " + config.TabelaHist + " SET final_price = ?, final_time = NOW(), final_roi = ? WHERE coin = ? AND date_tg = ? AND side = ?"
			result := database.DB.Exec(query, currValue, roi, coin, started, side)
			if result.Error != nil {
				WriteError("Erro ao atualizar os parâmetros na tabela hist_transactions: ", result.Error, coin)
				return
			}
		}
	}
}

func contagemRows(basecoin, started string) int {
	var modo bool
	modo = false
	if modo {
		query := "SELECT COUNT(*) FROM " + config.TabelaHist + " WHERE coin = ? AND date_tg = ?"

		var count int
		result := database.DB.Raw(query, basecoin, started).Scan(&count)
		if result.Error != nil {
			WriteError("Erro ao buscar a quantidade de linhas na tabela historico: ", result.Error, basecoin)
			return 0
		}
		return count
	}
	return 0
}

func BuscarValoresTelegram(coin string) []models.ResponseQuery {
	var bots []models.ResponseQuery

	if err := database.DB.Raw(`
		select hist_date,
			   trading_name                                                             coin,
			   case when trend_value > 0 then 'LONG' else 'SHORT' end                   tend,
			   curr_value,
			   target_perc                                                              SP,
			   target_perc                                                              SL,
			   other_value
		from findings_history
		where ((other_value = 31 AND trend_value < 0 AND trading_name in (
			SELECT trading_name
			FROM (
					 SELECT TIPO_ALERTA,
							trend,
							trading_name,
							SUM(CASE WHEN status = 'W' THEN 1 ELSE 0 END) AS total_win,
							COUNT(1)                                      AS total
					 FROM (
							  SELECT  ROUND(other_value)                                       AS TIPO_ALERTA,
									  trading_name,
									  (CASE WHEN trend_value > 0 THEN 'LONG' ELSE 'SHORT' END) AS trend,
									  status
							  FROM findings_history a
							  WHERE close_date > NOW() - INTERVAL 4 DAY
								AND status IN ('W', 'L')
						  ) x
					 GROUP BY TIPO_ALERTA, trading_name, trend) z
			WHERE z.TIPO_ALERTA IN (31)
			  AND trend = 'SHORT'
			  AND total > 1
			  AND ROUND(total_win / total * 100, 2) >= 70
			)) or (other_value = 51 AND trend_value > 0))
		  and trading_name not in (select coin from bots_real)
		  and status = 'R'
		  AND hist_date > (NOW() - INTERVAL 1 MINUTE)
		order by hist_date
	`, coin).Scan(&bots).Error; err != nil {
		return []models.ResponseQuery{}
	}

	return bots

}

func GetPrecision(currentCoin, apiKey string) (int, error) {
	url := "https://fapi.binance.com/fapi/v1/ticker/bookTicker?symbol=" + currentCoin
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", apiKey)

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

func GetPrecisionSymbol(currentCoin, apiKey string) (int, error) {
	url := "https://fapi.binance.com/fapi/v1/ticker/bookTicker?symbol=" + currentCoin
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", apiKey)

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

func SendMessageToDiscord(message, url string) error {
	config.ReadFile()
	if url != "" {
		method := "POST"

		payload := strings.NewReader(fmt.Sprintf(`{
        	"content": "%s"
    	}`, message))

		client := &http.Client{}
		req, err := http.NewRequest(method, url, payload)

		if err != nil {
			return err
		}
		req.Header.Add("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		_, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return nil
	}
	return nil
}

func GetStop(symbol, start string) bool {
	var count int64
	if err := database.DB.Raw(`
		SELECT COUNT(*)
		FROM findings_history fh
		WHERE fh.other_value = 218
		  AND fh.status = 'S' AND fh.trading_name = ? AND fh.hist_date = ?
	`, symbol, start).Scan(&count).Error; err != nil {
		log.Println("Erro ao buscar dados da query GetStop():", err)
		return false
	}
	return count == 1
}
