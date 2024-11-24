package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	serverscalc "github.com/ranjankuldeep/fakeNumber/internal/serversCalc"
	serversnextotpcalc "github.com/ranjankuldeep/fakeNumber/internal/serversNextOtpCalc"
	serversotpcalc "github.com/ranjankuldeep/fakeNumber/internal/serversOtpCalc"
	"github.com/ranjankuldeep/fakeNumber/internal/services"
	"github.com/ranjankuldeep/fakeNumber/internal/utils"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ApiRequest struct {
	URL     string
	Headers map[string]string
}

type ResponseData struct {
	ID     string
	Number string
}
type NumberData struct {
	Id     string
	Number string
}

type OTPData struct {
	Code string
}

type ServerSecrets struct {
	ApiKeyServer string
	Token        string
}

var numData NumberData

func HandleGetNumberRequest(c echo.Context) error {
	ctx := context.TODO()
	db := c.Get("db").(*mongo.Database)

	// Get query parameters
	serverDataCode := c.QueryParam("code")
	apiKey := c.QueryParam("api_key")
	server := c.QueryParam("server")
	temp := c.QueryParam("serverName")
	isMultiple := c.QueryParam("isMultiple")
	logs.Logger.Infof("%s %s %s %s", serverDataCode, apiKey, server, temp)

	serviceName := strings.ReplaceAll(temp, "%", " ")
	serverNumber, _ := strconv.Atoi(server)

	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid key"})
	}

	if serviceName == "" || server == "" || serverDataCode == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Service code, API key, and Server are required."})
	}

	// Fetch service details
	serverListCollection := models.InitializeServerListCollection(db)
	var service models.ServerList
	err := serverListCollection.FindOne(ctx, bson.M{"name": serviceName}).Decode(&service)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Service not found."})
	}

	// Fetch apiWalletUser details for calculating balance
	apiWalletUserCollection := models.InitializeApiWalletuserCollection(db)
	var apiWalletUser models.ApiWalletUser
	err = apiWalletUserCollection.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Invalid API key."})
	}

	// Fetch user details and return if user is blocked
	userCollection := models.InitializeUserCollection(db)
	var user models.User
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&user)
	// Check if the user is blocked
	if user.Blocked {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Your account is blocked, contact the Admin."})
	}

	//// Fetch server maintenance data
	// TODO: ALSO HADNLE THE MAITAINENCE
	serverCollection := models.InitializeServerCollection(db)
	var serverInfo models.Server
	err = serverCollection.FindOne(ctx, bson.M{"server": serverNumber}).Decode(&serverInfo)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Server not found."})
	}

	// Find the server list for the specified server name and server number
	serverListollection := models.InitializeServerListCollection(db)
	var serverList models.ServerList
	err = serverListollection.FindOne(ctx, bson.M{
		"name":           serviceName,
		"servers.server": serverNumber,
	}).Decode(&serverList)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Couldn't find serverlist"})
	}

	// Find the specific server data
	var serverData models.ServerData
	for _, s := range serverList.Servers {
		if s.Server == serverNumber {
			serverData = models.ServerData{
				Price:  s.Price,
				Code:   s.Code,
				Otp:    s.Otp,
				Server: serverNumber,
			}
		}
	}

	apiURLRequest, err := constructApiUrl(server, serverInfo.APIKey, serverInfo.Token, serverData, isMultiple)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Couldn't construcrt api url"})
	}
	logs.Logger.Info(fmt.Sprintf("url-%s", apiURLRequest.URL))
	switch server {
	case "1":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "2":
		// Multiple OTP server with same url
		number, id, err := serverscalc.ExtractNumberServer2(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "3":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "4":
		// Single OTP server
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "5":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "6":
		// Single OTP server
		// Done
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "7":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "8":
		// Done
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "9":
		// Single OTP server
		// Done
		number, id, err := serverscalc.ExtractNumberServer9(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "10":
		// Single OTP server
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	case "11":
		// Multiple OTP servers with different URLs
		number, id, err := serverscalc.ExtractNumberServer11(apiURLRequest.URL)
		if err != nil {
			if strings.Contains(err.Error(), "no_channels") {
				logs.Logger.Warn("No channels available. The channel limit has been reached.")
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
			}
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
		}
		numData.Id = id
		numData.Number = number
	}

	logs.Logger.Info(fmt.Sprintf("id-%s number-%s", numData.Id, numData.Number))

	// update the price with the discount
	price, _ := strconv.ParseFloat(serverData.Price, 64)
	discount, err := FetchDiscount(ctx, db, user.ID.Hex(), serviceName, serverNumber)
	price += discount

	// Check user balance
	if apiWalletUser.Balance < price {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INSUFFICENT_USER_BALANCE"})
	}

	// Deduct balance and save to DB
	newBalance := apiWalletUser.Balance - price
	roundedBalance := math.Round(newBalance*100) / 100
	_, err = apiWalletUserCollection.UpdateOne(ctx, bson.M{"userId": user.ID}, bson.M{"$set": bson.M{"balance": roundedBalance}})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "FAILED_TO_UPDATE_USER_BALANCE"})
	}

	transactionHistoryCollection := models.InitializeTransactionHistoryCollection(db)
	transaction := models.TransactionHistory{
		UserID:        apiWalletUser.UserID.Hex(),
		Service:       serviceName,
		TransactionID: numData.Id,
		Price:         fmt.Sprintf("%.2f", price),
		Server:        server,
		OTP:           "",
		ID:            primitive.NewObjectID(),
		Number:        numData.Number,
		Status:        "FINISHED",
		DateTime:      time.Now().Format("2006-01-02T15:04:05"),
	}
	_, err = transactionHistoryCollection.InsertOne(ctx, transaction)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save transaction history."})
	}

	orderCollection := models.InitializeOrderCollection(db)
	order := models.Order{
		ID:             primitive.NewObjectID(),
		UserID:         apiWalletUser.UserID,
		Service:        serviceName,
		Price:          price,
		Server:         serverNumber,
		NumberID:       numData.Id,
		Number:         numData.Number,
		OrderTime:      time.Now(),
		ExpirationTime: time.Now().Add(20 * time.Minute),
	}
	_, err = orderCollection.InsertOne(ctx, order)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save order."})
	}
	logs.Logger.Info(numData.Id, numData.Number)

	go func(id, number, userId string, db *mongo.Database, ctx context.Context) {
		defer func() {
			if r := recover(); r != nil {
				logs.Logger.Error("Recovered from panic in OTP handling goroutine:", r)
			}
		}()

		// Define the wait duration for each server case
		var waitDuration time.Duration
		switch server {
		case "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11":
			waitDuration = 3 * time.Minute
		}
		time.Sleep(waitDuration)

		// Fetch server data with maintenance check
		serverData, err := getServerDataWithMaintenanceCheck(ctx, db, server)
		if err != nil {
			logs.Logger.Error(err)
			return
		}

		// Check for existing OTP in the transaction collection
		var transactionData []models.TransactionHistory
		transactionCollection := models.InitializeTransactionHistoryCollection(db)

		filter := bson.M{
			"userId": userId,
			"id":     id,
		}
		cursor, err := transactionCollection.Find(ctx, filter)
		if err != nil {
			logs.Logger.Error(err)
			return
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &transactionData); err != nil {
			logs.Logger.Error(err)
			return
		}

		if len(transactionData) == 0 {
			return
		}

		otpArrived := false
		for _, transaction := range transactionData {
			if transaction.OTP != "" && transaction.OTP != "STATUS_WAIT_CODE" && transaction.OTP != "STATUS_CANCEL" {
				otpArrived = true
				break
			}
		}
		if otpArrived {
			logs.Logger.Infof("OTP already arrived for transaction %s, skipping third-party call.", id)
			return
		}

		// Construct number request URL
		constructedNumberRequest, err := constructNumberUrl(server, serverData.APIKey, serverData.Token, id, number)
		if err != nil {
			logs.Logger.Error(err)
			return
		}

		// Call third-party API to cancel the number
		err = CancelNumberThirdParty(constructedNumberRequest.URL, server, id, db, constructedNumberRequest.Headers)
		if err != nil {
			logs.Logger.Error(err)
			return
		}

		logs.Logger.Infof("Third-party call for transaction %s completed successfully.", id)
	}(numData.Id, numData.Number, apiWalletUser.UserID.Hex(), db, ctx)

	if numData.Id == "" || numData.Number == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "id": numData.Id, "number": numData.Number})
}

