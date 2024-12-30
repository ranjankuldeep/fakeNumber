package runner

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
	"github.com/ranjankuldeep/fakeNumber/internal/services"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func SendSellingUpdate(db *mongo.Database) (services.SellingUpdateDetails, error) {
	var details services.SellingUpdateDetails
	ctx := context.TODO()
	ist := time.FixedZone("IST", 5*3600+30*60)

	now := time.Now().In(ist)

	startOfDayTemp := time.Date(now.Year(), now.Month(), now.Day(), 1, 35, 0, 0, ist)
	endOfDayTemp := startOfDayTemp.Add(24 * time.Hour).Add(-time.Second)
	startOfDayStr := startOfDayTemp.Format("2006-01-02T15:04:05")
	endOfDayStr := endOfDayTemp.Format("2006-01-02T15:04:05")

	// 1. Fetch Total User Count
	usersCollection := models.InitializeUserCollection(db)
	totalUsers, err := usersCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return details, fmt.Errorf("failed to fetch total user count: %w", err)
	}
	details.TotalUserCount = int(totalUsers)

	// 2. Fetch Transaction Details
	transactionCollection := models.InitializeTransactionHistoryCollection(db)
	transactionsPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "date_time", Value: bson.D{
				{Key: "$gte", Value: startOfDayStr},
				{Key: "$lte", Value: endOfDayStr},
			}},
		}}},
		// Group by status and count
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$status"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
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

	// Aggregate transactions grouped by server with "SUCCESS" status
	serverPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "status", Value: "SUCCESS"},
			{Key: "date_time", Value: bson.D{
				{Key: "$gte", Value: startOfDayStr},
				{Key: "$lte", Value: endOfDayStr},
			}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$server"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cursor, err = transactionCollection.Aggregate(ctx, serverPipeline)
	if err != nil {
		return details, fmt.Errorf("failed to aggregate server data: %w", err)
	}
	defer cursor.Close(ctx)

	// Process aggregated server data
	details.ServerUpdates = make(map[int]int)
	for i := 1; i <= 11; i++ {
		details.ServerUpdates[i] = 0
	}

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

	// Define the pipeline for recharges within the day
	rechargePipeline := mongo.Pipeline{
		bson.D{{Key: "$addFields", Value: bson.D{
			{Key: "amount", Value: bson.D{{Key: "$toDouble", Value: "$amount"}}}, // Convert amount to double
		}}},
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "status", Value: "Received"},                       // Only successful recharges
			{Key: "amount", Value: bson.D{{Key: "$ne", Value: nil}}}, // Exclude null amounts
			{Key: "date_time", Value: bson.D{
				{Key: "$gte", Value: startOfDayStr},
				{Key: "$lte", Value: endOfDayStr},
			}}, // Match the specific day
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

	var dailyTotalRechargeAmount float64
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
		dailyTotalRechargeAmount += result.TotalAmount

		// Populate amounts based on payment type
		switch result.ID {
		case "trx":
			details.RechargeDetails.Trx = result.TotalAmount // Total amount for trx
		case "upi":
			details.RechargeDetails.Upi = result.TotalAmount // Total amount for upi
		case "Admin Added":
			details.RechargeDetails.AdminAdded = result.TotalAmount // Total amount for Admin Added
		}
	}
	// Set the daily total recharge amount
	details.RechargeDetails.Total = dailyTotalRechargeAmount

	// 4. Fetch Website Balance (Total Recharge Amount Irrespective of Time)
	apiWalletCollection := models.InitializeApiWalletuserCollection(db)
	apiWalletPipeline := mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil}, // Group all records
			{Key: "totalBalance", Value: bson.D{{Key: "$sum", Value: "$balance"}}}, // Sum all balances
		}}},
	}

	cursor, err = apiWalletCollection.Aggregate(ctx, apiWalletPipeline)
	if err != nil {
		return details, fmt.Errorf("failed to aggregate API wallet balances: %w", err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var apiWalletResult struct {
			TotalBalance float64 `bson:"totalBalance"`
		}
		if err := cursor.Decode(&apiWalletResult); err != nil {
			return details, fmt.Errorf("failed to decode API wallet balances: %w", err)
		}

		// Set Website Balance
		details.WebsiteBalance = math.Round(apiWalletResult.TotalBalance*100) / 100
	}

	// Set the daily total recharge amount
	details.RechargeDetails.Total = dailyTotalRechargeAmount

	// Fil the rest details now
	// Initialize map for server balances
	details.ServersBalance = make(map[string]string)

	// Fetch balances for servers 1 to 11
	for i := 1; i <= 11; i++ {
		serverID := strconv.Itoa(i)

		// Call GetServerBalance for each server
		balance, err := handlers.GetServerBalance(db, serverID)
		if err != nil {
			logs.Logger.Warnf("Failed to fetch balance for server %d: %v", i, err)
			continue // Skip this server and move to the next one
		}

		// Format balance with currency symbol
		formattedBalance := fmt.Sprintf("%.2f%s", balance.Value, balance.Symbol)

		// Map server ID to its balance
		switch serverID {
		case "1":
			details.ServersBalance["Fastsms"] = formattedBalance
		case "2":
			details.ServersBalance["5Sim"] = formattedBalance
		case "3":
			details.ServersBalance["Smshub"] = formattedBalance
		case "4":
			details.ServersBalance["TigerSMS"] = formattedBalance
		case "5":
			details.ServersBalance["GrizzlySMS"] = formattedBalance
		case "6":
			details.ServersBalance["Tempnum"] = formattedBalance
		case "7":
			details.ServersBalance["Smsbower"] = formattedBalance
		case "8":
			details.ServersBalance["Sms-activate"] = formattedBalance
		case "10":
			details.ServersBalance["Sms-activation-service"] = formattedBalance
		case "9":
			details.ServersBalance["CCPAY"] = formattedBalance
		case "11":
			details.ServersBalance["SMS-Man"] = formattedBalance
		}
	}
	// Send selling details via TeleBot
	err = services.SellingTeleBot(details)
	if err != nil {
		logs.Logger.Errorf("Error sending selling message")
	}
	log.Printf("%+v", details)
	return details, nil
}

var mu sync.Mutex

func StartSellingTicker(db *mongo.Database) {
	now := time.Now()
	nextInterval := now.Truncate(30 * time.Minute).Add(30 * time.Minute)
	timeUntilNext := time.Until(nextInterval)

	log.Printf("First SendSellingUpdate scheduled at: %v", nextInterval)
	time.Sleep(timeUntilNext)

	mu.Lock()
	_, err := SendSellingUpdate(db)
	mu.Unlock()
	if err != nil {
		log.Printf("Error in SendSellingUpdate: %v", err)
	}

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mu.Lock()
			_, err := SendSellingUpdate(db)
			mu.Unlock()
			if err != nil {
				log.Printf("Error in SendSellingUpdate: %v", err)
			}
		}
	}
}
