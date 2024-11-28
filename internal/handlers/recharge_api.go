package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/services"
	"github.com/ranjankuldeep/fakeNumber/internal/utils"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Models
type UpiRequest struct {
	TransactionId string `json:"transactionId"`
	UserId        string `json:"userId"`
	Email         string `json:"email"`
}

type UpiResponse struct {
	Name   string `json:"name,omitempty"`
	Amount int    `json:"amount,omitempty"`
	Date   string `json:"date,omitempty"`
	Error  string `json:"error,omitempty"`
}

type IpDetails struct {
	City            string `json:"city"`
	State           string `json:"state"`
	Pincode         string `json:"pincode"`
	Country         string `json:"country"`
	ServiceProvider string `json:"serviceProvider"`
	IP              string `json:"ip"`
}

// RechargeUpiApi handles the UPI recharge request
func RechargeUpiApi(c echo.Context) error {
	ctx := context.Background()

	transactionId := c.QueryParam("transactionId")
	userId := c.QueryParam("userId")
	email := c.QueryParam("email")

	if userId == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EMPTY_USER_ID"})
	}

	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EMPTY_EMAIL"})
	}

	db := c.Get("db").(*mongo.Database)

	// Check recharge maintenance
	var rechargeData models.RechargeAPI
	rechargeCollection := models.InitializeRechargeAPICollection(db)

	if err := rechargeCollection.FindOne(ctx, bson.M{"recharge_type": "upi"}).Decode(&rechargeData); err != nil {
		log.Println("Recharge data fetch error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	if rechargeData.Maintenance {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "UPI recharge is under maintenance."})
	}

	// Fetch transaction details
	upiUrl := fmt.Sprintf("https://own5k.in/p/u.php?txn=%s", transactionId)
	resp, err := http.Get(upiUrl)
	if err != nil {
		log.Println("UPI API error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	defer resp.Body.Close()

	var upiData UpiResponse
	if err := json.NewDecoder(resp.Body).Decode(&upiData); err != nil {
		log.Println("UPI response parse error:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Transaction Not Found. Please try again."})
	}

	// Handle error in UPI response
	if upiData.Error != "" {
		log.Println("UPI API returned error:", upiData.Error)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Transaction Not Found. Please try again."})
	}

	// Prepare payload for SaveRechargeHistory
	rechargeHistoryUrl := fmt.Sprintf("%sapi/save-recharge-history", os.Getenv("BASE_API_URL"))
	rechargePayload := map[string]interface{}{
		"userId":         userId,
		"transaction_id": transactionId,
		"amount":         upiData.Amount,
		"payment_type":   "upi",
		"date_time":      time.Now().Format("01/02/2006T03:04:05 PM"),
		"status":         "Received",
	}
	rechargePayloadBytes, _ := json.Marshal(rechargePayload)

	// Log the payload being sent
	log.Printf("[INFO] Sending recharge history payload: %s", string(rechargePayloadBytes))

	rechargeResp, err := http.Post(rechargeHistoryUrl, "application/json", bytes.NewReader(rechargePayloadBytes))
	if err != nil {
		log.Printf("[ERROR] Recharge history save error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save recharge history."})
	}
	defer rechargeResp.Body.Close()

	// Log the response status and body
	responseBody, _ := ioutil.ReadAll(rechargeResp.Body)
	log.Printf("[INFO] Recharge history save response status: %d", rechargeResp.StatusCode)
	log.Printf("[INFO] Recharge history save response body: %s", string(responseBody))

	if rechargeResp.StatusCode == http.StatusBadRequest {
		// Extract error message from the SaveRechargeHistory API response
		var errorResponse map[string]string
		if err := json.Unmarshal(responseBody, &errorResponse); err != nil {
			log.Printf("[ERROR] Failed to parse error response: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process recharge history response."})
		}
		return c.JSON(http.StatusBadRequest, errorResponse) // Return the exact error to the client
	}

	if rechargeResp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Recharge history save failed with status: %d, body: %s", rechargeResp.StatusCode, string(responseBody))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save recharge history."})
	}

	log.Println("[INFO] Recharge history saved successfully")

	// Fetch IP details
	ipDetails, err := utils.GetIpDetails(c)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Respond to client
	return c.JSON(http.StatusOK, map[string]string{
		"message":   fmt.Sprintf("%d₹ Added Successfully!", upiData.Amount),
		"ipDetails": ipDetails,
	})
}

type ResponseStructure struct {
	Result bool `json:"result"`
}

