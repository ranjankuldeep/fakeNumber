package serversnextotpcalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ranjankuldeep/fakeNumber/logs"
)

type Response struct {
	RequestID int  `json:"request_id"`
	Success   bool `json:"success"`
}

func CallNextOTPServerUnMarshalling(otpURL string, headers map[string]string) error {
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
	logs.Logger.Info(string(body))
	var apiResponse Response
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		fmt.Printf("Error parsing response JSON: %v\n", err)
		return err
	}

	if apiResponse.Success {
		return nil
	} else {
		return errors.New("NEXT_OTP_NOT_TRIGGERED")
	}
}
