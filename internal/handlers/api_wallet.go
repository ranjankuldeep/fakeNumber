package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/services"
	"github.com/ranjankuldeep/fakeNumber/internal/utils"
	"github.com/ranjankuldeep/fakeNumber/logs"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Middleware check for maintenance mode
func checkMaintenance(ctx context.Context, serverCol *mongo.Collection) (bool, error) {
	var serverData models.Server
	err := serverCol.FindOne(ctx, bson.M{"server": 0}).Decode(&serverData)
	if err != nil {
		return false, err
	}
	return serverData.Maintenance, nil
}

// Handler to retrieve API key
func ApiKey(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	// serverCol := models.InitializeServerCollection(db)
	walletCol := models.InitializeApiWalletuserCollection(db)

	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "userId is required"})
	}

	var user models.ApiWalletUser
	objID, _ := primitive.ObjectIDFromHex(userId)
	err := walletCol.FindOne(context.TODO(), bson.M{"userId": objID}).Decode(&user)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}
	return c.JSON(http.StatusOK, echo.Map{"api_key": user.APIKey})
}

// Handler to retrieve balance
func BalanceHandler(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	// serverCol := models.InitializeServerCollection(db)
	walletCol := models.InitializeApiWalletuserCollection(db)

	apiKey := c.QueryParam("api_key")
	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid Api Key"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.ApiWalletUser
	err := walletCol.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&user)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}
	return c.JSON(http.StatusOK, echo.Map{"balance": user.Balance})
}

// Handler to change API key
func ChangeAPIKeyHandler(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	serverCol := models.InitializeServerCollection(db)
	walletCol := models.InitializeApiWalletuserCollection(db)

	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "UserId is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	isMaintenance, err := checkMaintenance(ctx, serverCol)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	if isMaintenance {
		return c.JSON(http.StatusForbidden, echo.Map{"error": "Site is under maintenance."})
	}

	newApiKey := uuid.New().String()

	filter := bson.M{"userId": userId}
	update := bson.M{"$set": bson.M{"api_key": newApiKey}}

	_, err = walletCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update API key"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "API key updated successfully", "api_key": newApiKey})
}

// Handler to update UPI QR code
func UpiQRUpdateHandler(c echo.Context) error {
	return nil
}

// Handler to create or update API key for recharge type
func CreateOrUpdateAPIKeyHandler(c echo.Context) error {
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	rechargeCol := models.InitializeRechargeAPICollection(db)
	log.Println("INFO: Initialized recharge API collection")

	type APIKeyRequest struct {
		RechargeType string `json:"recharge_type"`
		APIKey       string `json:"api_key"`
	}

	var req APIKeyRequest
	if err := c.Bind(&req); err != nil {
		log.Println("ERROR: Failed to parse JSON payload:", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}
	if req.RechargeType == "" || req.APIKey == "" {
		log.Println("ERROR: Missing required fields - recharge_type or api_key")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "API key and recharge_type are required"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var existingAPI models.RechargeAPI
	err := rechargeCol.FindOne(ctx, bson.M{"recharge_type": req.RechargeType}).Decode(&existingAPI)

	if err == mongo.ErrNoDocuments {
		_, err = rechargeCol.InsertOne(ctx, models.RechargeAPI{
			RechargeType: req.RechargeType,
			APIKey:       req.APIKey,
		})
		if err != nil {
			log.Println("ERROR: Failed to create API key:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create API key"})
		}
		return c.JSON(http.StatusCreated, echo.Map{"message": "API key created successfully"})
	} else if err != nil {
		log.Println("ERROR: Failed to query recharge API collection:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	_, err = rechargeCol.UpdateOne(ctx, bson.M{"recharge_type": req.RechargeType}, bson.M{"$set": bson.M{"api_key": req.APIKey}})
	if err != nil {
		log.Println("ERROR: Failed to update API key:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update API key"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "API key updated successfully"})
}

func GetUpiQR(c echo.Context) error {
	amount := c.QueryParam("amt")
	if amount == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty amount"})
	}
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	var admintData models.RechargeAPI
	adminWalletCollection := models.InitializeRechargeAPICollection(db)
	err := adminWalletCollection.FindOne(context.TODO(), bson.M{"recharge_type": "upi"}).Decode(&admintData)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": ""})
	}
	upiId := admintData.APIKey
	qrUrl := fmt.Sprintf("https://own5k.in/qr/?upi=%s&amount=%s", upiId, amount)
	return c.JSON(http.StatusOK, echo.Map{
		"url": qrUrl,
	})
}

func UpdateRechargeHandler(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	walletCol := models.InitializeApiWalletuserCollection(db)
	var requestBody struct {
		UserID         string  `json:"userId"`
		RechargeAmount float64 `json:"recharge_amount"`
	}
	if err := c.Bind(&requestBody); err != nil {
		logs.Logger.Error("Failed to bind request body: ", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid request body"})
	}
	if requestBody.UserID == "" || requestBody.RechargeAmount == 0 {
		logs.Logger.Warn("Validation failed: UserID or NewBalance is missing")
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "User ID and new_balance are required"})
	}
	userObjectID, _ := primitive.ObjectIDFromHex(requestBody.UserID)

	var user models.User
	userCollection := models.InitializeUserCollection(db)
	err := userCollection.FindOne(context.TODO(), bson.M{
		"_id": userObjectID,
	}).Decode(&user)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rechargeHistory := map[string]interface{}{
		"userId":         requestBody.UserID,
		"transaction_id": fmt.Sprintf("Admin%02d%02d%02d", time.Now().Hour(), time.Now().Minute(), time.Now().Second()),
		"amount":         fmt.Sprintf("%.2f", requestBody.RechargeAmount),
		"payment_type":   "Admin Added",
		"date_time":      time.Now().Format("01/02/2006T03:04:05 PM"),
		"status":         "Received",
	}

	host := c.Request().Host
	protocol := "http" // Change to "https" if you're using HTTPS
	rechargeHistoryURL := fmt.Sprintf("%s://%s/api/save-recharge-history", protocol, host)
	rechargeHistoryJSON, _ := json.Marshal(rechargeHistory)
	req, err := http.NewRequest("POST", rechargeHistoryURL, bytes.NewBuffer(rechargeHistoryJSON))
	if err != nil {
		logs.Logger.Error("Failed to create recharge history request: ", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create recharge history request"})
	}
	req.Header.Set("Content-Type", "application/json")

	logs.Logger.Infof("Sending recharge history request to URL: %s", rechargeHistoryURL)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		logs.Logger.Errorf("Failed to save recharge history: %v, Status Code: %d", err, resp.StatusCode)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save recharge history"})
	}
	defer resp.Body.Close()
	logs.Logger.Info("Recharge history saved successfully")

	var walletUser models.ApiWalletUser
	err = walletCol.FindOne(ctx, bson.M{"userId": userObjectID}).Decode(&walletUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logs.Logger.Warnf("No user found with UserID: %s", requestBody.UserID)
			return c.JSON(http.StatusNotFound, echo.Map{"message": "User not found"})
		}
		logs.Logger.Error("Failed to fetch user: ", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch user"})
	}

	ipDetail, err := utils.GetIpDetails()
	if err != nil {
		logs.Logger.Error(err)
	}

	rechargeDetails := services.AdminRechargeDetails{
		Email:          user.Email,
		UserID:         userObjectID.Hex(),
		UpdatedBalance: fmt.Sprintf("%0.2f", walletUser.Balance),
		Amount:         fmt.Sprintf("%0.2f", requestBody.RechargeAmount),
		IP:             ipDetail,
	}

	err = services.AdminRechargeTeleBot(rechargeDetails)
	if err != nil {
		logs.Logger.Error(err)
		logs.Logger.Info("Error sending Admin Recharge")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message":  "Recharge Added Successfully",
		"recharge": requestBody.RechargeAmount,
	})
}