// Helper Functions
func FetchDiscount(ctx context.Context, db *mongo.Database, userId, sname string, server int) (float64, error) {
	totalDiscount := 0.0

	// User-specific discount
	userDiscountCollection := models.InitializeUserDiscountCollection(db)
	var userDiscount models.UserDiscount
	err := userDiscountCollection.FindOne(ctx, bson.M{"userId": userId, "service": sname, "server": server}).Decode(&userDiscount)
	if err != nil && err != mongo.ErrNoDocuments {
		return 0, err
	}
	if err == nil {
		totalDiscount += round(userDiscount.Discount, 2)
	}

	// Service discount
	serviceDiscountCollection := models.InitializeServiceDiscountCollection(db)
	var serviceDiscount models.ServiceDiscount
	err = serviceDiscountCollection.FindOne(ctx, bson.M{"service": sname, "server": server}).Decode(&serviceDiscount)
	if err != nil && err != mongo.ErrNoDocuments {
		return 0, err
	}
	if err == nil {
		totalDiscount += round(serviceDiscount.Discount, 2)
	}

	serverDiscountCollection := models.InitializeServerDiscountCollection(db)
	var serverDiscount models.ServerDiscount
	err = serverDiscountCollection.FindOne(ctx, bson.M{"server": server}).Decode(&serverDiscount)
	if err != nil && err != mongo.ErrNoDocuments {
		return 0, err
	}
	if err == nil {
		totalDiscount += round(serverDiscount.Discount, 2)
	}
	return round(totalDiscount, 2), nil
}

