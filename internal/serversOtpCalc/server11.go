package serversotpcalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ranjankuldeep/fakeNumber/logs"
)

type OTPServer11ResponseString struct {
	RequestID     string `json:"request_id"`
	ApplicationID int    `json:"application_id"`
	CountryID     int    `json:"country_id"`
	Number        string `json:"number"`
	ErrorCode     string `json:"error_code,omitempty"` // For waiting case
	ErrorMsg      string `json:"error_msg,omitempty"`  // For waiting case
	SMSCode       string `json:"sms_code,omitempty"`   // For OTP case
}

type OTPServer11ResponseInt struct {
	RequestID     int    `json:"request_id"`
	ApplicationID int    `json:"application_id"`
	CountryID     int    `json:"country_id"`
	Number        string `json:"number"`
	ErrorCode     string `json:"error_code,omitempty"` // For waiting case
	ErrorMsg      string `json:"error_msg,omitempty"`  // For waiting case
	SMSCode       string `json:"sms_code,omitempty"`   // For OTP case
}

func GetOTPServer11(otpURL string, requestID string) ([]string, error) {
	logs.Logger.Info(otpURL)

	resp, err := http.Get(otpURL)
	if err != nil {
		return []string{}, fmt.Errorf("failed to fetch OTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []string{}, fmt.Errorf("failed to read response body: %w", err)
	}

	logs.Logger.Infof("Response Body: %s", string(body))

	var otpRespString OTPServer11ResponseString
	err = json.Unmarshal(body, &otpRespString)
	if err == nil {
		logs.Logger.Infof("Parsed as string response: %+v", otpRespString)
		return processOTPResponseString(otpRespString)
	}

	var otpRespInt OTPServer11ResponseInt
	err = json.Unmarshal(body, &otpRespInt)
	if err == nil {
		logs.Logger.Infof("Parsed as int response: %+v", otpRespInt)
		return processOTPResponseInt(otpRespInt)
	}

	return []string{}, fmt.Errorf("failed to parse response JSON: %w", err)
}

func processOTPResponseString(resp OTPServer11ResponseString) ([]string, error) {
	if resp.ErrorCode == "wait_sms" {
		return []string{"STATUS_WAIT_CODE"}, nil
	}

	if resp.ErrorCode == "wrong_status" {
		return []string{"STATUS_CANCEL"}, nil
	}
	if resp.SMSCode != "" {
		return []string{resp.SMSCode}, nil
	}
	return []string{}, errors.New("Unexpected Response: No OTP Found and Not Waiting")
}

func processOTPResponseInt(resp OTPServer11ResponseInt) ([]string, error) {
	if resp.ErrorCode == "wait_sms" {
		return []string{}, nil
	}

	if resp.ErrorCode == "wrong_status" {
		return []string{}, errors.New("wrong_status")
	}
	if resp.SMSCode != "" {
		return []string{resp.SMSCode}, nil
	}
	return []string{}, errors.New("Unexpected Response: No OTP Found and Not Waiting")
}
