package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Handler to get recharge history for a user
func GetRechargeHistory(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	rechargeHistoryCol := models.InitializeRechargeHistoryCollection(db)
	serverCol := models.InitializeServerCollection(db)

	userId := c.QueryParam("userId")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var serverData models.Server
	err := serverCol.FindOne(ctx, bson.M{"server": 0}).Decode(&serverData)
	if err != nil && err != mongo.ErrNoDocuments {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error checking maintenance status"})
	}
	if serverData.Maintenance {
		return c.JSON(http.StatusForbidden, echo.Map{"error": "Site is under maintenance."})
	}

	var rechargeHistoryData []models.RechargeHistory
	cursor, err := rechargeHistoryCol.Find(ctx, bson.M{"userId": userId})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch recharge history"})
	}
	defer cursor.Close(ctx)
	cursor.All(ctx, &rechargeHistoryData)

	if len(rechargeHistoryData) == 0 {
		return c.JSON(http.StatusOK, echo.Map{"message": "No recharge history found for the provided userId"})
	}

	// Reverse the recharge history data
	for i, j := 0, len(rechargeHistoryData)-1; i < j; i, j = i+1, j-1 {
		rechargeHistoryData[i], rechargeHistoryData[j] = rechargeHistoryData[j], rechargeHistoryData[i]
	}

	return c.JSON(http.StatusOK, rechargeHistoryData)
}

// Handler to get transaction history for a user
func GetTransactionHistory(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	transactionHistoryCol := models.InitializeTransactionHistoryCollection(db)
	serverCol := models.InitializeServerCollection(db)

	userId := c.QueryParam("userId")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var serverData models.Server
	err := serverCol.FindOne(ctx, bson.M{"server": 0}).Decode(&serverData)
	if err != nil && err != mongo.ErrNoDocuments {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error checking maintenance status"})
	}
	if serverData.Maintenance {
		return c.JSON(http.StatusForbidden, echo.Map{"error": "Site is under maintenance."})
	}

	var transactionHistoryData []models.TransactionHistory
	cursor, err := transactionHistoryCol.Find(ctx, bson.M{"userId": userId})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch transaction history"})
	}
	defer cursor.Close(ctx)
	cursor.All(ctx, &transactionHistoryData)

	if len(transactionHistoryData) == 0 {
		return c.JSON(http.StatusOK, echo.Map{"message": "No transaction history found for the provided userId"})
	}

	// Reverse the transaction history data
	for i, j := 0, len(transactionHistoryData)-1; i < j; i, j = i+1, j-1 {
		transactionHistoryData[i], transactionHistoryData[j] = transactionHistoryData[j], transactionHistoryData[i]
	}
	return c.JSON(http.StatusOK, transactionHistoryData)
}

// Handler to save a recharge history entry
func SaveRechargeHistory(c echo.Context) error {
	fmt.Println("SaveRechargeHistory")

	// Get MongoDB collections
	db := c.Get("db").(*mongo.Database)
	rechargeHistoryCol := models.InitializeRechargeHistoryCollection(db)
	apiWalletCol := models.InitializeApiWalletuserCollection(db)

	// Request payload structure
	var request struct {
		UserID        string      `json:"userId"`
		TransactionID string      `json:"transaction_id"`
		Amount        json.Number `json:"amount"` // Use json.Number for flexible type handling
		PaymentType   string      `json:"payment_type"`
		DateTime      string      `json:"date_time"`
		Status        string      `json:"status"`
	}

	// Bind request payload
	if err := c.Bind(&request); err != nil {
		log.Println("[ERROR] Invalid request body:", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	// Validate required fields
	if request.UserID == "" || request.TransactionID == "" || request.Amount.String() == "" ||
		request.PaymentType == "" || request.DateTime == "" || request.Status == "" {
		log.Println("[ERROR] Missing required fields in request body")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	// Validate amount
	requestAmountInt, err := request.Amount.Int64()
	if err != nil || requestAmountInt <= 0 {
		log.Println("[ERROR] Invalid amount:", request.Amount.String())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid amount"})
	}

	// Format date_time
	const dateTimeFormat = "01/02/2006T03:04:05 PM"
	parsedTime, err := time.Parse("2006-01-02 03:04:05 PM", request.DateTime)
	if err != nil {
		log.Println("[ERROR] Invalid date_time format:", request.DateTime)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid date_time format"})
	}
	formattedDateTime := parsedTime.Format(dateTimeFormat)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the transaction already exists
	var existingTransaction models.RechargeHistory
	err = rechargeHistoryCol.FindOne(ctx, bson.M{"transaction_id": request.TransactionID}).Decode(&existingTransaction)
	if err == nil {
		log.Println("[ERROR] Transaction already exists:", request.TransactionID)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Transaction already done"})
	} else if err != mongo.ErrNoDocuments {
		log.Println("[ERROR] Database error while checking transaction:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	// Update balance if status is "Received"
	if request.Status == "Received" {
		_, err := apiWalletCol.UpdateOne(ctx,
			bson.M{"userId": request.UserID},
			bson.M{"$inc": bson.M{"balance": float64(requestAmountInt)}},
		)
		if err != nil {
			log.Println("[ERROR] Failed to update user balance:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update balance"})
		}
	}

	// Save recharge history
	rechargeHistory := models.RechargeHistory{
		UserID:        request.UserID,
		TransactionID: request.TransactionID,
		Amount:        request.Amount.String(),
		PaymentType:   request.PaymentType,
		DateTime:      formattedDateTime, // Save formatted date_time
		Status:        request.Status,
	}
	_, err = rechargeHistoryCol.InsertOne(ctx, rechargeHistory)
	if err != nil {
		log.Println("[ERROR] Failed to save recharge history:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save recharge"})
	}

	log.Println("[INFO] Recharge saved successfully for transaction:", request.TransactionID)
	return c.JSON(http.StatusOK, echo.Map{"message": "Recharge Saved Successfully!"})
}

// Handler to count transaction statuses
func TransactionCount(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	transactionHistoryCol := models.InitializeTransactionHistoryCollection(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var transactionHistories []models.TransactionHistory
	cursor, err := transactionHistoryCol.Find(ctx, bson.M{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch transaction history"})
	}
	defer cursor.Close(ctx)
	cursor.All(ctx, &transactionHistories)

	transactionsById := make(map[string][]models.TransactionHistory)
	for _, transaction := range transactionHistories {
		transactionsById[transaction.ID.String()] = append(transactionsById[transaction.ID.String()], transaction)
	}

	successCount := 0
	cancelledCount := 0
	pendingCount := 0

	for _, transactions := range transactionsById {
		hasFinished := false
		hasCancelled := false
		hasOtp := false
		for _, txn := range transactions {
			if txn.Status == "FINISHED" {
				hasFinished = true
			}
			if txn.Status == "CANCELLED" {
				hasCancelled = true
			}
			if txn.OTP != "" {
				hasOtp = true
			}
		}

		if hasFinished && hasOtp {
			successCount++
		} else if hasFinished && hasCancelled {
			cancelledCount++
		} else if hasFinished && !hasCancelled && !hasOtp {
			pendingCount++
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"successCount":   successCount,
		"cancelledCount": cancelledCount,
		"pendingCount":   pendingCount,
	})
}
