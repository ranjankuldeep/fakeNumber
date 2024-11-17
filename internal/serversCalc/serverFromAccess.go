package serverscalc

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

// ExtractServer3 fetches the response from the provided URL with headers, processes it, and extracts id and number
func ExtractNumberServerFromAccess(url string, headers map[string]string) (string, string, error) {
	// Create HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	// Add headers to the request
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// Perform the HTTP request
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

	// Convert body to string
	responseData := string(body)

	if responseData == "NO_BALANCE" {
		return "", "", errors.New("NO_BALANCE")
	} else if responseData == "NO_NUMBERS" {
		return "", "", errors.New("NO_NUMBERS")
	}

	// Split the response into parts
	responseParts := strings.Split(responseData, ":")
	if len(responseParts) < 3 {
		return "", "", errors.New("INVALID_RESPONSE_FORMAT")
	}

	// Extract id and number
	id := responseParts[1]
	number := responseParts[2][2:] // Remove the first 2 characters

	return id, number, nil
}
