package serversotpcalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ranjankuldeep/fakeNumber/logs"
)

type OTPServer11Response struct {
	RequestID     string `json:"request_id"`
	ApplicationID int    `json:"application_id"`
	CountryID     int    `json:"country_id"`
	Number        string `json:"number"`
	ErrorCode     string `json:"error_code,omitempty"` // For waiting case
	ErrorMsg      string `json:"error_msg,omitempty"`  // For waiting case
	SMSCode       string `json:"sms_code,omitempty"`   // For OTP case
}

func GetOTPServer11(otpURL string, requestID string) (string, error) {
	logs.Logger.Info(otpURL)
	resp, err := http.Get(otpURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch OTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var otpResp OTPServer11Response
	err = json.Unmarshal(body, &otpResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	if otpResp.ErrorCode == "wait_sms" {
		return "wait_sms", nil
	}

	if otpResp.ErrorCode == "wrong_status" {
		return "", errors.New("wrong_status")
	}
	if otpResp.SMSCode != "" {
		return otpResp.SMSCode, nil
	}
	return "", errors.New("Unexpected Response: No OTP Found and Not Waiting")
}
