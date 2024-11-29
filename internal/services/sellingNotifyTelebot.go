package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
)

// Struct for selling update details
type SellingUpdateDetails struct {
	TotalSold       int
	TotalCancelled  int
	TotalPending    int
	ServerUpdates   map[int]int
	RechargeDetails RechargeDetailsSelling
	ServersBalance  map[string]string
	WebsiteBalance  float64
	TotalUserCount  int
}

// Struct for recharge details
type RechargeDetailsSelling struct {
	Total      float64
	Trx        float64
	Upi        float64
	AdminAdded float64
}

func SellingTeleBot(details SellingUpdateDetails) error {
	result := fmt.Sprintf("Date => %s\n\n", time.Now().Format("02-01-2006 03:04:05pm"))

	// Total Selling Update
	// Total Selling Update
	result += "Total Number Selling Update\n"
	result += fmt.Sprintf("Total Sold       => %d\n", details.TotalSold)
	result += fmt.Sprintf("Total Cancelled  => %d\n", details.TotalCancelled)
	result += fmt.Sprintf("Total Pending    => %d\n\n", details.TotalPending)

	// Number Selling Update Via Servers (Sorted Order)
	result += "Number Selling Update Via Servers\n"
	// Extract and sort keys
	var keys []int
	for server := range details.ServerUpdates {
		keys = append(keys, server)
	}
	sort.Ints(keys) // Sort keys in ascending order

	// Iterate over sorted keys
	for _, server := range keys {
		result += fmt.Sprintf("Server %d => %d\n", server, details.ServerUpdates[server])
	}
	result += "\n"

	// Recharge Update
	result += "Recharge Update\n"
	result += fmt.Sprintf("Total => %.2f\n", details.RechargeDetails.Total)
	result += fmt.Sprintf("Trx   => %.2f\n", details.RechargeDetails.Trx)
	result += fmt.Sprintf("Upi   => %.2f\n", details.RechargeDetails.Upi)
	result += fmt.Sprintf("Admin Added => %.2f\n\n", details.RechargeDetails.AdminAdded)

	// Servers Balance
	result += "Servers Balance\n"
	serverOrder := []string{
		"Fastsms", "5Sim", "Smshub", "TigerSMS", "GrizzlySMS",
		"Tempnum", "Smsbower", "Sms-activate", "CCPAY", "Sms-activation-service", "SMS-Man",
	}
	for _, server := range serverOrder {
		balance, exists := details.ServersBalance[server]
		if exists {
			result += fmt.Sprintf("%s => %s\n", server, balance)
		} else {
			result += fmt.Sprintf("%s => Not Available\n", server)
		}
	}
	result += "\n"

	// Website Balance and Total User Count
	result += fmt.Sprintf("Website Balance  => %.2f\n", details.WebsiteBalance)
	result += fmt.Sprintf("Total User Count => %d\n", details.TotalUserCount)

	err := sendSellingMessage(result)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	return nil
}

func sendSellingMessage(message string) error {
	encodedMessage := url.QueryEscape(message)
	apiURL := fmt.Sprintf(
		"https://api.telegram.org/bot7311200292:AAF7NYfNP-DUcCRFevOKU4TYg4i-z2X8jtw/sendMessage?chat_id=6769991787&text=%s",
		encodedMessage,
	)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error! status: %d in sending message through TeleBot", resp.StatusCode)
	}
	var response numberResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if response.Ok == false {
		return fmt.Errorf("Unable to send Message")
	}
	return nil
}
