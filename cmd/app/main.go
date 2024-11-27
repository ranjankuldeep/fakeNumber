package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ranjankuldeep/fakeNumber/internal/database"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
	"github.com/ranjankuldeep/fakeNumber/internal/lib"
	"github.com/ranjankuldeep/fakeNumber/internal/routes"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func Load(envFile string) {
	err := godotenv.Load(dir(envFile))
	if err != nil {
		panic(fmt.Errorf("Error loading .env file: %w", err))
	}
}
func dir(envFile string) string {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			break
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			panic(fmt.Errorf("go.mod not found"))
		}
		currentDir = parent
	}

	return filepath.Join(currentDir, envFile)
}

var (
	store = sessions.NewCookieStore([]byte("mY FUckingSEcretKey"))
)

func main() {
	Load(".env")
	e := echo.New()

	uri := "mongodb+srv://test2:amardeep885@cluster0.blfflhg.mongodb.net/Express-Backend?retryWrites=true&w=majority"

	// CORS middleware to allow only http://localhost:5173
	// Configure CORS to allow requests from http://localhost:5173 with credentials
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:5174"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true, // Enable credentials
	}))

	// Connect to the MongoDB client
	client, err := database.ConnectDB(uri)
	if err != nil {
		log.Fatal("Error initializing MongoDB connection:", err)
	}

	// Select the specific database
	db := client.Database("Express-Backend")

	// Run periodically token update of server9
	go func() {
		for {
			log.Println("Running server token update task...")
			err := lib.UpdateServerToken(db)
			if err != nil {
				log.Printf("Error during token update: %v", err)
			}
			log.Println("Server token update task completed.")
			time.Sleep(2 * time.Hour) // Wait for 2 hours before running again
		}
	}()

	// Middleware to set the DB in the context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set DB in the context for every request
			c.Set("db", db)
			return next(c)
		}
	})
	routes.RegisterServiceRoutes(e)
	routes.RegisterGetDataRoutes(e)
	routes.RegisterUserRoutes(e)
	routes.RegisterApiWalletRoutes(e)
	routes.RegisterHistoryRoutes(e)
	routes.RegisterRechargeRoutes(e)
	routes.RegisterUserDiscountRoutes(e)
	routes.RegisterServerRoutes(e)
	routes.RegisterServiceDiscountRoutes(e)
	routes.RegisterServerDiscountRoutes(e)
	routes.RegisterApisRoutes(e)

	// update the server data
	err = UpdateServerData(db, context.TODO())
	if err != nil {
		logs.Logger.Error(err)
	}
	go MonitorOrders(db)
	e.Logger.Fatal(e.Start(":8000"))
}

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
		log.Printf("Error finding transactions for order %s: %v", order.NumberID, err)
		return
	}

	otpArrived := false
	if len(transactionData.OTP) != 0 {
		otpArrived = true
		return
	}
	if otpArrived == true {
		return
	}

	formattedData := handlers.FormatDateTime()
	if otpArrived {
		var existingTransaction models.TransactionHistory
		err = transactionCollection.FindOne(ctx, bson.M{"id": order.NumberID}).Decode(&existingTransaction)
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
