package serversotpcalc

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// GetOTPServer1 fetches the OTP status from the given URL
func GetOTPServer1(otpUrl string, headers map[string]string, id string) ([]string, error) {
	req, err := http.NewRequest("GET", otpUrl, nil)
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

	responseText := string(body)
	if strings.HasPrefix(responseText, "STATUS_OK:") {
		otp := strings.TrimPrefix(responseText, "STATUS_OK:")
		return []string{otp}, nil
	}
	switch responseText {
	case "STATUS_CANCEL":
		return []string{"STATUS_CANCEL"}, nil
	case "STATUS_WAIT_CODE":
		return []string{"STATUS_WAIT_CODE"}, nil
	}
	if strings.Contains(responseText, "STATUS_WAIT_RETRY") {
		return []string{"STATUS_WAIT_CODE"}, nil
	}
	if strings.Contains(responseText, "ACCESS_CANCEL") {
		return []string{"STATUS_CANCEL"}, nil
	}
	if strings.Contains(responseText, "STATUS_WAIT_RESEND") {
		return []string{}, nil
	}
	return []string{}, fmt.Errorf("UNEXPECTED_RESPONSE %v", responseText)
}
