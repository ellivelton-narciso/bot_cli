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
	TP1         float64 `json:tp1`
	TP2         float64 `json:tp2`
	TP3         float64 `json:tp3`
	SL1         float64 `json:sl1`
	SL2         float64 `json:sl2`
	SL3         float64 `json:sl3`
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
	TP1         float64
	TP2         float64
	TP3         float64
	SL1         float64
	SL2         float64
	SL3         float64
	Development bool
	UserConfig  UserStruct
)

func ReadFile() {
	user, err := ioutil.ReadFile("user.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = json.Unmarshal(user, &UserConfig)

	err = godotenv.Load()

	ApiKey = UserConfig.ApiKey
	SecretKey = UserConfig.SecretKey
	BaseURL = UserConfig.BaseURL
	Development = UserConfig.Development
	Host = os.Getenv("HOST")
	User = os.Getenv("USER")
	Pass = os.Getenv("PASS")
	Port = os.Getenv("PORT")
	DBname = os.Getenv("DBNAME")
	TP1 = UserConfig.TP1
	TP2 = UserConfig.TP2
	TP3 = UserConfig.TP3
	SL1 = UserConfig.SL1
	SL2 = UserConfig.SL2
	SL3 = UserConfig.SL3

}

func ComputeHmacSha256(secret string, message string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
