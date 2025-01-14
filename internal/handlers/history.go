package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	// Check server maintenance status
	var serverData models.Server
	err := serverCol.FindOne(ctx, bson.M{"server": 0}).Decode(&serverData)
	if err != nil && err != mongo.ErrNoDocuments {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error checking maintenance status"})
	}
	if serverData.Maintenance {
		return c.JSON(http.StatusForbidden, echo.Map{"error": "Site is under maintenance."})
	}

	// Query for recharge history
	var rechargeHistoryData []models.RechargeHistory
	cursor, err := rechargeHistoryCol.Find(ctx, bson.M{"userId": userId})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error fetching recharge history"})
	}
	defer cursor.Close(ctx)

	err = cursor.All(ctx, &rechargeHistoryData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error processing recharge history data"})
	}

	// Return an empty array if no history is found
	if len(rechargeHistoryData) == 0 {
		return c.JSON(http.StatusOK, []models.RechargeHistory{})
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
		return c.JSON(http.StatusOK, transactionHistoryData)
	}
	defer cursor.Close(ctx)
	cursor.All(ctx, &transactionHistoryData)

	if len(transactionHistoryData) == 0 {
		return c.JSON(http.StatusOK, transactionHistoryData)
	}

	// Reverse the transaction history data
	for i, j := 0, len(transactionHistoryData)-1; i < j; i, j = i+1, j-1 {
		transactionHistoryData[i], transactionHistoryData[j] = transactionHistoryData[j], transactionHistoryData[i]
	}
	return c.JSON(http.StatusOK, transactionHistoryData)
}

func SaveRechargeHistory(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	rechargeHistoryCol := models.InitializeRechargeHistoryCollection(db)
	apiWalletCol := models.InitializeApiWalletuserCollection(db)

	var request struct {
		UserID        string      `json:"userId"`
		TransactionID string      `json:"transaction_id"`
		Amount        json.Number `json:"amount"`
		PaymentType   string      `json:"payment_type"`
		Status        string      `json:"status"`
	}

	if err := c.Bind(&request); err != nil {
		log.Println("[ERROR] Invalid request body:", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	if request.UserID == "" || request.TransactionID == "" || request.Amount.String() == "" ||
		request.PaymentType == "" || request.Status == "" {
		log.Println("[ERROR] Missing required fields in request body")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	requestAmountFloat, err := request.Amount.Float64()
	if err != nil || requestAmountFloat <= 0 {
		log.Println("[ERROR] Invalid amount:", request.Amount.String())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid amount"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingTransaction models.RechargeHistory
	err = rechargeHistoryCol.FindOne(ctx, bson.M{"transaction_id": request.TransactionID}).Decode(&existingTransaction)
	if err == nil {
		log.Println("[ERROR] Transaction already exists:", request.TransactionID)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Transaction already done"})
	} else if err != mongo.ErrNoDocuments {
		log.Println("[ERROR] Database error while checking transaction:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	userObjectID, err := primitive.ObjectIDFromHex(request.UserID)
	if err != nil {
		log.Println("[ERROR] Invalid userId format:", request.UserID)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid userId format"})
	}

	var userWallet models.ApiWalletUser
	err = apiWalletCol.FindOne(ctx, bson.M{"userId": userObjectID}).Decode(&userWallet)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("[ERROR] User not found:", request.UserID)
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "User not found"})
		}
		log.Println("[ERROR] Database error while fetching user wallet:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	if request.Status == "Received" {
		log.Printf("[INFO] Updating balance for userId: %s with amount: %.2f\n", request.UserID, requestAmountFloat)
		_, err := apiWalletCol.UpdateOne(ctx,
			bson.M{"userId": userObjectID},
			bson.M{"$inc": bson.M{"balance": math.Round(requestAmountFloat*100) / 100}},
		)
		if err != nil {
			log.Println("[ERROR] Failed to update user balance:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update balance"})
		}
		log.Println("[INFO] Balance updated successfully")
	}

	rechargeHistory := models.RechargeHistory{
		UserID:        request.UserID,
		TransactionID: request.TransactionID,
		Amount:        fmt.Sprintf("%.2f", requestAmountFloat),
		PaymentType:   request.PaymentType,
		DateTime:      time.Now().In(time.FixedZone("IST", 5*3600+30*60)).Format("2006-01-02T15:04:05"),
		Status:        request.Status,
		CreatedAt:     time.Now(),
	}
	_, err = rechargeHistoryCol.InsertOne(ctx, rechargeHistory)
	if err != nil {
		log.Println("[ERROR] Failed to save recharge history:", err)
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
		for _, txn := range transactions {
			if txn.Status == "SUCCESS" {
				successCount++
			}
			if txn.Status == "CANCELLED" {
				cancelledCount++
			}
			if txn.Status == "PENDING" {
				pendingCount++
			}
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"successCount":   successCount,
		"cancelledCount": cancelledCount,
		"pendingCount":   pendingCount,
	})
}
