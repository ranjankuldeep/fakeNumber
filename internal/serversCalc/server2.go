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

// Response structure for the successful response
type Response struct {
	ID        int    `json:"id"`
	Phone     string `json:"phone"`
	Status    string `json:"status"`
	Country   string `json:"country"`
	CreatedAt string `json:"created_at"`
}

func ExtractNumberServer2(url string, headers map[string]string) (string, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	// Add headers
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	logs.Logger.Debug(string(body))

	if strings.Contains(string(body), "no free phones") {
		return "", "", errors.New("no number available: no free phones")
	}
	if strings.Contains(string(body), "not enough user balance") {
		return "", "", errors.New("no balance available: not enough user balance")
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		logs.Logger.Error(err)
		return "", "", err
	}
	phone := strings.TrimPrefix(response.Phone, "+91")

	// Return the ID and Phone
	return fmt.Sprintf("%s", phone), fmt.Sprintf("%d", response.ID), nil
}
