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
	"github.com/ranjankuldeep/fakeNumber/internal/database/services"
	serverscalc "github.com/ranjankuldeep/fakeNumber/internal/serversCalc"
	serversnextotpcalc "github.com/ranjankuldeep/fakeNumber/internal/serversNextOtpCalc"
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
		number, id, err := serverscalc.ExtractNumberServer9(apiURLRequest.URL, apiURLRequest.Headers)
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
	serviceName := c.QueryParam("serviceName")

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

	constructedOTPRequest, err := constructOtpUrl(server, serverData.APIKey, serverData.Token, id)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_SERVER"})
	}

	validOtp, err := fetchOTP(server, id, constructedOTPRequest)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	logs.Logger.Info(validOtp)
	var existingEntry models.TransactionHistory
	transactionCollection := models.InitializeTransactionHistoryCollection(db)

	err = transactionCollection.FindOne(ctx, bson.M{"id": id, "otp": validOtp}).Decode(&existingEntry)
	if err == mongo.ErrNoDocuments {
		logs.Logger.Info("i am printing continoulsy")
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
			OTP:         transaction.OTP,
			Ip:          formattedIpDetails,
		}
		err = services.OtpGetDetails(otpDetail)
		if err != nil {
			logs.Logger.Error(err)
		}
	}
	if validOtp != "" {
		if err := triggerNextOtp(db, server, serviceName, id); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "FAILED_TRIGGERING_NEXT_OTP"})
		}
	}
	return c.JSON(http.StatusOK, map[string]string{"otp": validOtp})
}

func triggerNextOtp(db *mongo.Database, server, serviceName, id string) error {
	serverNumber, _ := strconv.Atoi(server)
	serverListCollection := models.InitializeServerListCollection(db)

	filter := bson.M{"name": serviceName}

	var serverList models.ServerList
	err := serverListCollection.FindOne(context.Background(), filter).Decode(&serverList)
	if err != nil {
		log.Fatalf("Error finding server list: %v", err)
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
			nextOtpUrl := fmt.Sprintf("https://api.grizzlysms.com/stubs/handler_api.php?api_key=%s&action=setStatus&status=3&id%s143308304", secret.ApiKeyServer, id)
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
	var server models.Server
	serverCollection := models.InitializeServerCollection(db)
	err := serverCollection.FindOne(context.TODO(), bson.M{"server": serverNumber})
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

	var order models.Order
	orderCollection := models.InitializeOrderCollection(db)
	err = orderCollection.FindOne(ctx, bson.M{"numberId": id}).Decode(&order)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ORDER_NOT_FOUND"})
	}

	var user models.User
	userCollection := models.InitializeUserCollection(db)
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&user)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "USER_NOT_FOUND"})
	}

	serverData, err := getServerDataWithMaintenanceCheck(ctx, db, server)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	constructedNumberRequest, err := constructNumberUrl(server, serverData.APIKey, serverData.Token, id, order.Number)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "INVALID_SERVER"})
	}

	var transactionData models.TransactionHistory
	transactionCollection := models.InitializeTransactionHistoryCollection(db)

	err = transactionCollection.FindOne(ctx, bson.M{"userId": apiWalletUser.UserID.Hex(), "id": id}).Decode(&transactionData)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "TRANSACTION_HISTORY_NOT_FOUND"})
	}

	if transactionData.OTP != "" {
		orderCollection := models.InitializeOrderCollection(db)
		_, err = orderCollection.DeleteOne(ctx, bson.M{"numberId": id})
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "ORDER_NOT_FOUND"})
		} else {
			return c.JSON(http.StatusOK, map[string]string{"msg": "OTP_RECEIVED"})
		}
	}
	if transactionData.Status == "CANCELLED" {
		return c.JSON(http.StatusOK, map[string]string{"msg": "NUMBER_ALREADY_CANCELLED"})
	}
	_, _, err = fetchAndProcess(constructedNumberRequest.URL, server, id, db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

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

	if transaction.OTP == "" {
		update := bson.M{
			"$set": bson.M{"balance": newBalance},
		}
		filter := bson.M{"_id": apiWalletUser.UserID}

		_, err = apiWalletColl.UpdateOne(ctx, filter, update)
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	ipDetails, err := utils.GetIpDetails(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "ERROR_FETCHING_IP_DETAILS"})
	}
	services.NumberCancelDetails(user.Email, transaction.Service, price, server, int64(price), apiWalletUser.Balance, ipDetails)

	// Delete the order entry
	_, err = orderCollection.DeleteOne(ctx, bson.M{"numberId": id})
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "ORDER_ENTRY_NOT_FOUND"})
	}
	return nil
}

