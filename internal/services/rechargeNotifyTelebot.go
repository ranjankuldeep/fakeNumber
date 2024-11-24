package services

import (
	"context"
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
	Trx          float64
	ExchangeRate float64
	Amount       float64
	Address      string
	SendTo       string
	IP           string
	Hash         string
}

// Define input for UPI recharge
type UpiRechargeDetails struct {
	Email  string
	UserID string
	TrnID  string
	Amount float64
	IP     string
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
func TrxRechargeTeleBot(db *mongo.Database, details TrxRechargeDetails) (string, error) {
	user, err := FetchUser(details.UserID, db)
	if err != nil {
		return "", fmt.Errorf("failed to fetch user balance: %v", err)
	}

	result := fmt.Sprintf(
		"Trx Recharge\n\n"+
			"Date => %s\n\n"+
			"User Email => %s\n\n"+
			"Trx => %.2f\n\n"+
			"Trx Exchange Rate => %.2f\n\n"+
			"Total Amount in Inr => %.2f₹\n\n"+
			"Updated Balance => %.2f₹\n\n"+
			"User Trx address => %s\n\n"+
			"Send To => %s\n\n"+
			"IP Details => %s\n\n"+
			"Txn/Hash Id => %s\n\n",
		time.Now().Format("02-01-2006 03:04:05 PM"),
		details.Email, details.Trx, details.ExchangeRate, details.Amount, user.Balance, details.Address, details.SendTo, details.IP, details.Hash,
	)

	encodedResult := url.QueryEscape(result)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot<your-bot-token>/sendMessage?chat_id=<your-chat-id>&text=%s", encodedResult)
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: status %v", resp.Status)
	}

	return result, nil
}

// UpiRechargeTeleBot sends UPI recharge details to Telegram bot
func UpiRechargeTeleBot(db *mongo.Database, details UpiRechargeDetails) (string, error) {
	user, err := FetchUser(details.UserID, db)
	if err != nil {
		return "", fmt.Errorf("failed to fetch user balance: %v", err)
	}

	result := fmt.Sprintf(
		"Upi Recharge\n\n"+
			"Date => %s\n\n"+
			"User Email => %s\n\n"+
			"Amount => %.2f₹\n\n"+
			"Updated Balance => %.2f₹\n\n"+
			"IP Details => %s\n\n"+
			"Txn Id => %s\n\n",
		time.Now().Format("02-01-2006 03:04:05 PM"),
		details.Email, details.Amount, user.Balance, details.IP, details.TrnID,
	)

	encodedResult := url.QueryEscape(result)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot<your-bot-token>/sendMessage?chat_id=<your-chat-id>&text=%s", encodedResult)
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: status %v", resp.Status)
	}

	return result, nil
}
