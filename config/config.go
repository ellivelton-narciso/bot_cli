package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type UserStruct struct {
	ApiKey      string  `json:"apiKey"`
	SecretKey   string  `json:"secretKey"`
	BaseURL     string  `json:"baseURL"`
	Development bool    `json:"development"`
	Host        string  `json:"host"`
	User        string  `json:"user"`
	Pass        string  `json:"pass"`
	Port        string  `json:"port"`
	Dbname      string  `json:"dbname"`
	Value       float64 `json:"value"`
	Alavancagem float64 `json:"alavancagem"`
	Tabela      string  `json:"tabela"`
	TabelaHist  string  `json:"tabelaHist"`
}

var (
	ApiKey      string
	SecretKey   string
	BaseURL     string
	Host        string
	User        string
	Pass        string
	Port        string
	DBname      string
	Development bool
	Value       float64
	Alavancagem float64
	Tabela      string
	TabelaHist  string
	UserConfig  UserStruct
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
	Value = UserConfig.Value
	Alavancagem = UserConfig.Alavancagem
	Tabela = UserConfig.Tabela
	TabelaHist = UserConfig.TabelaHist
	Host = UserConfig.Host
	User = UserConfig.User
	Pass = UserConfig.Pass
	Port = UserConfig.Port
	DBname = UserConfig.Dbname

}

func ComputeHmacSha256(secret string, message string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
