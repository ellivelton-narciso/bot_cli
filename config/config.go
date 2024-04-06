package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io/ioutil"
	"os"
)

type UserStruct struct {
	ApiKey      string  `json:"apiKey"`
	SecretKey   string  `json:"secretKey"`
	BaseURL     string  `json:"baseURL"`
	Development bool    `json:"development"`
	Value       float64 `json:"value"`
	Alavancagem float64 `json:"alavancagem"`
	Tabela      string  `json:"tabela"`
	ViewFiltro  string  `json:"viewFiltro"`
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
	ViewFiltro  string
	TabelaHist  string
	UserConfig  UserStruct
)

func ReadFile() {
	user, err := ioutil.ReadFile("user.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = json.Unmarshal(user, &UserConfig)

	err = godotenv.Load(".env")
	if err != nil {
		fmt.Println(err)
	}

	ApiKey = UserConfig.ApiKey
	SecretKey = UserConfig.SecretKey
	BaseURL = UserConfig.BaseURL
	Development = UserConfig.Development
	Value = UserConfig.Value
	Alavancagem = UserConfig.Alavancagem
	Tabela = UserConfig.Tabela
	ViewFiltro = UserConfig.ViewFiltro
	TabelaHist = UserConfig.TabelaHist
	Host = os.Getenv("HOST")
	User = os.Getenv("USER")
	Pass = os.Getenv("PASS")
	Port = os.Getenv("PORT")
	DBname = os.Getenv("DBNAME")

}

func ComputeHmacSha256(secret string, message string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