func round(val float64, precision int) float64 {
	format := fmt.Sprintf("%%.%df", precision)
	valStr := fmt.Sprintf(format, val)
	result, _ := strconv.ParseFloat(valStr, 64)
	return result
}

func formatDateTime() string {
	return time.Now().Format("01/02/2006T03:04:05 PM")
}

func removeHTMLTags(input string) string {
	result := strings.ReplaceAll(input, "<br>", " ")
	return result
}

func HandleGetOtp(c echo.Context) error {
	ctx := context.Background()
	id := c.QueryParam("id")
	apiKey := c.QueryParam("api_key")
	server := c.QueryParam("server")
	serviceName := c.QueryParam("serviceName") // new parameter

	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EMPTY_ID"})
	}
	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EMPTY_APIKEY"})
	}
	if server == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"errror": "EMPTY_SERVER"})
	}

	db := c.Get("db").(*mongo.Database)

	// Validate API key and get user data
	var apiWalletUser models.ApiWalletUser
	apiWalletColl := models.InitializeApiWalletuserCollection(db)
	err := apiWalletColl.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_API_KEY"})
	}

	var userData models.User
	userCollection := models.InitializeUserCollection(db)
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&userData)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_API_KEY"})
	}

	// Get server data
	serverData, err := getServerDataWithMaintenanceCheck(ctx, db, server)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Construct OTP URL
	constructedOTPRequest, err := constructOtpUrl(server, serverData.APIKey, serverData.Token, id)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_SERVER"})
	}

	// Fetch OTPs
	validOtpList, err := fetchOTP(server, id, constructedOTPRequest) // Assuming this returns []string
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	for _, validOtp := range validOtpList {
		var existingEntry models.TransactionHistory
		transactionCollection := models.InitializeTransactionHistoryCollection(db)

		err = transactionCollection.FindOne(ctx, bson.M{"id": id, "otp": validOtp}).Decode(&existingEntry)
		if err == mongo.ErrNoDocuments || err == mongo.ErrEmptySlice {
			formattedDateTime := formatDateTime()

			var transaction models.TransactionHistory
			err = transactionCollection.FindOne(ctx, bson.M{"id": id}).Decode(&transaction)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			numberHistory := models.TransactionHistory{
				ID:            primitive.NewObjectID(),
				UserID:        apiWalletUser.UserID.Hex(),
				Service:       transaction.Service,
				Price:         transaction.Price,
				Server:        server,
				TransactionID: id,
				OTP:           validOtp,
				Status:        "FINISHED",
				Number:        transaction.Number,
				DateTime:      formattedDateTime,
			}

			_, err = transactionCollection.InsertOne(ctx, numberHistory)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			ipDetails, err := utils.GetIpDetails(c)
			if err != nil {
				logs.Logger.Error(err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			formattedIpDetails := removeHTMLTags(ipDetails)

			otpDetail := services.OTPDetails{
				Email:       userData.Email,
				ServiceName: transaction.Service,
				Price:       transaction.Price,
				Server:      transaction.Server,
				Number:      transaction.Number,
				OTP:         validOtp,
				Ip:          formattedIpDetails,
			}
			err = services.OtpGetDetails(otpDetail)
			if err != nil {
				logs.Logger.Error(err)
			}
		}

		// Trigger the next OTP asynchronously for each OTP
		go func(otp string) {
			err := triggerNextOtp(db, server, serviceName, id)
			if err != nil {
				log.Printf("Error triggering next OTP for ID: %s, OTP: %s - %v", id, otp, err)
			} else {
				log.Printf("Successfully triggered next OTP for ID: %s, OTP: %s", id, otp)
			}
		}(validOtp)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "All OTPs processed successfully",
	})
}