// RechargeTrxApi handles TRX recharge transactions.
func RechargeTrxApi(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	log.Println("INFO: RechargeTrxApi endpoint invoked")

	// Get query parameters
	address := c.QueryParam("address")
	hash := c.QueryParam("hash")
	userId := c.QueryParam("userId")
	exchangeRateStr := c.QueryParam("exchangeRate")
	email := c.QueryParam("email")
	logs.Logger.Info(hash, exchangeRateStr)

	// Validate input parameters
	if address == "" || hash == "" || userId == "" || exchangeRateStr == "" || email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing required query parameters",
		})
	}

	// Parse exchange rate to float
	exchangeRate, err := strconv.ParseFloat(exchangeRateStr, 64)
	if err != nil {
		log.Println("ERROR: Invalid exchange rate:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid exchange rate",
		})
	}

	// Fetch transaction data
	trxApiURL := fmt.Sprintf("https://own5k.in/tron/?type=txnid&address=%s&hash=%s", address, hash)
	req, _ := http.NewRequest("GET", trxApiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("ERROR: Failed to fetch TRX transaction data:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Println("ERROR: TRX transaction API returned non-200 status code:", resp.StatusCode)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Transaction not found",
		})
	}

	var trxData struct {
		TRX     float64 `json:"trx"`
		SUCCESS bool    `json:"success"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&trxData); err != nil {
		log.Println("ERROR: Failed to decode TRX transaction response:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process transaction data",
		})
	}

	if trxData.SUCCESS == false {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "TRX ADDRESS NOT FOUND",
		})
	}

	if trxData.TRX <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid TRX transaction",
		})
	}

	price := trxData.TRX * exchangeRate
	amount := strconv.FormatFloat(price, 'f', 2, 64)
	rechargeHistoryPayload := map[string]interface{}{
		"userId":         userId,
		"transaction_id": hash,
		"amount":         amount,
		"payment_type":   "trx",
		"date_time":      time.Now().Format("01/02/2006T03:04:05 PM"),
		"status":         "Received",
	}
	payloadBytes, _ := json.Marshal(rechargeHistoryPayload)

	rechargeHistoryURL := fmt.Sprintf("%sapi/save-recharge-history", os.Getenv("BASE_API_URL"))
	rechargeResp, err := http.Post(rechargeHistoryURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		log.Println("ERROR: Already Done Tranasaction:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "transaction done already",
		})
	}

	defer rechargeResp.Body.Close()
	if rechargeResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(rechargeResp.Body)
		log.Printf("ERROR: Failed to save recharge history. Response: %s", string(body))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Transaction Already Done",
		})
	}
	log.Println("INFO: Recharge history saved successfully for hash:", hash)

	userIdObject, _ := primitive.ObjectIDFromHex(userId)
	apiWalletCollection := models.InitializeApiWalletuserCollection(db)
	var apiWalletUser models.ApiWalletUser
	err = apiWalletCollection.FindOne(context.TODO(), bson.M{"userId": userIdObject}).Decode(&apiWalletUser)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "",
		})
	}
	var adminWallet models.RechargeAPI
	rechargeWalletCollection := models.InitializeRechargeAPICollection(db)
	err = rechargeWalletCollection.FindOne(context.TODO(), bson.M{"recharge_type": "trx"}).Decode(&adminWallet)
	if err != nil {
		logs.Logger.Error(err)
	}

	var user models.User
	userCollection := models.InitializeUserCollection(db)
	err = userCollection.FindOne(context.TODO(), bson.M{"_id": userIdObject}).Decode(&user)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "",
		})
	}

	unsendTrxColl := models.InitializeUnsendTrxCollection(db)
	toAddress := adminWallet.APIKey
	fromAddress := apiWalletUser.TRXAddress
	privateKey := apiWalletUser.TRXPrivateKey

	sentUrl := fmt.Sprintf("https://own5k.in/tron/?type=send&from=%s&key=%s&to=%s", fromAddress, privateKey, toAddress)
	newClient := &http.Client{Timeout: 10 * time.Second}

	ipDetails, err := utils.GetIpDetails(c)
	if err != nil {
		log.Println("ERROR: Failed to fetch IP details:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch IP details",
		})
	}
	rechargeDetail := services.TrxRechargeDetails{
		Email:        user.Email,
		UserID:       userId,
		Trx:          fmt.Sprintf("%.2f", trxData.TRX),
		ExchangeRate: exchangeRateStr,
		Amount:       amount,
		Balance:      fmt.Sprintf("%.2f", apiWalletUser.Balance),
		Address:      fromAddress,
		SendTo:       toAddress,
		Status:       "",
		Hash:         hash,
		IP:           ipDetails,
	}

	// Make the GET request
	response, err := newClient.Get(sentUrl)
	if err != nil {
		log.Println("Failed to call URL:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "",
		})
	}
	defer resp.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Println("Received non-200 status code:", resp.StatusCode)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "",
		})
	}
	var responseData ResponseStructure

	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		log.Println("Failed to unmarshal response:", err)
		unsendTrx := models.UnsendTrx{
			Email:         user.Email,
			TrxAddress:    apiWalletUser.TRXAddress,
			TrxPrivateKey: apiWalletUser.TRXPrivateKey,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		_, insertErr := unsendTrxColl.InsertOne(context.TODO(), unsendTrx)
		if insertErr != nil {
			log.Println("Failed to insert unsend transaction:", insertErr)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "",
			})
		}
		rechargeDetail.Status = "fail"
		err = services.TrxRechargeTeleBot(rechargeDetail)
		if err != nil {
			logs.Logger.Error(err)
			logs.Logger.Info("recharget trx send failed")
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message":   fmt.Sprintf("%.2f₹ Added Successfully!", price),
			"ipDetails": ipDetails,
		})
	}
	rechargeDetail.Status = "ok"
	err = services.TrxRechargeTeleBot(rechargeDetail)
	if err != nil {
		logs.Logger.Error(err)
		logs.Logger.Info("recharget trx send failed")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   fmt.Sprintf("%.2f₹ Added Successfully!", price),
		"ipDetails": ipDetails,
	})
}

func ExchangeRate(c echo.Context) error {
	log.Println("INFO: ExchangeRate endpoint invoked")
	apiURL := "https://own5k.in/p/trxprice.php"

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("ERROR: Failed to fetch exchange rate: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve exchange rate",
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: Received non-200 status code: %d", resp.StatusCode)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve exchange rate",
		})
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read response body: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process exchange rate",
		})
	}
	return c.Blob(http.StatusOK, "application/json", body)
}

func ToggleMaintenance(c echo.Context) error {
	log.Println("INFO: Starting ToggleMaintenance handler")
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	log.Println("INFO: Database instance retrieved successfully")

	type RequestBody struct {
		RechargeType string `json:"rechargeType"`
		Status       bool   `json:"status"`
	}

	var input RequestBody
	log.Println("INFO: Parsing input JSON")
	if err := c.Bind(&input); err != nil {
		log.Println("ERROR: Failed to bind input JSON:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	log.Printf("INFO: Received input - RechargeType: %s, Status: %t\n", input.RechargeType, input.Status)

	// Validate the input
	if input.RechargeType == "" {
		log.Println("ERROR: RechargeType is required")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "RechargeType is required"})
	}

	// Define the filter and update
	filter := bson.M{"recharge_type": input.RechargeType}
	update := bson.M{"$set": bson.M{"maintenance": input.Status}}

	// Log the filter and update details
	log.Printf("INFO: Updating record with filter: %+v and update: %+v\n", filter, update)

	// Initialize the collection
	rechargeApiCol := db.Collection("recharge-apis")

	// Perform the update
	var updatedRecord bson.M
	err := rechargeApiCol.FindOneAndUpdate(
		context.TODO(),
		filter,
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updatedRecord)

	// Handle the result
	if err == mongo.ErrNoDocuments {
		log.Println("INFO: No record found for the given recharge type")
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Recharge type not found"})
	} else if err != nil {
		log.Println("ERROR: Failed to update maintenance status:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	// Log the successful update
	log.Printf("INFO: Successfully updated maintenance status. Updated record: %+v\n", updatedRecord)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Maintenance status updated successfully.",
		"data":    updatedRecord,
	})
}

// GetMaintenanceStatus retrieves the maintenance status.

type MaintenanceResponse struct {
	Maintenance bool   `json:"maintenance"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
}

