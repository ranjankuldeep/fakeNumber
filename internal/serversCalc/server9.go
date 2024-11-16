package serverscalc

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

// Response structure to match the API response
type APIResponse struct {
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

// ExtractNumberAndId fetches and processes the API response to extract the number and ID
func ExtractNumberServer9(url string) (string, string, error) {
	// Make HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	// Parse the JSON response
	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", "", err
	}

	// Handle response code logic
	switch apiResponse.Code {
	case "200":
		// Extract ID and number if available
		if len(apiResponse.Data.PhoneNumber) > 0 {
			serialNumber := apiResponse.Data.PhoneNumber[0].SerialNumber
			number := apiResponse.Data.PhoneNumber[0].Number
			return serialNumber, number, nil
		}
		return "", "", errors.New("NO_PHONE_NUMBER_AVAILABLE")
	case "221":
		return "", "", errors.New("PHONE_NUMBER_IS_INITIATING")
	case "210":
		return "", "", errors.New("TOKEN_ERROR")
	default:
		return "", "", errors.New(apiResponse.Message)
	}
}
