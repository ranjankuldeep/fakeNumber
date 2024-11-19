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

func FetchUser(userID string, db *mongo.Database) (*models.ApiWalletUser, error) {
	apiWalletUserCollection := models.InitializeApiWalletuserCollection(db)
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

// TrxRechargeTeleBot sends transaction recharge details to Telegram bot
func TrxRechargeTeleBot(db *mongo.Database, email, userID string, trx float64, exchangeRate, amount float64, address, sendTo, ip, hash string) (string, error) {
	user, err := FetchUser(userID, db)
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
		time.Now().Format("02-01-2006 03:04:05PM"),
		email, trx, exchangeRate, amount, user.Balance, address, sendTo, ip, hash,
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
func UpiRechargeTeleBot(db *mongo.Database, email, userID, trnID string, amount float64, ip string) (string, error) {
	user, err := FetchUser(userID, db)
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
		time.Now().Format("02-01-2006 03:04:05PM"),
		email, amount, user.Balance, ip, trnID,
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
