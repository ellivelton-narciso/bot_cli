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
	ApiKey      string `json:"apiKey"`
	SecretKey   string `json:"secretKey"`
	BaseURL     string `json:"baseURL"`
	BaseCoin    string `json:"baseCoin"`
	Development bool   `json:"development"`
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

}

func ComputeHmacSha256(secret string, message string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