func GetMaintenanceStatus(c echo.Context) error {
	// Retrieve database instance
	db := c.Get("db").(*mongo.Database)
	rechargeCollection := db.Collection("recharge-apis")

	// Parse query parameter
	rechargeType := c.QueryParam("rechargeType")
	if rechargeType == "" {
		return c.JSON(http.StatusBadRequest, MaintenanceResponse{Error: "Recharge type is required"})
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find the record in the database
	var record bson.M
	err := rechargeCollection.FindOne(ctx, bson.M{"recharge_type": rechargeType}).Decode(&record)

	if err == mongo.ErrNoDocuments {
		// Record not found
		return c.JSON(http.StatusNotFound, MaintenanceResponse{Message: "Recharge type not found"})
	} else if err != nil {
		// Internal server error
		log.Println("Error fetching maintenance status:", err)
		return c.JSON(http.StatusInternalServerError, MaintenanceResponse{Error: "Internal server error"})
	}

	// Return the maintenance status
	maintenance, ok := record["maintenance"].(bool)
	if !ok {
		log.Println("Error: Invalid maintenance status format")
		return c.JSON(http.StatusInternalServerError, MaintenanceResponse{Error: "Internal server error"})
	}

	return c.JSON(http.StatusOK, MaintenanceResponse{Maintenance: maintenance})
}
