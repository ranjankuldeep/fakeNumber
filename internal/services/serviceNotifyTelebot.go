package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type OTPDetails struct {
	Email       string
	ServiceName string
	ServiceCode string
	Price       string
	Server      string
	Number      string
	OTP         string
	Ip          string
}

type NumberDetails struct {
	Email       string
	ServiceName string
	ServiceCode string
	Price       string
	Server      string
	Balance     string
	Number      string
	Ip          string
}
type CancelDetails struct {
	Email       string
	ServiceName string
	ServiceCode string
	Price       string
	Server      string
	Number      string
	Balance     string
	IP          string
}

type numberResponse struct {
	Ok bool `json:"ok"`
}

// sendMessage sends a message to the Telegram Bot API
func sendMessage(message string) error {
	encodedMessage := url.QueryEscape(message)
	apiURL := fmt.Sprintf(
		"https://api.telegram.org/bot7032433639:AAHmG8mSIaZGvhpBlaWflyew7QwiNUf0wSA/sendMessage?chat_id=6769991787&text=%s",
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

// NumberGetDetails sends number get details to Telegram
func NumberGetDetails(numberInfo NumberDetails) error {
	result := "Number Get\n\n"
	result += fmt.Sprintf("Date => %s\n\n", time.Now().Format("02-01-2006 03:04:05PM"))
	result += fmt.Sprintf("User Email => %s\n\n", numberInfo.Email)
	result += fmt.Sprintf("Service Name => %s\n\n", numberInfo.ServiceName)
	result += fmt.Sprintf("Service Code => %s\n\n", numberInfo.ServiceCode)
	result += fmt.Sprintf("Price => %s₹\n\n", numberInfo.Price)
	result += fmt.Sprintf("Server => %s\n\n", numberInfo.Server)
	result += fmt.Sprintf("Number => %s\n\n", numberInfo.Number)
	result += fmt.Sprintf("Balance => %s₹\n\n", numberInfo.Balance)
	result += fmt.Sprintf("IP Details => \n%s\n\n", numberInfo.Ip)

	err := sendMessage(result)
	if err != nil {
		return err
	}
	return nil
}

// OtpGetDetails sends OTP get details to Telegram
func OtpGetDetails(otpInfo OTPDetails) error {
	result := "Otp Get\n\n"
	result += fmt.Sprintf("Date => %s\n\n", time.Now().Format("02-01-2006 03:04:05PM"))
	result += fmt.Sprintf("User Email => %s\n\n", otpInfo.Email)
	result += fmt.Sprintf("Service Name => %s\n\n", otpInfo.ServiceName)
	result += fmt.Sprintf("Service Code => %s\n\n", otpInfo.ServiceCode)
	result += fmt.Sprintf("Price => %s₹\n\n", otpInfo.Price)
	result += fmt.Sprintf("Server => %s\n\n", otpInfo.Server)
	result += fmt.Sprintf("Number => %s\n\n", otpInfo.Number)
	result += fmt.Sprintf("Otp => %s\n\n", otpInfo.OTP)
	result += fmt.Sprintf("IP Details => \n%s\n\n", otpInfo.Ip)
	err := sendMessage(result)
	if err != nil {
		return err
	}
	return nil
}

// NumberCancelDetails sends number cancel details to Telegram
func NumberCancelDetails(cancelInfo CancelDetails) error {
	result := "Number Cancel\n\n"
	result += fmt.Sprintf("Date => %s\n\n", time.Now().Format("02-01-2006 03:04:05PM"))
	result += fmt.Sprintf("User Email => %s\n\n", cancelInfo.Email)
	result += fmt.Sprintf("Service Name => %s\n\n", cancelInfo.ServiceName)
	result += fmt.Sprintf("Service Code => %s\n\n", cancelInfo.ServiceCode)
	result += fmt.Sprintf("Price => %s₹\n\n", cancelInfo.Price)
	result += fmt.Sprintf("Server => %s\n\n", cancelInfo.Server)
	result += fmt.Sprintf("Number => %s\n\n", cancelInfo.Number)
	result += fmt.Sprintf("Balance => %s₹\n\n", cancelInfo.Balance)
	result += "Status => Number Cancelled\n\n"
	result += fmt.Sprintf("IP Details => \n%s\n\n", cancelInfo.IP)
	err := sendMessage(result)
	if err != nil {
		return err
	}
	return nil
}
