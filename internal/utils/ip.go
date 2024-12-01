package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/logs"
)

type IPDetails struct {
	City            string `json:"city"`
	State           string `json:"state"`
	Pincode         string `json:"pincode"`
	Country         string `json:"country"`
	ServiceProvider string `json:"serviceProvider"`
	IP              string `json:"ip"`
}

func GetUserIP(c echo.Context) (string, error) {
	r := c.Request()
	if clientIP := r.Header.Get("HTTP_CLIENT_IP"); clientIP != "" {
		return clientIP, nil
	} else if forwardedIP := r.Header.Get("X-Forwarded-For"); forwardedIP != "" {
		return forwardedIP, nil
	}
	if realIP := c.RealIP(); realIP != "" {
		return realIP, nil
	}
	return "", fmt.Errorf("unable to determine user IP")
}

func ExtractIpDetails(c echo.Context) (string, error) {
	ip, err := GetUserIP(c)
	if err != nil {
		return "", fmt.Errorf("failed to get ip address")
	}
	logs.Logger.Info(ip)

	apiURL := fmt.Sprintf("http://ip-api.com/json/%s", ip)
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", errors.New("unbale to fetch ip details")
	}

	defer resp.Body.Close()
	var data struct {
		Status     string `json:"status"`
		Message    string `json:"message"`
		City       string `json:"city"`
		RegionName string `json:"regionName"`
		Zip        string `json:"zip"`
		Country    string `json:"country"`
		Isp        string `json:"isp"`
		Query      string `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("Unable to parse ip details")
	}

	if data.Status == "fail" {
		return "", errors.New(data.Message)
	}
	ipDetails := IPDetails{
		City:            data.City,
		State:           data.RegionName,
		Pincode:         data.Zip,
		Country:         data.Country,
		ServiceProvider: data.Isp,
		IP:              ip,
	}
	response := fmt.Sprintf(
		"City: %s\nState: %s\nPincode: %s\nCountry: %s\nService Provider: %s\nIP: %s",
		ipDetails.City,
		ipDetails.State,
		ipDetails.Pincode,
		ipDetails.Country,
		ipDetails.ServiceProvider,
		ipDetails.IP,
	)
	return response, nil
}
