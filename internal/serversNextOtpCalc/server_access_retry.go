package serversnextotpcalc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ranjankuldeep/fakeNumber/logs"
)

func CallNextOTPServerRetry(otpURL string, headers map[string]string) error {
	logs.Logger.Info(otpURL)
	req, err := http.NewRequest("GET", otpURL, nil)
	if err != nil {
		fmt.Printf("Error creating the HTTP request: %v\n", err)
		return err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making the API call: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return err
	}
	responseString := string(body)
	logs.Logger.Info("Response: %s\n", responseString)
	if strings.Contains(responseString, "ACCESS_RETRY_GET") {
		return nil
	} else {
		return errors.New("NEXT_OTP_NOT_TRIGGERED")
	}
}
