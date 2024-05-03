package util

import (
	"candles/config"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func SendMessageToDiscord(message string) error {
	config.ReadFile()
	if config.AlertasDisc != "" {
		url := config.AlertasDisc
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
