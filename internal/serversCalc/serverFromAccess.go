package serverscalc

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ranjankuldeep/fakeNumber/logs"
)

func ExtractNumberServerFromAccess(url string, headers map[string]string) (string, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	logs.Logger.Debug(string(body))
	responseData := string(body)

	if responseData == "NO_BALANCE" {
		return "", "", errors.New("NO_BALANCE")
	} else if responseData == "NO_NUMBERS" {
		return "", "", errors.New("NO_NUMBERS")
	} else if responseData == "BAD_KEY" {
		return "", "", errors.New("BAD_KEY_FROM_SERVER")
	}

	responseParts := strings.Split(responseData, ":")
	if len(responseParts) < 3 {
		return "", "", errors.New("INVALID_RESPONSE_FORMAT")
	}
	id := responseParts[1]
	number := responseParts[2][2:] // Remove the first 2 characters
	return id, number, nil
}
