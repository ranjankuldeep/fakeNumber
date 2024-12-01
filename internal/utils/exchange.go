package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func FetchTRXPrice() (float64, error) {
	apiURL := "https://min-api.cryptocompare.com/data/price?fsym=TRX&tsyms=INR"
	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch TRX price: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	var responseData struct {
		INR float64 `json:"INR"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	return responseData.INR, nil
}
