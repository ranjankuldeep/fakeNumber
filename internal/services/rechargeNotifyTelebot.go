package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Define input for TRX recharge
type TrxRechargeDetails struct {
	Email        string
	UserID       string
	Trx          string
	ExchangeRate string
	Amount       string
	Balance      string
	Address      string
	SendTo       string
	Status       string
	IP           string
	Hash         string
}

// Define input for UPI recharge
type UpiRechargeDetails struct {
	Email   string
	UserID  string
	TrnID   string
	Balance string
	Amount  string
	IP      string
}

type AdminRechargeDetails struct {
	Email          string
	UserID         string
	UpdatedBalance string
	Amount         string
	IP             string
}

// FetchUser retrieves user details from the database
func FetchUser(userID string, db *mongo.Database) (*models.ApiWalletUser, error) {
	apiWalletUserCollection := db.Collection("apiWalletUsers") // Update the collection name as needed
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid userID: %v", err)
	}

	filter := bson.M{"userId": userObjectID}
	var user models.ApiWalletUser
	err = apiWalletUserCollection.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("error fetching user balance: %v", err)
	}
	return &user, nil
}

// TrxRechargeTeleBot sends TRX recharge details to Telegram bot
func TrxRechargeTeleBot(details TrxRechargeDetails) error {
	result := "Trx Recharge\n\n"
	result += fmt.Sprintf("Date => %s\n\n", time.Now().Format("02-01-2006 03:04:05 PM"))
	result += fmt.Sprintf("User Email => %s\n\n", details.Email)
	result += fmt.Sprintf("Trx => %s\n\n", details.Trx) // amount of trx
	result += fmt.Sprintf("Trx Exchange Rate => %s\n\n", details.ExchangeRate)
	result += fmt.Sprintf("Total Amount in Inr => %s₹\n\n", details.Amount)
	result += fmt.Sprintf("Updated Balance => %s\n\n", details.Balance)
	result += fmt.Sprintf("User Trx address => %s\n\n", details.Address)
	result += fmt.Sprintf("Send To => %s\n\n", details.SendTo)
	result += fmt.Sprintf("Send Status => %s\n\n", details.Status)
	result += fmt.Sprintf("Hash Id => %s\n\n", details.Hash)
	result += fmt.Sprintf("IP Details => \n%s\n\n", details.IP)

	err := sendRCMessage(result)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	return nil
}

func UpiRechargeTeleBot(details UpiRechargeDetails) (string, error) {
	result := "Upi Recharge\n\n"
	result += fmt.Sprintf("Date => %s\n\n", time.Now().Format("02-01-2006 03:04:05 PM"))
	result += fmt.Sprintf("User Email => %s\n\n", details.Email)
	result += fmt.Sprintf("Amount => %s₹\n\n", details.Amount) // Use string Amount directly
	result += fmt.Sprintf("Updated Balance => %s\n\n", details.Balance)
	result += fmt.Sprintf("IP Details => %s\n\n", details.IP)
	result += fmt.Sprintf("Txn Id => \n%s\n\n", details.TrnID)

	// Use sendMessage to send the result
	err := sendRCMessage(result)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}
	return result, nil
}

func sendRCMessage(message string) error {
	encodedMessage := url.QueryEscape(message)
	apiURL := fmt.Sprintf(
		"https://api.telegram.org/bot6740130325:AAEp1cTpT2o6qgIR4Mb3T2j4s6VDjSVV5Jo/sendMessage?chat_id=6769991787&text=%s",
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
