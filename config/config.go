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
	BaseCoin    string  `json:"baseCoin"`
	Development bool    `json:"development"`
	TP1         float64 `json:tp1`
	TP2         float64 `json:tp2`
	TP3         float64 `json:tp3`
	SL1         float64 `json:sl1`
	SL2         float64 `json:sl2`
	SL3         float64 `json:sl3`
}
type ConfigStruct struct {
	Host   string `json:"host"`
	User   string `json:"user"`
	Pass   string `json:"pass"`
	Port   string `json:"port"`
	Dbname string `json:"dbname"`
}

var (
	ApiKey      string
	SecretKey   string
	BaseURL     string
	BaseCoin    string
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
	Config      ConfigStruct
	UserConfig  UserStruct
)

func ReadFile() {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = json.Unmarshal(file, &Config)
	user, err := ioutil.ReadFile("user.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = json.Unmarshal(user, &UserConfig)

	ApiKey = UserConfig.ApiKey
	SecretKey = UserConfig.SecretKey
	BaseURL = UserConfig.BaseURL
	BaseCoin = UserConfig.BaseCoin
	Development = UserConfig.Development
	Host = Config.Host
	User = Config.User
	Pass = Config.Pass
	Port = Config.Port
	DBname = Config.Dbname
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
