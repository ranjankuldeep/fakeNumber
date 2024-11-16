package serverscalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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

// FetchNumberWithToken fetches the token and uses it to fetch the number in one function.
func ExtractNumberServer9() (string, string, error) {
	tokenURL := "http://www.phantomunion.com:10023/pickCode-api/push/ticket?key=af725ae5a94b62313009148d6581c9cf"
	// Step 1: Fetch Token
	resp, err := http.Get(tokenURL)
	if err != nil {
		return "", "", fmt.Errorf("error fetching token: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error reading token response: %w", err)
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", "", fmt.Errorf("error parsing token response: %w", err)
	}

	if tokenResponse.Code != "200" {
		return "", "", errors.New(tokenResponse.Message)
	}

	token := tokenResponse.Data.Token
	if token == "" {
		return "", "", errors.New("no token received from API")
	}

	// Step 2: Fetch Number
	fullURL := fmt.Sprintf("http://www.phantomunion.com:10023/pickCode-api/push/buyCandy?token=%s&businessCode=10643&quantity=1&country=IN&effectiveTime=10", token)
	resp, err = http.Get(fullURL)
	if err != nil {
		return "", "", fmt.Errorf("error fetching number: %w", err)
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error reading number response: %w", err)
	}

	var numberResponse NumberResponse
	if err := json.Unmarshal(body, &numberResponse); err != nil {
		return "", "", fmt.Errorf("error parsing number response: %w", err)
	}

	if numberResponse.Code != "200" {
		return "", "", errors.New(numberResponse.Message)
	}

	if len(numberResponse.Data.PhoneNumber) == 0 {
		return "", "", errors.New("no phone number found in response")
	}

	phoneData := numberResponse.Data.PhoneNumber[0]
	return phoneData.SerialNumber, phoneData.Number, nil
}
