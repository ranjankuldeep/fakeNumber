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
	"github.com/ranjankuldeep/fakeNumber/logs"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	// isMaintenance, err := checkMaintenance(ctx, serverCol)
	// if err != nil {
	// 	return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	// }
	// if isMaintenance {
	// 	return c.JSON(http.StatusForbidden, echo.Map{"error": "Site is under maintenance."})
	// }

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
	// file := c.FormValue("file")
	// if file == "" {
	// 	return c.String(http.StatusBadRequest, "QR code file is required")
	// }

	// base64Data := file[strings.IndexByte(file, ',')+1:]
	// bufferData, err := base64.StdEncoding.DecodeString(base64Data)
	// if err != nil {
	// 	return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid file format"})
	// }

	// filePath := filepath.Join("uploads", "upi-qr-code.png")
	// os.MkdirAll(filepath.Dir(filePath), os.ModePerm)

	// err = ioutil.WriteFile(filePath, bufferData, 0644)
	// if err != nil {
	// 	return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save QR code"})
	// }

	// return c.String(http.StatusOK, "QR code updated successfully")
	return nil
}

// Handler to create or update API key for recharge type
func CreateOrUpdateAPIKeyHandler(c echo.Context) error {
	// Log: Start of the function
	log.Println("INFO: Starting CreateOrUpdateAPIKeyHandler")

	// Retrieve the database instance
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	log.Println("INFO: Database instance retrieved successfully")

	// Initialize the recharge API collection
	rechargeCol := models.InitializeRechargeAPICollection(db)
	log.Println("INFO: Initialized recharge API collection")

	// Define a struct to map the JSON payload
	type APIKeyRequest struct {
		RechargeType string `json:"recharge_type"`
		APIKey       string `json:"api_key"`
	}

	// Parse the JSON payload
	var req APIKeyRequest
	if err := c.Bind(&req); err != nil {
		log.Println("ERROR: Failed to parse JSON payload:", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}
	log.Printf("INFO: Received payload - Recharge Type: %s, API Key: %s\n", req.RechargeType, req.APIKey)

	// Validate inputs
	if req.RechargeType == "" || req.APIKey == "" {
		log.Println("ERROR: Missing required fields - recharge_type or api_key")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "API key and recharge_type are required"})
	}
	log.Println("INFO: Inputs validated successfully")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the recharge type already exists
	log.Printf("INFO: Checking if recharge_type '%s' exists in the database\n", req.RechargeType)
	var existingAPI models.RechargeAPI
	err := rechargeCol.FindOne(ctx, bson.M{"recharge_type": req.RechargeType}).Decode(&existingAPI)

	// Handle document not found
	if err == mongo.ErrNoDocuments {
		log.Println("INFO: Recharge type not found, creating a new API key")
		_, err = rechargeCol.InsertOne(ctx, models.RechargeAPI{
			RechargeType: req.RechargeType,
			APIKey:       req.APIKey,
		})
		if err != nil {
			log.Println("ERROR: Failed to create API key:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create API key"})
		}
		log.Println("INFO: API key created successfully")
		return c.JSON(http.StatusCreated, echo.Map{"message": "API key created successfully"})
	} else if err != nil {
		log.Println("ERROR: Failed to query recharge API collection:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Update existing API key
	log.Printf("INFO: Recharge type '%s' exists, updating API key\n", req.RechargeType)
	_, err = rechargeCol.UpdateOne(ctx, bson.M{"recharge_type": req.RechargeType}, bson.M{"$set": bson.M{"api_key": req.APIKey}})
	if err != nil {
		log.Println("ERROR: Failed to update API key:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update API key"})
	}

	// Log success
	log.Println("INFO: API key updated successfully")
	return c.JSON(http.StatusOK, echo.Map{"message": "API key updated successfully"})
}

func GetUpiQR(c echo.Context) error {
	amount := c.QueryParam("amt")
	if amount == "" {
		logs.Logger.Info(amount)
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

// UpdateBalanceHandler handles balance updates for a user
func UpdateBalanceHandler(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	walletCol := db.Collection("api_wallet_users")

	// Parse request body
	var requestBody struct {
		UserID     string  `json:"userId"`
		NewBalance float64 `json:"new_balance"`
	}
	if err := c.Bind(&requestBody); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid request body"})
	}

	// Validate input
	if requestBody.UserID == "" || requestBody.NewBalance == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "User ID and new_balance are required"})
	}

	// Create a MongoDB context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find the user by userId
	var user struct {
		Balance float64 `bson:"balance"`
	}
	err := walletCol.FindOne(ctx, bson.M{"userId": requestBody.UserID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusNotFound, echo.Map{"message": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch user"})
	}

	// Calculate the balance difference
	oldBalance := user.Balance
	balanceDifference := requestBody.NewBalance - oldBalance

	// Save the recharge history if there's a balance change
	if balanceDifference != 0 {
		rechargeHistory := map[string]interface{}{
			"userId":         requestBody.UserID,
			"transaction_id": time.Now().UnixNano(), // Unique transaction ID
			"amount":         fmt.Sprintf("%.2f", balanceDifference),
			"payment_type":   "Admin Added",
			"date_time":      time.Now().Format("01/02/2006T03:04:05 PM"), // Format: MM/DD/YYYYThh:mm:ss A
			"status":         "Received",
		}

		// Prepare request for saving recharge history
		rechargeHistoryURL := fmt.Sprintf("%s/api/save-recharge-history", c.Echo().Server.Addr)
		rechargeHistoryJSON, _ := json.Marshal(rechargeHistory)
		req, err := http.NewRequest("POST", rechargeHistoryURL, bytes.NewBuffer(rechargeHistoryJSON))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create recharge history request"})
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save recharge history"})
		}
		defer resp.Body.Close()
	}

	// Update the user's balance in the database
	update := bson.M{"$set": bson.M{"balance": requestBody.NewBalance}}
	_, err = walletCol.UpdateOne(ctx, bson.M{"userId": requestBody.UserID}, update, options.Update())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update balance"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "Balance updated successfully",
		"balance": requestBody.NewBalance,
	})
}

// GetAPIKeyHandler handles fetching an API key based on recharge type
func GetAPIKeyHandler(c echo.Context) error {
	// Log the start of the function
	log.Println("INFO: Starting GetAPIKeyHandler")

	// Retrieve the database instance from context
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	log.Println("INFO: Database instance retrieved successfully")

	// Define the collection name
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
