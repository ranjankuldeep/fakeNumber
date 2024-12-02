package runner

import (
	"context"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func MonitorOrders(db *mongo.Database) {
	orderCollection := models.InitializeOrderCollection(db)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Fetch data only with expired time.
	for {
		select {
		case <-ticker.C:
			var orders []models.Order
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cursor, err := orderCollection.Find(ctx, bson.M{})
			if err != nil {
				log.Printf("Error finding orders: %v", err)
				continue
			}
			defer cursor.Close(ctx)

			if err := cursor.All(ctx, &orders); err != nil {
				log.Printf("Error decoding orders: %v", err)
				continue
			}

			for _, order := range orders {
				go handleOrder(order.UserID, db) // Trigger a goroutine for each order
			}
		}
	}
}

// handleOrder processes an individual order for expiration and OTP handling
func handleOrder(userId primitive.ObjectID, db *mongo.Database) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in OTP handling goroutine: %v", r)
		}
	}()

	orderCollection := models.InitializeOrderCollection(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orderFilter := bson.M{"userId": userId}
	cursor, err := orderCollection.Find(ctx, orderFilter)
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	var orders []models.Order
	for cursor.Next(ctx) {
		var order models.Order
		if err := cursor.Decode(&order); err != nil {
			log.Printf("Error decoding order: %v", err)
			return
		}
		orders = append(orders, order)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Cursor error: %v", err)
		return
	}

	for _, order := range orders {
		go processOrder(order, db)
	}
}

// processOrder handles expiration logic and checks OTP status for an order
func processOrder(order models.Order, db *mongo.Database) {
	expirationTime := order.ExpirationTime
	currentTime := time.Now()

	if currentTime.Before(expirationTime) {
		return
	}

	transactionCollection := models.InitializeTransactionHistoryCollection(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	transactionFilter := bson.M{
		"userId": order.UserID.Hex(),
		"id":     order.NumberID,
	}

	var transactionData models.TransactionHistory
	err := transactionCollection.FindOne(ctx, transactionFilter).Decode(&transactionData)
	if err != nil {
		return
	}

	otpArrived := false
	if len(transactionData.OTP) != 0 {
		otpArrived = true
	}

	formattedData := handlers.FormatDateTime()
	if otpArrived {
		orderCollection := models.InitializeOrderCollection(db)
		_, err = orderCollection.DeleteOne(ctx, bson.M{"numberId": order.NumberID})
		if err != nil {
			logs.Logger.Error(err)
			return
		}
		return
	}

	// Refund the balnce
	var apiWalletUser models.ApiWalletUser
	apiWalletColl := models.InitializeApiWalletuserCollection(db)
	err = apiWalletColl.FindOne(ctx, bson.M{"userId": order.UserID}).Decode(&apiWalletUser)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return
	}

	var existingCancelledTransaction models.TransactionHistory
	err = transactionCollection.FindOne(ctx, bson.M{"id": order.NumberID}).Decode(&existingCancelledTransaction)
	if err != nil {
		logs.Logger.Error(err)
		return
	}

	transactionUpdateFilter := bson.M{"id": order.NumberID}
	transactionpdate := bson.M{
		"$set": bson.M{
			"status":    "CANCELLED",
			"date_time": formattedData,
		},
	}

	_, err = transactionCollection.UpdateOne(ctx, transactionUpdateFilter, transactionpdate)
	if err != nil {
		logs.Logger.Error(err)
		return
	}

	// Refund the balance if no otp arrived
	price, err := strconv.ParseFloat(existingCancelledTransaction.Price, 64)
	newBalance := apiWalletUser.Balance + price
	newBalance = math.Round(newBalance*100) / 100

	update := bson.M{
		"$set": bson.M{"balance": newBalance},
	}
	balanceFilter := bson.M{"userId": apiWalletUser.UserID}

	_, err = apiWalletColl.UpdateOne(ctx, balanceFilter, update)
	if err != nil {
		logs.Logger.Error(err)
		return
	}

	// Perform third-party cancellation
	var serverInfo models.Server
	serverCollection := models.InitializeServerCollection(db)
	err = serverCollection.FindOne(ctx, bson.M{"server": order.Server}).Decode(&serverInfo)
	if err != nil {
		log.Printf("Error finding server info for order %s: %v", order.NumberID, err)
		return
	}

	server := strconv.Itoa(serverInfo.ServerNumber)
	constructedNumberRequest, err := handlers.ConstructNumberUrl(server, serverInfo.APIKey, serverInfo.Token, order.NumberID, order.Number)
	if err != nil {
		log.Printf("Error constructing third-party request: %v", err)
		return
	}

	orderCollection := models.InitializeOrderCollection(db)
	_, err = orderCollection.DeleteOne(ctx, bson.M{"numberId": order.NumberID})
	if err != nil {
		logs.Logger.Error(err)
		return
	}

	err = handlers.CancelNumberThirdParty(constructedNumberRequest.URL, server, order.NumberID, db, constructedNumberRequest.Headers)
	if err != nil {
		log.Printf("Error canceling number via third party: %v", err)
		return
	}
}
