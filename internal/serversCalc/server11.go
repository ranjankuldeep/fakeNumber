package serverscalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ranjankuldeep/fakeNumber/logs"
)

// APIResponse structure to match the valid response format
type APIResponseServer11 struct {
	RequestID     int    `json:"request_id,omitempty"`
	ApplicationID int    `json:"application_id,omitempty"`
	CountryID     int    `json:"country_id,omitempty"`
	Number        string `json:"number,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
	ErrorMessage  string `json:"error_msg,omitempty"`
	Success       bool   `json:"success,omitempty"`
}

// FetchNumber fetches the number and ID from the given API
func ExtractNumberServer11(url string) (string, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		logs.Logger.Error(err)
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Logger.Error(err)
		return "", "", err
	}

	var apiResponse APIResponseServer11
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		logs.Logger.Error(err)
		return "", "", err
	}

	phone := strings.TrimPrefix(apiResponse.Number, "91")
	if apiResponse.RequestID != 0 && apiResponse.Number != "" {
		return phone, fmt.Sprintf("%d", apiResponse.RequestID), nil
	}
	return "", "", errors.New(apiResponse.ErrorCode)
}
