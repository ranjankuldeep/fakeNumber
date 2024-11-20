package serversotpcalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ranjankuldeep/fakeNumber/logs"
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
func FetchTokenAndOTP(otpURL, serialNumber string, headers map[string]string) ([]string, error) {
	logs.Logger.Info(otpURL)
	req, err := http.NewRequest("GET", otpURL, nil)
	if err != nil {
		return []string{}, fmt.Errorf("failed to create OTP request: %w", err)
	}

	if len(headers) > 0 {
		for key, value := range headers {
			req.Header.Add(key, value)
		}
	}

	otpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []string{}, fmt.Errorf("failed to fetch OTP: %w", err)
	}
	defer otpResp.Body.Close()

	if otpResp.StatusCode != http.StatusOK {
		return []string{}, fmt.Errorf("unexpected status code while fetching OTP: %d", otpResp.StatusCode)
	}

	otpBody, err := ioutil.ReadAll(otpResp.Body)
	if err != nil {
		return []string{}, fmt.Errorf("failed to read OTP response: %w", err)
	}

	var otpResponse OTPServer9Response
	err = json.Unmarshal(otpBody, &otpResponse)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse OTP response: %w", err)
	}

	if otpResponse.Code != "200" {
		return []string{}, errors.New(otpResponse.Message)
	} else if otpResponse.Code == "210" {
		return []string{}, errors.New(otpResponse.Message)
	}

	for _, vc := range otpResponse.Data.VerificationCode {
		if vc.Vc != "" {
			return []string{vc.Vc}, nil
		} else if vc.Vc == "" {
			return []string{"STATUS_WAIT_CODE"}, nil
		}
	}
	return []string{}, errors.New("NO_OTP_FOUND")
}
