package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ranjankuldeep/fakeNumber/logs"
)

// TronAddressResponse represents the structure of the response from the Tron address API
type TronAddressResponse struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privatekey"`
}

func GenerateTronAddress() (string, string, error) {
	apiURL := "https://php.paidsms.in/tron/?type=address"
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch Tron address: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	var tronAddress TronAddressResponse
	if err := json.Unmarshal(body, &tronAddress); err != nil {
		return "", "", fmt.Errorf("failed to parse JSON response: %w", err)
	}
	logs.Logger.Info(tronAddress)
	// Return the address and private key
	return tronAddress.PrivateKey, tronAddress.Address, nil
}