func triggerNextOtp(db *mongo.Database, server, serviceName, id string) error {
	serverNumber, _ := strconv.Atoi(server)
	serverListCollection := models.InitializeServerListCollection(db)

	filter := bson.M{"name": serviceName}

	var serverList models.ServerList
	err := serverListCollection.FindOne(context.Background(), filter).Decode(&serverList)
	if err != nil {
		logs.Logger.Errorf("Error finding server list: %v", err)
		return err
	}

	var foundServer models.ServerData
	for _, server := range serverList.Servers {
		if server.Server == serverNumber {
			foundServer = server
			break
		}
	}

	if foundServer.Otp == "Multiple Otp" {
		switch serverNumber {
		case 1:
			secret, err := getApiKeyServer(db, serverNumber)
			if err != nil {
				return err
			}
			nextOtpUrl := fmt.Sprintf("https://fastsms.su/stubs/handler_api.php?api_key=%s&action=setStatus&id=%s&status=3", secret.ApiKeyServer, id)
			headers := map[string]string{}
			if err := serversnextotpcalc.CallNextOTPServerWaiting(nextOtpUrl, headers); err != nil {
				return err
			}
		case 3:
			secret, err := getApiKeyServer(db, serverNumber)
			if err != nil {
				return err
			}
			nextOtpUrl := fmt.Sprintf("https://smshub.org/stubs/handler_api.php?api_key=%s&action=setStatus&status=3&id=%s", secret.ApiKeyServer, id)
			headers := map[string]string{}
			if err := serversnextotpcalc.CallNextOTPServerRetry(nextOtpUrl, headers); err != nil {
				return err
			}
		case 5:
			secret, err := getApiKeyServer(db, serverNumber)
			if err != nil {
				return err
			}
			nextOtpUrl := fmt.Sprintf("https://api.grizzlysms.com/stubs/handler_api.php?api_key=%s&action=setStatus&status=3&id=%s", secret.ApiKeyServer, id)
			headers := map[string]string{}
			if err := serversnextotpcalc.CallNextOTPServerRetry(nextOtpUrl, headers); err != nil {
				return err
			}
		case 7:
			secret, err := getApiKeyServer(db, serverNumber)
			if err != nil {
				return err
			}
			nextOtpUrl := fmt.Sprintf("https://smsbower.online/stubs/handler_api.php?api_key=%s&action=setStatus&status=3&id=%s", secret.ApiKeyServer, id)
			headers := map[string]string{}
			if err := serversnextotpcalc.CallNextOTPServerRetry(nextOtpUrl, headers); err != nil {
				return err
			}
		case 8:
			secret, err := getApiKeyServer(db, serverNumber)
			if err != nil {
				return err
			}
			nextOtpUrl := fmt.Sprintf("https://api.sms-activate.io/stubs/handler_api.php?api_key=%s&action=setStatus&status=3&id=%s", secret.ApiKeyServer, id)
			headers := map[string]string{}
			if err := serversnextotpcalc.CallNextOTPServerRetry(nextOtpUrl, headers); err != nil {
				return err
			}
		case 11:
			secret, err := getApiKeyServer(db, serverNumber)
			if err != nil {
				return err
			}
			nextOtpUrl := fmt.Sprintf("https://api2.sms-man.com/control/set-status?token=%s&request_id=%s&status=retrysms", secret.Token, id)
			headers := map[string]string{}
			if err := serversnextotpcalc.CallNextOTPServerRetry(nextOtpUrl, headers); err != nil {
				return err
			}
		}
	}
	return nil
}

