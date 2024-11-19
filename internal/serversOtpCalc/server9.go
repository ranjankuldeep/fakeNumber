package serversotpcalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// TokenResponse represents the response for the token API
type TokenResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
}

// OTPResponse represents the response for the OTP API
type OTPServer9Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		VerificationCode []struct {
			SerialNumber string `json:"serialNumber"`
			Vc           string `json:"vc"` // OTP text
			BusinessCode string `json:"businessCode"`
		} `json:"verificationCode"`
	} `json:"data"`
}

// FetchTokenAndOTP fetches the token and then fetches the OTP using the token
func FetchTokenAndOTP(otpURL, serialNumber string, headers map[string]string) (string, error) {
	req, err := http.NewRequest("GET", otpURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create OTP request: %w", err)
	}

	if len(headers) > 0 {
		for key, value := range headers {
			req.Header.Add(key, value)
		}
	}

	otpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch OTP: %w", err)
	}
	defer otpResp.Body.Close()

	if otpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code while fetching OTP: %d", otpResp.StatusCode)
	}

	otpBody, err := ioutil.ReadAll(otpResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OTP response: %w", err)
	}

	var otpResponse OTPServer9Response
	err = json.Unmarshal(otpBody, &otpResponse)
	if err != nil {
		return "", fmt.Errorf("failed to parse OTP response: %w", err)
	}

	if otpResponse.Code == "210" {
		return "", errors.New(otpResponse.Message)
	}
	if otpResponse.Code != "200" {
		return "", errors.New(otpResponse.Message)
	}

	for _, vc := range otpResponse.Data.VerificationCode {
		if vc.Vc != "" {
			return vc.Vc, nil
		}
	}
	return "", errors.New("no OTP found in the response")
}
