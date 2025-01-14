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
	"sync"
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
	Name   string  `json:"name,omitempty"`
	Amount float64 `json:"amount,omitempty"`
	Date   string  `json:"date,omitempty"`
	Error  string  `json:"error,omitempty"`
}

type IpDetails struct {
	City            string `json:"city"`
	State           string `json:"state"`
	Pincode         string `json:"pincode"`
	Country         string `json:"country"`
	ServiceProvider string `json:"serviceProvider"`
	IP              string `json:"ip"`
}

var userLocks sync.Map

func RechargeUpiApi(c echo.Context) error {
	ctx := context.Background()
	db := c.Get("db").(*mongo.Database)

	transactionId := c.QueryParam("transactionId")
	userId := c.QueryParam("userId")

	if userId == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EMPTY_USER_ID"})
	}
	userLock := getUserLock(userId)

	userLock.Lock()
	defer userLock.Unlock()

	userObjectID, _ := primitive.ObjectIDFromHex(userId)
	var user models.User

	userCollection := models.InitializeUserCollection(db)
	err := userCollection.FindOne(context.TODO(), bson.M{"_id": userObjectID}).Decode(&user)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	var rechargeData models.RechargeAPI
	rechargeCollection := models.InitializeRechargeAPICollection(db)

	if err := rechargeCollection.FindOne(ctx, bson.M{"recharge_type": "upi"}).Decode(&rechargeData); err != nil {
		log.Println("Recharge data fetch error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	if rechargeData.Maintenance {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "UPI recharge is under maintenance."})
	}
	upiUrl := fmt.Sprintf("https://php.paidsms.in/u.php?txn=%s", transactionId)
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

	if upiData.Error != "" {
		log.Println("UPI API returned error:", upiData.Error)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Transaction Not Found. Please try again."})
	}

	var minimumRecharge models.MinimumRecharge
	minimumCollection := models.InitializeMinimumCollection(db)
	err = minimumCollection.FindOne(ctx, bson.M{}).Decode(&minimumRecharge)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Transaction Not Found. Please try again."})
	}

	if float64(upiData.Amount) < minimumRecharge.MinimumRecharge {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Recharge amount is less than %0.2f amount", minimumRecharge.MinimumRecharge)})
	}
	rechargeHistoryUrl := fmt.Sprintf("%sapi/save-recharge-history", os.Getenv("BASE_API_URL"))
	rechargePayload := map[string]interface{}{
		"userId":         userId,
		"transaction_id": transactionId,
		"amount":         upiData.Amount,
		"payment_type":   "upi",
		"status":         "Received",
	}
	rechargePayloadBytes, _ := json.Marshal(rechargePayload)
	rechargeResp, err := http.Post(rechargeHistoryUrl, "application/json", bytes.NewReader(rechargePayloadBytes))
	if err != nil {
		log.Printf("[ERROR] Recharge history save error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save recharge history."})
	}
	defer rechargeResp.Body.Close()
	responseBody, _ := ioutil.ReadAll(rechargeResp.Body)
	log.Printf("[INFO] Recharge history save response status: %d", rechargeResp.StatusCode)
	log.Printf("[INFO] Recharge history save response body: %s", string(responseBody))

	if rechargeResp.StatusCode == http.StatusBadRequest {
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

	var apiWalletUser models.ApiWalletUser
	apiWalletCollection := models.InitializeApiWalletuserCollection(db)
	err = apiWalletCollection.FindOne(ctx, bson.M{"userId": userObjectID}).Decode(&apiWalletUser)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch api wallet user"})
	}

	ipDetail, err := utils.ExtractIpDetails(c)
	if err != nil {
		logs.Logger.Error(err)
	}

	rechargeDetail := services.UpiRechargeDetails{
		Email:   user.Email,
		UserID:  userId,
		TrnID:   transactionId,
		Amount:  fmt.Sprintf("%0.2f", upiData.Amount),
		Balance: fmt.Sprintf("%0.2f", apiWalletUser.Balance),
		IP:      ipDetail,
	}
	err = services.UpiRechargeTeleBot(rechargeDetail)
	if err != nil {
		logs.Logger.Error(err)
		logs.Logger.Error("Unable to send upi recharge message")
	}
	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("%0.2f₹ Added Successfully!", upiData.Amount),
	})
}

type ResponseStructure struct {
	Status string `json:"status"`
}

