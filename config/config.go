package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ConfigStruct struct {
	ApiKey    string `json:"apiKey"`
	SecretKey string `json:"secretKey"`
	BaseURL   string `json:"baseURL"`
	BaseCoin  string `json:"baseCoin"`
	Host      string `json:"host"`
	User      string `json:"user"`
	Pass      string `json:"pass"`
	Port      string `json:"port"`
	Dbname    string `json:"dbname"`
}

var (
	ApiKey    string
	SecretKey string
	BaseURL   string
	BaseCoin  string
	Host      string
	User      string
	Pass      string
	Port      string
	DBname    string
	Config    ConfigStruct
)

func ReadFile() {
	file, err := ioutil.ReadFile("config.json")

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = json.Unmarshal(file, &Config)

	ApiKey = Config.ApiKey
	SecretKey = Config.SecretKey
	BaseURL = Config.BaseURL
	BaseCoin = Config.BaseCoin
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

func signPayload(timestamp, payload string, secretKey []byte) string {
	message := fmt.Sprintf("%s%s", timestamp, payload)
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