func getApiKeyServer(db *mongo.Database, serverNumber int) (ServerSecrets, error) {
	logs.Logger.Info(serverNumber)
	var server models.Server
	serverCollection := models.InitializeServerCollection(db)
	err := serverCollection.FindOne(context.TODO(), bson.M{"server": serverNumber}).Decode(&server)
	if err != nil {
		return ServerSecrets{}, fmt.Errorf("NO_SERVER_FOUND")
	}

	return ServerSecrets{
		ApiKeyServer: server.APIKey,
		Token:        server.Token,
	}, nil
}

func searchCodes(codes []string, db *mongo.Database) ([]string, error) {
	results := []string{}
	collection := db.Collection("serverList")

	for _, code := range codes {
		var serverData struct {
			Name    string `bson:"name"`
			Servers []struct {
				Code         string `bson:"code"`
				ServerNumber int    `bson:"server"`
			} `bson:"servers"`
		}

		err := collection.FindOne(context.TODO(), bson.M{"servers.code": code}).Decode(&serverData)
		if err != nil {
			log.Printf("Error searching for code %s: %v\n", code, err)
			continue
		}

		for _, server := range serverData.Servers {
			if server.Code == code && server.ServerNumber == 1 {
				results = append(results, serverData.Name)
				break
			}
		}
	}

	return results, nil
}

func HandleCancelOrder(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	id := c.QueryParam("id")
	userId := c.QueryParam("userId")
	if id == "" {
		fmt.Println("ERROR: id is missing")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "id is required"})
	}

	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		fmt.Println("ERROR: Invalid userId format:", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid userId format"})
	}

	orderCollection := models.InitializeOrderCollection(db)
	filter := bson.M{
		"userId":   userObjectID,
		"numberId": id,
	}

	deleteResult, err := orderCollection.DeleteOne(context.TODO(), filter)
	if err != nil {
		fmt.Println("ERROR: Unable to delete the order:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to cancel the order"})
	}

	if deleteResult.DeletedCount == 0 {
		fmt.Println("ERROR: No matching order found to delete")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "No matching order found"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "Order canceled successfully"})
}

