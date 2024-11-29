package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
	result += "Total Number Selling Update\n"
	result += fmt.Sprintf("Total Sold       => %d // success\n", details.TotalSold)
	result += fmt.Sprintf("Total Cancelled  => %d // cancelled\n", details.TotalCancelled)
	result += fmt.Sprintf("Total Pending    => %d // pending\n\n", details.TotalPending)

	// Number Selling Update Via Servers
	result += "Number Selling Update Via Servers\n"
	for server, count := range details.ServerUpdates {
		result += fmt.Sprintf("Server %d => %d\n", server, count)
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
	for server, balance := range details.ServersBalance {
		result += fmt.Sprintf("%s => %s\n", server, balance)
	}
	result += "\n"

	// Website Balance and Total User Count
	result += fmt.Sprintf("Website Balance  => %.2f\n", details.WebsiteBalance)
	result += fmt.Sprintf("Total User Count => %d\n", details.TotalUserCount)

	err := sendRCMessage(result)
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

func FetchSellingUpdate(ctx context.Context, db *mongo.Database) (SellingUpdateDetails, error) {
	var details SellingUpdateDetails

	// 1. Fetch Total User Count
	usersCollection := models.InitializeUserCollection(db)
	totalUsers, err := usersCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return details, fmt.Errorf("failed to fetch total user count: %w", err)
	}
	details.TotalUserCount = int(totalUsers)

	// 2. Fetch Transaction Details
	transactionCollection := models.InitializeTransactionHistoryCollection(db)

	// Aggregate total sold, cancelled, and pending
	transactionsPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$status"},
				{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			}},
		},
	}
	cursor, err := transactionCollection.Aggregate(ctx, transactionsPipeline)
	if err != nil {
		return details, fmt.Errorf("failed to aggregate transaction data: %w", err)
	}
	defer cursor.Close(ctx)

	// Process aggregated transaction data
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return details, fmt.Errorf("failed to decode transaction data: %w", err)
		}

		switch result.ID {
		case "SUCCESS":
			details.TotalSold = result.Count
		case "CANCELLED":
			details.TotalCancelled = result.Count
		case "PENDING":
			details.TotalPending = result.Count
		}
	}

	// Aggregate transactions grouped by server
	serverPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$server"},
				{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			}},
		},
	}
	cursor, err = transactionCollection.Aggregate(ctx, serverPipeline)
	if err != nil {
		return details, fmt.Errorf("failed to aggregate server data: %w", err)
	}
	defer cursor.Close(ctx)

	// Process aggregated server data
	details.ServerUpdates = make(map[int]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return details, fmt.Errorf("failed to decode server data: %w", err)
		}

		serverNumber, err := strconv.Atoi(result.ID)
		if err != nil {
			return details, fmt.Errorf("failed to convert server ID to int: %w", err)
		}
		details.ServerUpdates[serverNumber] = result.Count
	}

	// 3. Fetch Recharge Details
	rechargeCollection := models.InitializeRechargeHistoryCollection(db)
	rechargePipeline := mongo.Pipeline{
		bson.D{{Key: "$addFields", Value: bson.D{
			{Key: "amount", Value: bson.D{{Key: "$toDouble", Value: "$amount"}}}, // Convert amount to double
		}}},
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "status", Value: "Received"},                       // Only successful recharges
			{Key: "amount", Value: bson.D{{Key: "$ne", Value: nil}}}, // Exclude null amounts
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$payment_type"},                                 // Group by payment type
			{Key: "totalAmount", Value: bson.D{{Key: "$sum", Value: "$amount"}}}, // Sum the amount per type
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},               // Count occurrences per type
		}}},
	}
	cursor, err = rechargeCollection.Aggregate(ctx, rechargePipeline)
	if err != nil {
		return details, fmt.Errorf("failed to aggregate recharge data: %w", err)
	}
	defer cursor.Close(ctx)

	var totalRechargeAmount float64
	for cursor.Next(ctx) {
		var result struct {
			ID          string  `bson:"_id"`
			TotalAmount float64 `bson:"totalAmount"`
			Count       int     `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return details, fmt.Errorf("failed to decode recharge data: %w", err)
		}

		// Sum up the total recharge amount
		totalRechargeAmount += result.TotalAmount

		// Populate counts based on payment type
		switch result.ID {
		case "trx":
			details.RechargeDetails.Trx = float64(result.Count)
		case "upi":
			details.RechargeDetails.Upi = float64(result.Count)
		}
	}
	details.RechargeDetails.Total = totalRechargeAmount
	return details, nil
}
