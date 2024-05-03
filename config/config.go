package config

import (
	"candles/models"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var (
	ApiKey      string
	SecretKey   string
	BaseURL     string
	Development bool
	Host        string
	User        string
	Pass        string
	Port        string
	DBname      string
	TabelaHist  string
	BaseCoin    string
	AlertasDisc string
	WakeUp      int
	Sleep       int
	Meta        bool
	UserConfig  models.UserStruct
)

func ReadFile() {
	user, err := ioutil.ReadFile("config.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = json.Unmarshal(user, &UserConfig)

	ApiKey = UserConfig.ApiKey
	SecretKey = UserConfig.SecretKey
	BaseURL = UserConfig.BaseURL
	Development = UserConfig.Development
	Host = UserConfig.Host
	User = UserConfig.User
	Pass = UserConfig.Pass
	Port = UserConfig.Port
	DBname = UserConfig.Dbname
	BaseCoin = "USDT"
	AlertasDisc = UserConfig.AlertasDisc
	WakeUp = UserConfig.WakeUp
	Sleep = UserConfig.Sleep
	TabelaHist = UserConfig.TabelaHist
	Meta = UserConfig.Meta

	if WakeUp < 0 && WakeUp > 23 {
		WakeUp = 7
	}
	if Sleep < 0 && Sleep > 23 {
		Sleep = 2
	}

}

func ComputeHmacSha256(secret string, message string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
