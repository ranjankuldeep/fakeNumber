package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func GetIpDetails() (string, error) {
	apiURL := fmt.Sprintf("https://own5k.in/p/ip.php")
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
