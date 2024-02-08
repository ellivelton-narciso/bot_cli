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
}

var (
	ApiKey    string
	SecretKey string
	BaseURL   string
	BaseCoin  string
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
}

func ComputeHmacSha256(secret string, message string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