func HandleCheckOTP(c echo.Context) error {
	fmt.Println("DEBUG: Received request to check OTP")
	otp := c.QueryParam("otp")
	apiKey := c.QueryParam("api_key")
	fmt.Println("DEBUG: Received request with OTP:", otp, "and API Key:", apiKey)

	// Validate input parameters
	if otp == "" {
		fmt.Println("ERROR: OTP is missing")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "OTP is required"})
	}
	if apiKey == "" {
		fmt.Println("ERROR: API Key is missing")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "API Key is required"})
	}
	db := c.Get("db").(*mongo.Database)
	var serverData models.Server
	err := db.Collection("servers").FindOne(context.TODO(), bson.M{"server": 1}).Decode(&serverData)
	if err != nil {
		fmt.Println("ERROR: Failed to fetch server data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Server not found"})
	}
	fmt.Println("DEBUG: Retrieved server data:", serverData)

	// First API Call: Fetch OTP data
	encodedOtp := url.QueryEscape(otp)
	url := fmt.Sprintf("https://fastsms.su/stubs/handler_api.php?api_key=%s&action=getOtp&sms=Dear user, your OTP is %s", serverData.APIKey, encodedOtp)
	fmt.Println("DEBUG: Fetching OTP data from URL:", url)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR: Failed to fetch OTP data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch OTP data"})
	}
	defer resp.Body.Close()

	var otpData []string
	if err := json.NewDecoder(resp.Body).Decode(&otpData); err != nil {
		fmt.Println("ERROR: Failed to decode OTP response:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Invalid OTP response format"})
	}
	fmt.Println("DEBUG: OTP service response:", otpData)

	// Handle empty OTP data
	// Handle empty OTP data
	if len(otpData) == 0 {
		fmt.Println("DEBUG: No OTP data found")
		return c.JSON(http.StatusOK, echo.Map{"results": []string{}})
	}

	// Extract the first element from the OTP data
	otpKey := otpData[0]
	fmt.Println("DEBUG: Extracted OTP key:", otpKey)

	// Second API Call: Fetch services data
	servicesURL := "https://fastsms.su/stubs/handler_api.php?api_key=d91be54bb695297dd517edfdf7da5add&action=getServices"
	fmt.Println("DEBUG: Fetching services data from URL:", servicesURL)

	resp, err = http.Get(servicesURL)
	if err != nil {
		fmt.Println("ERROR: Failed to fetch services data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch services data"})
	}
	defer resp.Body.Close()

	var servicesData map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&servicesData); err != nil {
		fmt.Println("ERROR: Failed to decode services response:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Invalid services response format"})
	}

	// Search for the OTP key in the services response
	serviceName, found := servicesData[otpKey]
	if !found {

		return c.JSON(http.StatusOK, echo.Map{"results": []string{}})
	}
	fmt.Println("DEBUG: Found service name for OTP key:", serviceName)

	// Return the service name in the results array
	return c.JSON(http.StatusOK, echo.Map{"results": []string{serviceName}})
}

