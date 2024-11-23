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
	"github.com/ranjankuldeep/fakeNumber/internal/utils"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
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

	// Get query parameters
	transactionId := c.QueryParam("transactionId")
	userId := c.QueryParam("userId")
	email := c.QueryParam("email")

	// Validate input parameters
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
		"date_time":      upiData.Date,
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

// RechargeTrxApi handles TRX recharge transactions.
func RechargeTrxApi(c echo.Context) error {
	log.Println("INFO: RechargeTrxApi endpoint invoked")

	// ctx := context.Background()

	// Get query parameters
	address := c.QueryParam("address")
	hash := c.QueryParam("hash")
	userId := c.QueryParam("userId")
	exchangeRateStr := c.QueryParam("exchangeRate")
	email := c.QueryParam("email")

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

	// Get MongoDB database instance
	// db := c.Get("db").(*mongo.Database)

	// Commenting out the recharge maintenance check
	/*
		var rechargeData models.RechargeAPI
		rechargeCollection := models.InitializeRechargeAPICollection(db)
		if err := rechargeCollection.FindOne(ctx, bson.M{"recharge_type": "trx"}).Decode(&rechargeData); err != nil {
			log.Println("ERROR: Failed to fetch recharge data:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal server error",
			})
		}
		if rechargeData.Maintenance {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "TRX recharge is under maintenance",
			})
		}
	*/

	// Fetch transaction data
	trxApiURL := fmt.Sprintf("https://own5k.in/tron/?type=txnid&address=%s&hash=%s", address, hash)
	resp, err := http.Get(trxApiURL)
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

	// Parse TRX transaction response
	var trxData struct {
		TRX float64 `json:"trx"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&trxData); err != nil {
		log.Println("ERROR: Failed to decode TRX transaction response:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process transaction data",
		})
	}

	// Check transaction amount
	if trxData.TRX <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid TRX transaction",
		})
	}

	// Calculate amount in INR
	price := trxData.TRX * exchangeRate

	// Save recharge history
	rechargeHistoryPayload := map[string]interface{}{
		"userId":         userId,
		"transaction_id": hash,
		"amount":         fmt.Sprintf("%.2f", price),
		"payment_type":   "trx",
		"date_time":      time.Now().Format("01/02/2006T03:04:05 PM"),
		"status":         "Received",
	}
	payloadBytes, _ := json.Marshal(rechargeHistoryPayload)

	rechargeHistoryURL := fmt.Sprintf("%sapi/save-recharge-history", os.Getenv("BASE_API_URL"))
	rechargeResp, err := http.Post(rechargeHistoryURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		log.Println("ERROR: Failed to save recharge history:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save recharge history",
		})
	}
	defer rechargeResp.Body.Close()

	if rechargeResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(rechargeResp.Body)
		log.Printf("ERROR: Failed to save recharge history. Response: %s", string(body))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save recharge history",
		})
	}

	// Log recharge success
	log.Println("INFO: Recharge history saved successfully for hash:", hash)

	// Fetch IP details
	ipDetails, err := utils.GetIpDetails(c)
	if err != nil {
		log.Println("ERROR: Failed to fetch IP details:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch IP details",
		})
	}

	// Respond to client
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   fmt.Sprintf("%.2f₹ Added Successfully!", price),
		"ipDetails": ipDetails,
	})
}

// ExchangeRate handles exchange rate queries.

// ExchangeRate fetches the exchange rate and returns it as-is
func ExchangeRate(c echo.Context) error {
	log.Println("INFO: ExchangeRate endpoint invoked")

	// URL for the external API providing exchange rates
	apiURL := "https://own5k.in/p/trxprice.php" // Replace with the actual API URL

	// Make an HTTP GET request to the external API
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("ERROR: Failed to fetch exchange rate: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve exchange rate",
		})
	}
	defer resp.Body.Close()

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: Received non-200 status code: %d", resp.StatusCode)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve exchange rate",
		})
	}

	// Stream the response body directly to the client
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read response body: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process exchange rate",
		})
	}

	return c.Blob(http.StatusOK, "application/json", body)
}

// ToggleMaintenance handles toggling maintenance mode.
func ToggleMaintenance(c echo.Context) error {
	// Log the start of the function
	log.Println("INFO: Starting ToggleMaintenance handler")

	// Retrieve the database instance from the context
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	log.Println("INFO: Database instance retrieved successfully")

	// Define a struct for the input
	type RequestBody struct {
		RechargeType string `json:"rechargeType"`
		Status       bool   `json:"status"`
	}

	// Parse the input JSON
	var input RequestBody
	log.Println("INFO: Parsing input JSON")
	if err := c.Bind(&input); err != nil {
		log.Println("ERROR: Failed to bind input JSON:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	// Log the received input
	log.Printf("INFO: Received input - RechargeType: %s, Status: %t\n", input.RechargeType, input.Status)

	// Validate the input
	if input.RechargeType == "" {
		log.Println("ERROR: RechargeType is required")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "RechargeType is required"})
	}

	// Define the filter and update
	filter := bson.M{"recharge_type": input.RechargeType}
	update := bson.M{"$set": bson.M{"maintainance": input.Status}}

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
func GetMaintenanceStatus(c echo.Context) error {
	// Logic for retrieving maintenance status
	return c.JSON(http.StatusOK, map[string]string{"message": "Maintenance status retrieved successfully"})
}
