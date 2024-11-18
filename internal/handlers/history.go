package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
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
	log.Println(transactionHistoryData)
	return c.JSON(http.StatusOK, transactionHistoryData)
}

// Handler to save a recharge history entry
func SaveRechargeHistory(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	rechargeHistoryCol := models.InitializeRechargeHistoryCollection(db)
	apiWalletCol := models.InitializeApiWalletuserCollection(db)

	var request struct {
		UserID        string `json:"userId"`
		TransactionID string `json:"transaction_id"`
		Amount        string `json:"amount"`
		PaymentType   string `json:"payment_type"`
		DateTime      string `json:"date_time"`
		Status        string `json:"status"`
	}
	requestAmountInt, _ := strconv.Atoi(request.Amount)
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	if request.UserID == "" || request.TransactionID == "" || requestAmountInt == 0 || request.PaymentType == "" || request.DateTime == "" || request.Status == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the transaction already exists
	var existingTransaction models.RechargeHistory
	err := rechargeHistoryCol.FindOne(ctx, bson.M{"transaction_id": request.TransactionID}).Decode(&existingTransaction)
	if err == nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Transaction already done"})
	}

	// Update balance if status is "Received"
	if request.Status == "Received" {
		var apiWallet models.ApiWalletUser
		err := apiWalletCol.FindOne(ctx, bson.M{"userId": request.UserID}).Decode(&apiWallet)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "User not found"})
		}

		requestAmountInt, _ := strconv.Atoi(request.Amount)
		apiWallet.Balance += float64(requestAmountInt)
		apiWalletCol.UpdateOne(ctx, bson.M{"userId": request.UserID}, bson.M{"$set": bson.M{"balance": apiWallet.Balance}})
	}

	// Save recharge history
	rechargeHistory := models.RechargeHistory{
		UserID:        request.UserID,
		TransactionID: request.TransactionID,
		Amount:        request.Amount,
		PaymentType:   request.PaymentType,
		DateTime:      request.DateTime,
		Status:        request.Status,
	}
	_, err = rechargeHistoryCol.InsertOne(ctx, rechargeHistory)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save recharge"})
	}

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