func HandleNumberCancel(c echo.Context) error {
	ctx := context.Background()
	id := c.QueryParam("id")
	apiKey := c.QueryParam("api_key")
	server := c.QueryParam("server")

	fmt.Println("DEBUG: Received request with ID:", id, "API Key:", apiKey, "Server:", server)

	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EMPTY_ID"})
	}
	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EMPTY_APIKEY"})
	}
	if server == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"errror": "EMPTY_SERVER"})
	}

	db := c.Get("db").(*mongo.Database)
	var existingOrder models.Order
	orderCollection := models.InitializeOrderCollection(db)
	err := orderCollection.FindOne(ctx, bson.M{"numberId": id}).Decode(&existingOrder)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"errror": "number already cancelled"})
	}

	var apiWalletUser models.ApiWalletUser
	apiWalletColl := models.InitializeApiWalletuserCollection(db)
	err = apiWalletColl.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_API_KEY"})
	}

	var user models.User
	userCollection := models.InitializeUserCollection(db)
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&user)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "USER_NOT_FOUND"})
	}

	// case 1: if request comes with in 2 minutes
	timeDifference := time.Now().Sub(existingOrder.OrderTime)
	if timeDifference < 2*time.Minute {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "wait 2 mints before cancel"})
	}

	// case 2: if server in maintainance then send this response
	serverData, err := getServerDataWithMaintenanceCheck(ctx, db, server)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "under maintenance"})
	}

	// case 3: if otp already arrived
	var transactionData []models.TransactionHistory
	transactionCollection := models.InitializeTransactionHistoryCollection(db)

	filter := bson.M{
		"userId": apiWalletUser.UserID.Hex(),
		"id":     id,
	}
	cursor, err := transactionCollection.Find(ctx, filter)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "FAILED_TO_FETCH_TRANSACTION_HISTORY"})
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &transactionData); err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "FAILED_TO_PARSE_TRANSACTION_HISTORY"})
	}

	if len(transactionData) == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "TRANSACTION_HISTORY_NOT_FOUND"})
	}
	otpArrived := false
	for _, transaction := range transactionData {
		if transaction.OTP != "" && transaction.OTP != "STATUS_WAIT_CODE" && transaction.OTP != "STATUS_CANCEL" {
			otpArrived = true
		}
	}
	if otpArrived == true {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "otp already come"})
	}

	constructedNumberRequest, err := constructNumberUrl(server, serverData.APIKey, serverData.Token, id, existingOrder.Number)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_SERVER"})
	}

	_, err = orderCollection.DeleteOne(ctx, bson.M{"numberId": id})
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "ORDER_NOT_FOUND"})
	}

	err = CancelNumberThirdParty(constructedNumberRequest.URL, server, id, db, constructedNumberRequest.Headers)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// if no existing entry found with status cancelled then make a new transaction with status cancelled.
	var transaction models.TransactionHistory
	formattedData := formatDateTime()

	err = transactionCollection.FindOne(ctx, bson.M{"id": id}).Decode(&transaction)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	numberHistory := models.TransactionHistory{
		UserID:        transaction.UserID,
		Service:       transaction.Service,
		Price:         transaction.Price,
		Server:        server,
		TransactionID: id,
		OTP:           "",
		Status:        "CANCELLED",
		Number:        transaction.Number,
		DateTime:      formattedData,
	}

	_, err = transactionCollection.InsertOne(ctx, numberHistory)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	price, err := strconv.ParseFloat(transaction.Price, 64)
	newBalance := apiWalletUser.Balance + price
	newBalance = math.Round(newBalance*100) / 100

	update := bson.M{
		"$set": bson.M{"balance": newBalance},
	}
	balanceFilter := bson.M{"_id": apiWalletUser.UserID}

	_, err = apiWalletColl.UpdateOne(ctx, balanceFilter, update)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	ipDetails, err := utils.GetIpDetails(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "ERROR_FETCHING_IP_DETAILS"})
	}
	services.NumberCancelDetails(user.Email, transaction.Service, price, server, int64(price), apiWalletUser.Balance, ipDetails)
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func CancelNumberThirdParty(apiURL, server, id string, db *mongo.Database, headers map[string]string) error {
	logs.Logger.Infof("Number Cancel URL: %s", apiURL)
	client := &http.Client{}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create API request: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch API: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	responseData := string(body)
	logs.Logger.Info(responseData)
	if strings.TrimSpace(responseData) == "" {
		return errors.New("RECEIVED_EMTPY_RESPONSE_FROM_THIRD_PARTY_SERVER")
	}

	switch server {
	case "1":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") {
			return nil
		} else if strings.HasPrefix(responseData, "ACCESS_APPROVED") {
			return nil
		} else if strings.HasPrefix(responseData, "ACCESS_CANCEL_ALREADY") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))

	case "2":
		var responseDataJSON map[string]interface{}
		err = json.Unmarshal(body, &responseDataJSON)
		if err != nil {
			return fmt.Errorf("failed to parse JSON response: %w", err)
		}
		if responseDataJSON["status"] == "CANCELED" {
			return nil
		} else if responseDataJSON["status"] == "order has sms" {
			return nil
		} else if responseDataJSON["status"] == "order not found" {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FROM_THIRD_PARTY_SERVER_%s", server))

	case "3":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") || strings.HasPrefix(responseData, "ALREADY_CANCELLED") ||
			strings.HasPrefix(responseData, "ACCESS_ACTIVATION") {
			return nil
		} else {
			return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
		}
	case "4":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") {
			return nil
		} else if strings.HasPrefix(responseData, "EARLY_CANCEL_DENIED") {
			return errors.New("EARLY_CANCEL_DENIED")
		} else if strings.HasPrefix(responseData, "BAD_STATUS") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FROM_THIRD_PARTY_SERVER_%s", server))
	case "5":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") || strings.HasPrefix(responseData, "BAD_ACTION") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FROM_THIRD_PARTY_SERVER_%s", server))
	case "6":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") || strings.HasPrefix(responseData, "NO_ACTIVATION") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	case "8":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") || strings.HasPrefix(responseData, "BAD_STATUS") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	case "7":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") || strings.HasPrefix(responseData, "BAD_STATUS") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	case "9":
		if strings.HasPrefix(responseData, "success") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	case "10":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") || strings.HasPrefix(responseData, "ACCESS_CANCEL") || strings.HasPrefix(responseData, "ACCESS_CANCEL") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	case "11":
		var responseDataJSON map[string]interface{}
		err = json.Unmarshal(body, &responseDataJSON)
		if err != nil {
			return fmt.Errorf("failed to parse JSON response: %w", err)
		}
		if success, ok := responseDataJSON["success"].(bool); ok && success {
			return nil
		} else if responseDataJSON["error_code"] == "change_status" {
			return nil
		}
		errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	default:
		return errors.New("INVALID_SERVER_VALUE")
	}
	return nil
}

