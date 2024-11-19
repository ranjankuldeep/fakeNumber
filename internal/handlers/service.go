package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/database/services"
	serverscalc "github.com/ranjankuldeep/fakeNumber/internal/serversCalc"
	serversotpcalc "github.com/ranjankuldeep/fakeNumber/internal/serversOtpCalc"
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

var numData NumberData

func HandleGetNumberRequest(c echo.Context) error {
	ctx := context.TODO()
	db := c.Get("db").(*mongo.Database)
	logs.Logger.Info("reached")

	// Get query parameters
	serverDataCode := c.QueryParam("code")
	apiKey := c.QueryParam("api_key")
	server := c.QueryParam("server")
	temp := c.QueryParam("serverName")
	logs.Logger.Infof("%s %s %s %s", serverDataCode, apiKey, server, temp)

	serviceName := strings.ReplaceAll(temp, "%", " ")
	serverNumber, _ := strconv.Atoi(server)

	if serviceName == "" || apiKey == "" || server == "" || serverDataCode == "" {
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
	logs.Logger.Info("token ", serverInfo.Token)
	// fetch id and numbers
	apiURLRequest, err := constructApiUrl(server, serverInfo.APIKey, serverInfo.Token, serverData)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Couldn't construcrt api url"})
	}
	logs.Logger.Info(fmt.Sprintf("url-%s", apiURLRequest.URL))
	// handler all the server case and extract id and number
	switch server {
	case "1":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "2":
		// Multiple OTP server with same url
		number, id, err := serverscalc.ExtractNumberServer2(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "3":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "4":
		// Single OTP server
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "5":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "6":
		// Single OTP server
		// Done
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "7":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "8":
		// Done
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "9":
		// Single OTP server
		// Done
		number, id, err := serverscalc.ExtractNumberServer9()
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "10":
		// Single OTP server
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		numData.Id = id
		numData.Number = number
	case "11":
		// TODO: FIX the Multi url get number
		// Multiple OTP server with different url
		number, id, err := serverscalc.ExtractNumberServer11(apiURLRequest.URL)
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "couldn't fetch the number"})
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
	_, err = apiWalletUserCollection.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"balance": newBalance}})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "FAILED_TO_UPDATE_USER_BALANCE"})
	}

	// Save transaction history
	transactionHistoryCollection := models.InitializeTransactionHistoryCollection(db)
	transaction := models.TransactionHistory{
		UserID:        apiWalletUser.UserID.Hex(),
		Service:       serviceName,
		TransactionID: numData.Id,
		Price:         fmt.Sprintf("%.2f", price),
		Server:        server,
		ID:            primitive.NewObjectID(),
		Number:        numData.Number,
		Status:        "FINISHED",
		DateTime:      time.Now().Format("2006-01-02T15:04:05"),
	}
	_, err = transactionHistoryCollection.InsertOne(ctx, transaction)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save transaction history."})
	}

	// Save order
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
	return c.JSON(http.StatusOK, map[string]string{"id": numData.Id, "number": numData.Number})
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

	// Server discount
	serverDiscountCollection := models.InitializeServerDiscountCollection(db)
	var serverDiscount models.ServerDiscount
	err = serverDiscountCollection.FindOne(ctx, bson.M{"server": server}).Decode(&serverDiscount)
	if err != nil && err != mongo.ErrNoDocuments {
		return 0, err
	}
	if err == nil {
		totalDiscount += round(serverDiscount.Discount, 2)
	}

	// Return the total discount rounded to 2 decimal places
	return round(totalDiscount, 2), nil
}

// Helper function to round to 2 decimal places
func round(val float64, precision int) float64 {
	format := fmt.Sprintf("%%.%df", precision)
	valStr := fmt.Sprintf(format, val)
	result, _ := strconv.ParseFloat(valStr, 64)
	return result
}

func formatDateTime() string {
	// Format the current date and time
	return time.Now().Format("01/02/2006T03:04:05 PM")
}

func removeHTMLTags(input string) string {
	// Replace specific HTML tags
	result := strings.ReplaceAll(input, "<br>", " ")
	return result
}

