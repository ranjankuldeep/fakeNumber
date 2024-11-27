package serversotpcalc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ranjankuldeep/fakeNumber/logs"
)

// OTPResponse represents the structure of the response from the API
type OTPResponse struct {
	ID       int    `json:"id"`
	Phone    string `json:"phone"`
	Operator string `json:"operator"`
	Product  string `json:"product"`
	Price    int    `json:"price"`
	Status   string `json:"status"`
	Expires  string `json:"expires"`
	SMS      []struct {
		CreatedAt string `json:"created_at"`
		Date      string `json:"date"`
		Sender    string `json:"sender"`
		Text      string `json:"text"`
		Code      string `json:"code"`
	} `json:"sms"`
	CreatedAt string `json:"created_at"`
	Country   string `json:"country"`
}

func GetSMSTextsServer2(otpURL string, id string, headers map[string]string) ([]string, error) {
	logs.Logger.Info(otpURL)
	req, err := http.NewRequest("GET", otpURL, nil)
	if err != nil {
		return []string{}, fmt.Errorf("failed to create request: %w", err)
	}

	if len(headers) > 0 {
		for key, value := range headers {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []string{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []string{}, fmt.Errorf("failed to read response body: %w", err)
	}
	logs.Logger.Info(string(body))
	var otpResponse OTPResponse
	err = json.Unmarshal(body, &otpResponse)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse response JSON: %w", err)
	}
	logs.Logger.Info(otpResponse)

	var smsTexts []string
	for _, sms := range otpResponse.SMS {
		smsTexts = append(smsTexts, sms.Text)
	}

	if otpResponse.Status == "CANCELED" {
		return []string{}, fmt.Errorf("ACCESS_CANCEL")
	}
	if otpResponse.Status == "TIMEOUT" {
		return []string{}, fmt.Errorf("ACCESS_CANCEL")
	}
	if len(smsTexts) == 0 {
		return []string{}, nil
	}
	return smsTexts, nil
}
