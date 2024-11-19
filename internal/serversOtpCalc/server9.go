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
func FetchTokenAndOTP(otpURL, serialNumber string) (string, error) {
	tokenURL := "http://www.phantomunion.com:10023/pickCode-api/push/ticket?key=d1967b3a7609f20d010907ed41af1596"
	resp, err := http.Get(tokenURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code while fetching token: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Code != "200" {
		return "", fmt.Errorf("failed to fetch token: %s", tokenResp.Message)
	}

	token := tokenResp.Data.Token

	// Step 2: Fetch OTP using the token
	req, err := http.NewRequest("GET", otpURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create OTP request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

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

	if otpResponse.Code != "200" {
		return "", fmt.Errorf("failed to fetch OTP: %s", otpResponse.Message)
	}

	for _, vc := range otpResponse.Data.VerificationCode {
		if vc.Vc != "" {
			return vc.Vc, nil
		}
	}
	return "", errors.New("no OTP found in the response")
}