func HandleGetOtp(c echo.Context) error {
	ctx := context.Background()

	id := c.QueryParam("id")
	apiKey := c.QueryParam("api_key")
	server := c.QueryParam("server")

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

	var apiWalletUser models.ApiWalletUser
	apiWalletColl := models.InitializeApiWalletuserCollection(db)

	err := apiWalletColl.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_API_KEY"})
	}

	var userData models.User
	userCollection := models.InitializeUserCollection(db)

	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&userData)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_API_KEY"})
	}

	serverData, err := getServerDataWithMaintenanceCheck(ctx, db, server)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// construct api url and headers
	constructedOTPRequest, err := constructOtpUrl(server, serverData.APIKey, serverData.Token, id)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_SERVER"})
	}
	logs.Logger.Info(constructedOTPRequest.URL)

	validOtp, err := fetchOTP(server, id, constructedOTPRequest)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	logs.Logger.Info(validOtp)

	// Check if existing entry with id and otp exists
	var existingEntry models.TransactionHistory
	transactionCollection := models.InitializeTransactionHistoryCollection(db)

	err = transactionCollection.FindOne(ctx, bson.M{"id": id, "otp": validOtp}).Decode(&existingEntry)
	if err == mongo.ErrNoDocuments {
		formattedDateTime := formatDateTime()

		// Find the corresponding transaction history entry
		var transaction models.TransactionHistory
		err = transactionCollection.FindOne(ctx, bson.M{"id": id}).Decode(&transaction)
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Create a new transactionHistory instance
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
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		ipDetails, err := utils.GetIpDetails(c)
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		formattedIpDetails := removeHTMLTags(ipDetails)

		// Send OTP details
		otpDetail := services.OTPDetails{
			Email:       userData.Email,
			ServiceName: transaction.Service,
			Price:       transaction.Price,
			Server:      transaction.Server,
			Number:      transaction.Number,
			OTP:         transaction.OTP,
			Ip:          formattedIpDetails,
		}
		err = services.OtpGetDetails(otpDetail)
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	return c.JSON(http.StatusOK, map[string]string{"otp": validOtp})
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
func HandleCheckOTP(c echo.Context) error {
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

	// Fetch server data
	fmt.Println("DEBUG: Fetching server data for server 1")
	var serverData models.Server
	err := db.Collection("servers").FindOne(context.TODO(), bson.M{"server": 1}).Decode(&serverData)
	if err != nil {
		fmt.Println("ERROR: Failed to fetch server data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Server not found"})
	}
	fmt.Println("DEBUG: Retrieved server data:", serverData)

	// Call external OTP service
	encodedOtp := url.QueryEscape(otp)
	url := fmt.Sprintf("https://fastsms.su/stubs/handler_api.php?api_key=d91be54bb695297dd517edfdf7da5add&action=getOtp&sms=%s", encodedOtp)
	fmt.Println("DEBUG: Fetching OTP data from URL:", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR: Failed to fetch OTP data from external service:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch OTP data"})
	}
	defer resp.Body.Close()

	var data interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("ERROR: Failed to decode response from OTP service:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Invalid response from OTP service"})
	}
	fmt.Println("DEBUG: OTP service response:", data)

	// Handle the response
	switch v := data.(type) {
	case bool:
		fmt.Println("DEBUG: Response type is bool:", v)
		if !v {
			fmt.Println("ERROR: OTP not found")
			return c.JSON(http.StatusNotFound, echo.Map{"error": "OTP not found"})
		}
	case []interface{}:
		fmt.Println("DEBUG: Response type is array:", v)
		codes := []string{}
		if strings.Contains(v[0].(string), "|") {
			parts := strings.Split(v[0].(string), "|")
			fmt.Println("DEBUG: Splitting codes:", parts)
			for _, part := range parts {
				code := strings.TrimSpace(strings.ReplaceAll(part, `\d`, ""))
				if code != "" {
					codes = append(codes, code)
				}
			}
		} else {
			code := strings.TrimSpace(strings.ReplaceAll(v[0].(string), `\d`, ""))
			codes = append(codes, code)
		}
		fmt.Println("DEBUG: Extracted codes:", codes)

		// Search for matching codes in the database
		results, err := searchCodes(codes, db)
		if err != nil {
			fmt.Println("ERROR: Failed to search codes:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to search codes"})
		}
		fmt.Println("DEBUG: Search results:", results)

		if len(results) > 0 {
			fmt.Println("DEBUG: Found matching results")
			return c.JSON(http.StatusOK, echo.Map{"results": results})
		} else {
			fmt.Println("ERROR: No valid data found for the provided codes")
			return c.JSON(http.StatusNotFound, echo.Map{"error": "No valid data found for the provided codes"})
		}
	default:
		fmt.Println("ERROR: Unexpected response format:", v)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unexpected response format"})
	}

	// This line should never be reached
	fmt.Println("ERROR: Unhandled case reached")
	return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unhandled case"})
}