func UpdateWalletBalanceHandler(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	walletCol := models.InitializeApiWalletuserCollection(db)
	var requestBody struct {
		UserID     string  `json:"userId"`
		NewBalance float64 `json:"new_balance"`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Bind(&requestBody); err != nil {
		logs.Logger.Error("Failed to bind request body: ", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid request body"})
	}
	if requestBody.UserID == "" || requestBody.NewBalance == 0 {
		logs.Logger.Warn("Validation failed: UserID or NewBalance is missing")
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "User ID and new_balance are required"})
	}
	userObjectID, _ := primitive.ObjectIDFromHex(requestBody.UserID)

	var user models.User
	userCollection := models.InitializeUserCollection(db)
	err := userCollection.FindOne(context.TODO(), bson.M{
		"_id": userObjectID,
	}).Decode(&user)

	update := bson.M{"$set": bson.M{"balance": requestBody.NewBalance}}
	logs.Logger.Info("Updating user balance in the database")
	_, err = walletCol.UpdateOne(ctx, bson.M{"userId": userObjectID}, update)
	if err != nil {
		logs.Logger.Error("Failed to update balance: ", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update balance"})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"message":    "Balance Updated Successfully",
		"newBalance": requestBody.NewBalance,
	})
}

// GetAPIKeyHandler handles fetching an API key based on recharge type
func GetAPIKeyHandler(c echo.Context) error {
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	rechargeCol := db.Collection("recharge-apis")
	log.Println("INFO: Collection initialized: recharge-apis")

	// Get the "type" query parameter
	rechargeType := c.QueryParam("type")
	log.Printf("INFO: Received query parameter - type: %s\n", rechargeType)

	// Validate that the "type" parameter is provided
	if rechargeType == "" {
		log.Println("ERROR: Missing required query parameter 'type'")
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "recharge_type is required"})
	}

	// MongoDB context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	log.Println("INFO: MongoDB context created with 5-second timeout")

	// Query the database for the document with the specified recharge type
	var doc struct {
		APIKey string `bson:"api_key"`
	}
	log.Printf("INFO: Querying database for recharge_type: %s\n", rechargeType)
	err := rechargeCol.FindOne(ctx, bson.M{"recharge_type": rechargeType}).Decode(&doc)

	// Handle potential errors
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("INFO: No document found for recharge_type: %s\n", rechargeType)
			return c.JSON(http.StatusNotFound, echo.Map{"message": "API key not found"})
		}
		log.Println("ERROR: Failed to query database:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Log the successfully retrieved API key
	log.Printf("INFO: Successfully retrieved API key for recharge_type: %s\n", rechargeType)

	// Respond with the API key
	return c.JSON(http.StatusOK, echo.Map{"api_key": doc.APIKey})
}
