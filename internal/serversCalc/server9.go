package serverscalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ranjankuldeep/fakeNumber/logs"
)

// TokenResponse represents the response structure for the token API.
type TokenResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
}

// NumberResponse represents the response structure for the number API.
type NumberResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PhoneNumber []struct {
			Number       string `json:"number"`
			BusinessCode string `json:"businessCode"`
			SerialNumber string `json:"serialNumber"`
			Imsi         string `json:"imsi"`
			Country      string `json:"country"`
			AreaCode     string `json:"areaCode"`
		} `json:"phoneNumber"`
		Balance string `json:"balance"`
	} `json:"data"`
}

func ExtractNumberServer9(fullURL string, headers map[string]string) (string, string, error) {
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error fetching number: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error reading number response: %w", err)
	}
	logs.Logger.Debug(string(body))

	var numberResponse NumberResponse
	if err := json.Unmarshal(body, &numberResponse); err != nil {
		return "", "", fmt.Errorf("error parsing number response: %w", err)
	}

	if numberResponse.Code == "221" {
		return "", "", errors.New(numberResponse.Message)
	}
	if numberResponse.Code != "200" {
		return "", "", errors.New(numberResponse.Message)
	}

	if len(numberResponse.Data.PhoneNumber) == 0 {
		return "", "", errors.New("no phone number found in response")
	}
	phoneData := numberResponse.Data.PhoneNumber[0]
	phone := strings.TrimPrefix(phoneData.Number, "+91")
	return phone, phoneData.SerialNumber, nil
}