func HandleNumberCancel(c echo.Context) error {
	ctx := context.Background()
	id := c.QueryParam("id")
	apiKey := c.QueryParam("api_key")
	server := c.QueryParam("server")

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

	var apiWalletUser models.ApiWalletUser
	apiWalletColl := models.InitializeApiWalletuserCollection(db)

	err := apiWalletColl.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_API_KEY"})
	}

	_, err = getServerDataWithMaintenanceCheck(ctx, db, server)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// // construct api url and headers
	// constructedNumberRequest, err := constructNumberUrl(server, serverData.APIKey, serverData.Token, id)
	// if err != nil {
	// 	logs.Logger.Error(err)
	// 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_SERVER"})
	// }

	// validOtp, err := fetchOTP(server, id, constructedOTPRequest)
	// if err != nil {
	// 	logs.Logger.Error(err)
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	// }

	// Save transaction history logic here...
	// Process the transaction here

	// Respond with the extracted OTP
	// return c.JSON(http.StatusOK, map[string]string{"otp": validOtp})
	return nil
}

// Helper functions
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

func fetchOTP(server, id string, otpRequest ApiRequest) (string, error) {
	otpData := OTPData{}
	switch server {
	case "1", "3", "4", "5", "6", "7", "8", "10":
		otp, err := serversotpcalc.GetOTPServer1(otpRequest.URL, otpRequest.Headers, id)
		if err != nil {
			logs.Logger.Error(err)
			return "", err
		}
		otpData.Code = otp
	case "2":
		otp, err := serversotpcalc.GetSMSTextsServer2(otpRequest.URL, id, otpRequest.Headers)
		if err != nil {
			logs.Logger.Error(err)
			return "", err
		}
		otpData.Code = otp
	case "9":
		otp, err := serversotpcalc.FetchTokenAndOTP(otpRequest.URL, id, otpRequest.Headers)
		if err != nil {
			logs.Logger.Error(err)
			return "", err
		}
		otpData.Code = otp
	case "11":
		otp, err := serversotpcalc.GetOTPServer11(otpRequest.URL, id)
		if err != nil {
			logs.Logger.Error(err)
			return "", err
		}
		otpData.Code = otp

	default:
		return "", fmt.Errorf("INVALID_SERVER_CHOICE")
	}
	return otpData.Code, nil
}

func constructApiUrl(server, apiKeyServer string, apiToken string, data models.ServerData) (ApiRequest, error) {
	switch server {
	case "1":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://fastsms.su/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "2":
		return ApiRequest{
			URL: fmt.Sprintf("https://5sim.net/v1/user/buy/activation/india/any/%s", data.Code),
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", apiToken),
				"Accept":        "application/json",
			},
		}, nil

	case "3":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://smshub.org/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&operator=any&country=22&maxPrice=%s",
				apiKeyServer, data.Code, data.Price,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "4":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://api.tiger-sms.com/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "5":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://api.grizzlysms.com/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "6":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://tempnum.org/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "7":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://smsbower.online/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "8":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://api.sms-activate.io/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&operator=any&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "9":
		return ApiRequest{
			URL: fmt.Sprintf(
				"http://www.phantomunion.com:10023/pickCode-api/push/buyCandy?token=%s&businessCode=%s&quantity=1&country=IN&effectiveTime=10",
				apiToken, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil
	case "10":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://sms-activation-service.com/stubs/handler_api?api_key=%s&action=getNumber&service=%s&operator=any&country=22 ",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil
	case "11":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://api.sms-man.com/control/get-number?token=%s&application_id=1491&country_id=14&hasMultipleSms=false",
				apiToken,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	default:
		return ApiRequest{}, errors.New("invalid server value")
	}
}

func constructOtpUrl(server, apiKeyServer, token, id string) (ApiRequest, error) {
	var request ApiRequest
	request.Headers = map[string]string{}

	switch server {
	case "1":
		return ApiRequest{
			URL:     fmt.Sprintf("https://fastsms.su/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "2":
		return ApiRequest{
			URL:     fmt.Sprintf("https://5sim.net/v1/user/check/%s", id),
			Headers: map[string]string{"Authorization": fmt.Sprintf("Bearer %s", token), "Accept": "application/json"},
		}, nil
	case "3":
		return ApiRequest{
			URL:     fmt.Sprintf("https://smshub.org/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "4":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.tiger-sms.com/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "5":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.grizzlysms.com/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "6":
		return ApiRequest{
			URL:     fmt.Sprintf("https://tempnum.org/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "7":
		return ApiRequest{
			URL:     fmt.Sprintf("https://smsbower.online/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "8":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.sms-activate.io/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "9":
		return ApiRequest{
			URL:     fmt.Sprintf("http://www.phantomunion.com:10023/pickCode-api/push/sweetWrapper?token=%s&serialNumber=%s", token, id),
			Headers: map[string]string{},
		}, nil
	case "10":
		return ApiRequest{
			URL:     fmt.Sprintf("https://sms-activation-service.com/stubs/handler_api?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "11":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.sms-man.com/control/get-sms?token=%s&request_id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	default:
		return ApiRequest{}, fmt.Errorf("INVLAID_SERVER_CHOICE")
	}
}
