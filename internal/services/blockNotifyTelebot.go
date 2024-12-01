package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type BlockUserDetails struct {
	Date  string
	Email string

	TotalRecharge  string
	UsedBalance    string
	CurrentBalance string
	FraudAmount    string
	ToBeBalance    string

	Reason string
}

// OtpGetDetails sends OTP get details to Telegram
func UserBlockDetails(blockInfo BlockUserDetails) error {
	result := "User Block\n\n"
	result += fmt.Sprintf("Date => %s\n\n", time.Now().Format("02-01-2006 03:04:05PM"))
	result += fmt.Sprintf("User Email => %s\n\n", blockInfo.Email)
	result += fmt.Sprintf("Total Rc => %s\n\n", blockInfo.TotalRecharge)
	result += fmt.Sprintf("Used Balance => %s\n\n", blockInfo.UsedBalance)
	result += fmt.Sprintf("To Be Balance => %s\n\n", blockInfo.ToBeBalance)
	result += fmt.Sprintf("Current Balance => %s\n\n", blockInfo.CurrentBalance)
	result += fmt.Sprintf("Fraud Amount => %s\n\n", blockInfo.FraudAmount)
	result += fmt.Sprintf("Reason => %s\n\n", blockInfo.Reason)
	err := sendBlockMessage(result)
	if err != nil {
		return err
	}
	return nil
}

func sendBlockMessage(message string) error {
	encodedMessage := url.QueryEscape(message)
	apiURL := fmt.Sprintf(
		"https://api.telegram.org/bot6868379504:AAEyCD-0YPsJBtNRhxWk1uSDBCh71H1c5Lg/sendMessage?chat_id=6769991787&text=%s",
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