func getServerDataWithMaintenanceCheck(ctx context.Context, db *mongo.Database, server string) (models.Server, error) {
	serverNumber, _ := strconv.Atoi(server)
	var serverData models.Server
	collection := models.InitializeServerCollection(db)
	err := collection.FindOne(ctx, bson.M{"server": serverNumber}).Decode(&serverData)
	if err != nil {
		return models.Server{}, err
	}
	if serverData.Maintenance == true {
		return models.Server{}, fmt.Errorf("SERVER_UNDER_MAINTENANCE")
	}
	return serverData, nil
}

func fetchOTP(server, id string, otpRequest ApiRequest) ([]string, error) {
	otpData := []string{}
	switch server {
	case "1", "3", "4", "5", "6", "7", "8", "10":
		otp, err := serversotpcalc.GetOTPServer1(otpRequest.URL, otpRequest.Headers, id)
		if err != nil {
			return otpData, err
		}
		otpData = append(otpData, otp...)
	case "2":
		otp, err := serversotpcalc.GetSMSTextsServer2(otpRequest.URL, id, otpRequest.Headers)
		if err != nil {
			return otpData, err
		}
		otpData = append(otpData, otp...)
	case "9":
		otp, err := serversotpcalc.FetchTokenAndOTP(otpRequest.URL, id, otpRequest.Headers)
		if err != nil {
			return otpData, err
		}
		otpData = append(otpData, otp...)
	case "11":
		otp, err := serversotpcalc.GetOTPServer11(otpRequest.URL, id)
		if err != nil {
			return otpData, err
		}
		otpData = append(otpData, otp...)

	default:
		return otpData, fmt.Errorf("INVALID_SERVER_CHOICE")
	}
	return otpData, nil
}
