package serverscalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// APIResponse structure to match the valid response format
type APIResponseServer11 struct {
	RequestID     int    `json:"request_id,omitempty"`
	ApplicationID int    `json:"application_id,omitempty"`
	CountryID     int    `json:"country_id,omitempty"`
	Number        string `json:"number,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
	ErrorMessage  string `json:"error_msg,omitempty"`
	Success       bool   `json:"success,omitempty"`
}

// FetchNumber fetches the number and ID from the given API
func ExtractNumberServer11(url string) (string, string, error) {
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
	var apiResponse APIResponseServer11
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", "", err
	}

	// Handle errors based on error_code or success flag
	if apiResponse.ErrorCode != "" {
		return "", "", errors.New(apiResponse.ErrorCode)
	}

	// Check for invalid API key scenario
	if !apiResponse.Success && apiResponse.ErrorCode == "wrong_token" {
		return "", "", errors.New(apiResponse.ErrorCode)
	}

	// Extract and return ID and Number for valid responses
	if apiResponse.RequestID != 0 && apiResponse.Number != "" {
		return fmt.Sprintf("%d", apiResponse.RequestID), apiResponse.Number, nil
	}

	// Handle unexpected response formats
	return "", "", errors.New("unknown error or invalid response")
}
