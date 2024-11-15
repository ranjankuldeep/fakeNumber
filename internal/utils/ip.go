package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/labstack/echo/v4"
)

type IPDetails struct {
	City            string `json:"city"`
	State           string `json:"state"`
	Pincode         string `json:"pincode"`
	Country         string `json:"country"`
	ServiceProvider string `json:"serviceProvider"`
	IP              string `json:"ip"`
}

// getIpDetails fetches details about the IP address using the ip-api service
func GetIpDetails(c echo.Context) (*IPDetails, error) {
	// Get the IP address from the request headers or connection
	ip := c.Request().Header.Get("X-Forwarded-For")
	if ip == "" {
		ip, _, _ = net.SplitHostPort(c.Request().RemoteAddr)
	}

	// Call the IP-API service
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s", ip)
	resp, err := http.Get(apiURL)
	if err != nil {
		return &IPDetails{
			City:            "unknown",
			State:           "unknown",
			Pincode:         "unknown",
			Country:         "unknown",
			ServiceProvider: "unknown",
			IP:              ip,
		}, fmt.Errorf("failed to fetch IP details: %v", err)
	}
	defer resp.Body.Close()

	// Decode the response JSON
	var apiResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		City    string `json:"city"`
		Region  string `json:"regionName"`
		Zip     string `json:"zip"`
		Country string `json:"country"`
		ISP     string `json:"isp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return &IPDetails{
			City:            "unknown",
			State:           "unknown",
			Pincode:         "unknown",
			Country:         "unknown",
			ServiceProvider: "unknown",
			IP:              ip,
		}, fmt.Errorf("failed to parse IP details response: %v", err)
	}

	// Check if the API request was successful
	if apiResponse.Status == "fail" {
		return &IPDetails{
			City:            "unknown",
			State:           "unknown",
			Pincode:         "unknown",
			Country:         "unknown",
			ServiceProvider: "unknown",
			IP:              ip,
		}, fmt.Errorf("failed to fetch IP details: %s", apiResponse.Message)
	}

	// Return the extracted IP details
	return &IPDetails{
		City:            apiResponse.City,
		State:           apiResponse.Region,
		Pincode:         apiResponse.Zip,
		Country:         apiResponse.Country,
		ServiceProvider: apiResponse.ISP,
		IP:              ip,
	}, nil
}