// Helper functions
func fetchAndProcess(apiURL, server, id string, db *mongo.Database) (bool, models.TransactionHistory, error) {
	var existingEntry models.TransactionHistory
	otpReceived := false

	resp, err := http.Get(apiURL)
	if err != nil {
		return false, existingEntry, fmt.Errorf("failed to fetch API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, existingEntry, errors.New("error occurred during API request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, existingEntry, fmt.Errorf("failed to read response body: %w", err)
	}
	responseData := string(body)

	if strings.TrimSpace(responseData) == "" {
		return false, existingEntry, errors.New("received empty response data")
	}
	collection := models.InitializeTransactionHistoryCollection(db)

	switch server {
	case "1":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") {
			err = collection.FindOne(context.TODO(), bson.M{"id": id, "status": "CANCELLED"}).Decode(&existingEntry)
			if err != nil && err != mongo.ErrNoDocuments {
				return false, existingEntry, err
			}
		}
		if strings.HasPrefix(responseData, "ACCESS_APPROVED") {
			otpReceived = true
		}

	case "2":
		var responseDataJSON map[string]interface{}
		err = json.Unmarshal(body, &responseDataJSON)
		if err != nil {
			return false, existingEntry, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		if responseDataJSON["status"] == "CANCELED" {
			err = collection.FindOne(context.TODO(), bson.M{"id": id, "status": "CANCELLED"}).Decode(&existingEntry)
			if err != nil && err != mongo.ErrNoDocuments {
				return false, existingEntry, err
			}
		}
		if responseDataJSON["status"] == "order has sms" {
			otpReceived = true
		}
	case "3":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") || strings.HasPrefix(responseData, "ALREADY_CANCELLED") ||
			strings.HasPrefix(responseData, "ACCESS_ACTIVATION") {
			err = collection.FindOne(context.TODO(), bson.M{"id": id, "status": "CANCELLED"}).Decode(&existingEntry)
			if err != nil && err != mongo.ErrNoDocuments {
				return false, existingEntry, err
			}
		}
	case "4", "5", "6":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") || strings.HasPrefix(responseData, "BAD_STATUS") ||
			strings.HasPrefix(responseData, "BAD_ACTION") || strings.HasPrefix(responseData, "NO_ACTIVATION") {
			err = collection.FindOne(context.TODO(), bson.M{"id": id, "status": "CANCELLED"}).Decode(&existingEntry)
			if err != nil && err != mongo.ErrNoDocuments {
				return false, existingEntry, err
			}
		}

	case "7", "8":
		var responseDataJSON map[string]interface{}
		err = json.Unmarshal(body, &responseDataJSON)
		if err != nil {
			return false, existingEntry, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		if success, ok := responseDataJSON["success"].(bool); ok && success {
			err = collection.FindOne(context.TODO(), bson.M{"id": id, "status": "CANCELLED"}).Decode(&existingEntry)
			if err != nil && err != mongo.ErrNoDocuments {
				return false, existingEntry, err
			}
		}

	case "9":
		if strings.HasPrefix(responseData, "success") {
			err = collection.FindOne(context.TODO(), bson.M{"id": id, "status": "CANCELLED"}).Decode(&existingEntry)
			if err != nil && err != mongo.ErrNoDocuments {
				return false, existingEntry, err
			}
		}
	case "10":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") {
			err = collection.FindOne(context.TODO(), bson.M{"id": id, "status": "CANCELLED"}).Decode(&existingEntry)
			if err != nil && err != mongo.ErrNoDocuments {
				return false, existingEntry, err
			}
		}

	case "11":
		var responseDataJSON map[string]interface{}
		err = json.Unmarshal(body, &responseDataJSON)
		if err != nil {
			return false, existingEntry, fmt.Errorf("failed to parse JSON response: %w", err)
		}

		if success, ok := responseDataJSON["success"].(bool); ok && success {
			err = collection.FindOne(context.TODO(), bson.M{"id": id, "status": "CANCELLED"}).Decode(&existingEntry)
			if err != nil && err != mongo.ErrNoDocuments {
				return false, existingEntry, err
			}
		}
		break
	default:
		return false, existingEntry, errors.New("invalid server value")
	}
	return otpReceived, existingEntry, nil
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

func fetchOTP(server, id string, otpRequest ApiRequest) (string, error) {
	otpData := OTPData{}
	switch server {
	case "1", "3", "4", "5", "6", "7", "8", "10":
		otp, err := serversotpcalc.GetOTPServer1(otpRequest.URL, otpRequest.Headers, id)
		if err != nil {
			return "", err
		}
		otpData.Code = otp
	case "2":
		otp, err := serversotpcalc.GetSMSTextsServer2(otpRequest.URL, id, otpRequest.Headers)
		if err != nil {
			return "", err
		}
		otpData.Code = otp
	case "9":
		otp, err := serversotpcalc.FetchTokenAndOTP(otpRequest.URL, id, otpRequest.Headers)
		if err != nil {
			return "", err
		}
		otpData.Code = otp
	case "11":
		otp, err := serversotpcalc.GetOTPServer11(otpRequest.URL, id)
		if err != nil {
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

func constructNumberUrl(server, apiKeyServer, token, id, number string) (ApiRequest, error) {
	switch server {
	case "1":
		return ApiRequest{
			URL:     fmt.Sprintf("https://fastsms.su/stubs/handler_api.php?api_key=%s&action=setStatus&id=%s&status=8", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "2":
		return ApiRequest{
			URL:     fmt.Sprintf("https://5sim.net/v1/user/cancel/%s", id),
			Headers: map[string]string{"Authorization": fmt.Sprintf("Bearer %s", token), "Accept": "application/json"},
		}, nil
	case "3":
		return ApiRequest{
			URL:     fmt.Sprintf("https://smshub.org/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "4":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.tiger-sms.com/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "5":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.grizzlysms.com/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "6":
		return ApiRequest{
			URL:     fmt.Sprintf("https://tempnum.org/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "7":
		return ApiRequest{
			URL:     fmt.Sprintf("https://smsbower.online/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "8":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.sms-activate.io/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "9":
		return ApiRequest{
			URL:     fmt.Sprintf("https://own5k.in/p/ccpay.php?type=cancel&number=%s", number),
			Headers: map[string]string{},
		}, nil
	case "10":
		return ApiRequest{
			URL:     fmt.Sprintf("https://sms-activation-service.com/stubs/handler_api?api_key=%s&action=setStatus&id=%s&status=8", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "11":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api2.sms-man.com/control/set-status?token=%s&request_id=%s&status=reject", token, id),
			Headers: map[string]string{},
		}, nil
	default:
		return ApiRequest{}, fmt.Errorf("INVLAID_SERVER_CHOICE")
	}
}
