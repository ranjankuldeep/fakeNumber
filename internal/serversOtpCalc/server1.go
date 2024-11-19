package serversotpcalc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// GetOTPServer1 fetches the OTP status from the given URL
func GetOTPServer1(otpUrl string, headers map[string]string, id string) (string, error) {
	req, err := http.NewRequest("GET", otpUrl, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	if len(headers) > 0 {
		for key, value := range headers {
			req.Header.Add(key, value)
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	responseText := string(body)
	if strings.HasPrefix(responseText, "STATUS_OK:") {
		otp := strings.TrimPrefix(responseText, "STATUS_OK:")
		return otp, nil
	}

	switch responseText {
	case "STATUS_WAIT_CODE":
		return "", errors.New("waiting for SMS code")
	case "STATUS_CANCEL":
		return "", errors.New("number canceled")
	default:
		return "", fmt.Errorf("unexpected response: %s", responseText)
	}
}