func RechargeTrxApi(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	address := c.QueryParam("address")
	hash := c.QueryParam("hash")
	userId := c.QueryParam("userId")

	if address == "" || hash == "" || userId == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing required query parameters",
		})
	}

	userLock := getUserLock(userId)
	userLock.Lock()
	defer userLock.Unlock()

	exchangeRate, err := utils.FetchTRXPrice()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get exchange rate",
		})
	}

	trxApiURL := fmt.Sprintf("https://php.paidsms.in/tron/?type=txnid&address=%s&hash=%s", address, hash)
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
		Message string  `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&trxData); err != nil {
		log.Println("ERROR: Failed to decode TRX transaction response:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process transaction data",
		})
	}

	if trxData.SUCCESS == false {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": trxData.Message,
		})
	}

	// if trxData.TRX <= 1 {
	// 	return c.JSON(http.StatusBadRequest, map[string]string{
	// 		"error": "Invalid TRX transaction",
	// 	})
	// }

	price := trxData.TRX * exchangeRate
	amount := strconv.FormatFloat(price, 'f', 2, 64)
	rechargeHistoryPayload := map[string]interface{}{
		"userId":         userId,
		"transaction_id": hash,
		"amount":         amount,
		"payment_type":   "trx",
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
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "",
		})
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

	sentUrl := fmt.Sprintf("https://php.paidsms.in/tron/?type=send&from=%s&key=%s&to=%s", fromAddress, privateKey, toAddress)
	ipDetail, err := utils.ExtractIpDetails(c)
	if err != nil {
		logs.Logger.Error(err)
	}
	rechargeDetail := services.TrxRechargeDetails{
		Email:        user.Email,
		UserID:       userId,
		Trx:          fmt.Sprintf("%.2f", trxData.TRX),
		ExchangeRate: fmt.Sprintf("%0.2f", exchangeRate),
		Amount:       amount,
		Balance:      fmt.Sprintf("%.2f", apiWalletUser.Balance),
		Address:      fromAddress,
		SendTo:       toAddress,
		Status:       "",
		Hash:         hash,
		IP:           ipDetail,
	}
	response, err := http.Get(sentUrl)
	if err != nil {
		log.Fatalf("Failed to call URL: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatalf("Non-200 status code received: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}
	log.Printf("Raw Response: %s", string(body))
	var responseData ResponseStructure
	if err := json.Unmarshal(body, &responseData); err != nil {
		log.Fatalf("Failed to unmarshal response: %v", err)
	}
	logs.Logger.Info(responseData)
	if responseData.Status == "Fail" {
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
		logs.Logger.Infof("%+v", rechargeDetail)
		err = services.TrxRechargeTeleBot(rechargeDetail)
		if err != nil {
			logs.Logger.Error(err)
			logs.Logger.Info("recharget trx send failed")
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("%.2f₹ Added Successfully!", price),
		})
	}
	rechargeDetail.Status = "ok"
	logs.Logger.Infof("%+v", rechargeDetail)
	err = services.TrxRechargeTeleBot(rechargeDetail)
	if err != nil {
		logs.Logger.Error(err)
		logs.Logger.Info("recharget trx send failed")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("%.2f₹ Added Successfully!", price),
	})
}

func ExchangeRate(c echo.Context) error {
	log.Println("INFO: ExchangeRate endpoint invoked")
	apiURL := "https://php.paidsms.in/trxprice.php"

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

	if input.RechargeType == "" {
		log.Println("ERROR: RechargeType is required")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "RechargeType is required"})
	}
	filter := bson.M{"recharge_type": input.RechargeType}
	update := bson.M{"$set": bson.M{"maintenance": input.Status}}

	log.Printf("INFO: Upserting record with filter: %+v and update: %+v\n", filter, update)

	rechargeApiCol := db.Collection("recharge-apis")

	opts := options.Update().SetUpsert(true)
	result, err := rechargeApiCol.UpdateOne(context.TODO(), filter, update, opts)

	if err != nil {
		log.Println("ERROR: Failed to update or insert maintenance status:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	log.Printf("INFO: Successfully upserted maintenance status. Matched count: %d, Modified count: %d, Upserted ID: %v\n",
		result.MatchedCount, result.ModifiedCount, result.UpsertedID)

	response := map[string]interface{}{
		"message":       "Maintenance status updated successfully.",
		"matchedCount":  result.MatchedCount,
		"modifiedCount": result.ModifiedCount,
	}

	if result.UpsertedID != nil {
		response["upsertedId"] = result.UpsertedID
	}

	return c.JSON(http.StatusOK, response)
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

func getUserLock(userId string) *sync.Mutex {
	lock, ok := userLocks.Load(userId)
	if !ok {
		newLock := &sync.Mutex{}
		userLocks.Store(userId, newLock)
		return newLock
	}
	return lock.(*sync.Mutex)
}

func cleanupUserLock(userId string) {
	userLocks.Delete(userId)
}
