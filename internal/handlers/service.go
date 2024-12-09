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
	"go.mongodb.org/mongo-driver/mongo/options"
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

func FetchMarginAndExchangeRate(ctx context.Context, db *mongo.Database) (map[int]float64, map[int]float64, error) {
	serverCollection := models.InitializeServerCollection(db)
	marginMap := make(map[int]float64)
	exchangeRateMap := make(map[int]float64)

	cursor, err := serverCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch servers: %w", err)
	}
	defer cursor.Close(ctx)

	// Iterate over the fetched servers and populate the maps
	for cursor.Next(ctx) {
		var server models.Server
		if err := cursor.Decode(&server); err != nil {
			return nil, nil, fmt.Errorf("failed to decode server: %w", err)
		}
		marginMap[server.ServerNumber] = server.Margin
		exchangeRateMap[server.ServerNumber] = server.ExchangeRate
	}

	if err := cursor.Err(); err != nil {
		return nil, nil, fmt.Errorf("error while iterating over servers: %w", err)
	}
	return marginMap, exchangeRateMap, nil
}

func HandleGetNumberRequest(c echo.Context) error {
	ctx := context.TODO()
	db := c.Get("db").(*mongo.Database)
	apiKey := c.QueryParam("apikey")
	server := c.QueryParam("server")
	code := c.QueryParam("code")
	otp := c.QueryParam("otptype")

	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty api key"})
	}
	if server == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty server value"})
	}
	if otp == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty otp value"})
	}
	if otp != "single" && otp != "multiple" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid otp type"})
	}
	if code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty code value"})
	}
	serverNumber, _ := strconv.Atoi(server)

	serverCollection := models.InitializeServerCollection(db)
	var server0 models.Server
	err := serverCollection.FindOne(ctx, bson.M{"server": 0}).Decode(&server0)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if server0.Maintenance == true {
		return c.JSON(http.StatusOK, map[string]string{"error": "site is under maintenance"})
	}

	apiWalletUserCollection := models.InitializeApiWalletuserCollection(db)
	var apiWalletUser models.ApiWalletUser
	err = apiWalletUserCollection.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid api key"})
	}

	var user models.User
	userCollection := models.InitializeUserCollection(db)
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&user)
	if user.Blocked == true {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "account blocked"})
	}

	var serverInfo models.Server
	err = serverCollection.FindOne(ctx, bson.M{"server": serverNumber}).Decode(&serverInfo)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "server not found"})
	}
	if serverInfo.Maintenance {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "server under maintenance"})
	}
	if serverInfo.Block == true {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "invalid server number"})
	}

	var serviceList models.ServerList
	serverListollection := models.InitializeServerListCollection(db)
	err = serverListollection.FindOne(ctx, bson.M{
		"servers.server": serverNumber,
		"servers.code":   code,
	}).Decode(&serviceList)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	var serverData models.ServerData
	for _, s := range serviceList.Servers {
		if s.Server == serverNumber {
			serverData = models.ServerData{
				Price:  s.Price,
				Code:   s.Code,
				Otp:    s.Otp,
				Server: serverNumber,
			}
		}
	}

	isMultiple := "true"
	if otp == "single" {
		isMultiple = "false"
	}

	var serviceName string
	for _, s := range serviceList.Servers {
		if s.Server == serverNumber && s.Code == code {
			serviceName = serviceList.Name
			break
		}
	}

	price, _ := strconv.ParseFloat(serverData.Price, 64)
	discount, err := FetchDiscount(ctx, db, user.ID.Hex(), serviceName, serverNumber)
	price += discount
	if apiWalletUser.Balance < price {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "low balance"})
	}

	apiURLRequest, err := constructApiUrl(db, server, serverInfo.APIKey, serverInfo.Token, serverData, isMultiple)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	numData, err := ExtractNumber(server, apiURLRequest)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	newBalance := apiWalletUser.Balance - price
	roundedBalance := math.Round(newBalance*100) / 100
	roundedPrice := math.Round(price*100) / 100

	session, err := db.Client().StartSession()
	if err != nil {
		logs.Logger.Error("Failed to start session:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to start transaction session"})
	}
	defer session.EndSession(context.Background())
	_, err = session.WithTransaction(context.Background(), func(sc mongo.SessionContext) (interface{}, error) {
		maxRetries := 3
		retryCount := 0
		for retryCount < maxRetries {
			updateResult, err := apiWalletUserCollection.UpdateOne(
				sc,
				bson.M{"userId": user.ID},
				bson.M{"$inc": bson.M{"balance": -roundedPrice}},
			)
			if err != nil {
				logs.Logger.Error("Failed to decrement balance (attempt", retryCount+1, "):", err)
				retryCount++
				time.Sleep(2 * time.Second)
				continue
			}
			if updateResult.ModifiedCount == 0 {
				logs.Logger.Warn("Balance update resulted in no changes (attempt", retryCount+1, ")")
				retryCount++
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}

		// if retryCount == maxRetries {
		// 	logs.Logger.Error("Failed to decrement balance after", maxRetries, "attempts")
		// 	return nil, errors.New("failed to decrement balance after multiple retries")
		// }
		transactionHistoryCollection := models.InitializeTransactionHistoryCollection(db)
		transaction := models.TransactionHistory{
			UserID:        apiWalletUser.UserID.Hex(),
			Service:       serviceName,
			TransactionID: numData.Id,
			Price:         fmt.Sprintf("%.2f", price),
			Server:        server,
			OTP:           []string{},
			ID:            primitive.NewObjectID(),
			Number:        numData.Number,
			Status:        "PENDING",
			DateTime:      time.Now().In(time.FixedZone("IST", 5*3600+30*60)).Format("2006-01-02T15:04:05"),
			CreatedAt:     time.Now(),
		}
		_, err = transactionHistoryCollection.InsertOne(sc, transaction)
		if err != nil {
			logs.Logger.Error("Failed to insert transaction history:", err)
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		logs.Logger.Error("Transaction failed:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	logs.Logger.Info("Successfully updated balance and created transaction history.")
	var expirationTime time.Time
	switch server {
	case "1", "2", "3", "4", "5", "6", "8", "9", "10", "11":
		expirationTime = time.Now().Add(19 * time.Minute)
	case "7":
		expirationTime = time.Now().Add(9 * time.Minute)
	}

	orderCollection := models.InitializeOrderCollection(db)
	order := models.Order{
		ID:             primitive.NewObjectID(),
		UserID:         apiWalletUser.UserID,
		Service:        serviceName,
		Price:          price,
		NumberType:     map[string]string{"true": "Multiple", "false": "Single"}[isMultiple],
		Server:         serverNumber,
		NumberID:       numData.Id,
		Number:         numData.Number,
		OrderTime:      time.Now(),
		ExpirationTime: expirationTime,
	}
	_, err = orderCollection.InsertOne(ctx, order)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	logs.Logger.Info(numData.Id, numData.Number)
	if numData.Id == "" || numData.Number == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
	}

	ipDetail, err := utils.ExtractIpDetails(c)
	if err != nil {
		logs.Logger.Error(err)
	}

	numberDetails := services.NumberDetails{
		Email:       user.Email,
		ServiceName: serviceName,
		ServiceCode: serverData.Code,
		Price:       fmt.Sprintf("%.2f", price),
		Server:      server,
		Balance:     fmt.Sprintf("%.2f", roundedBalance),
		Number:      numData.Number,
		Ip:          ipDetail,
	}
	err = services.NumberGetDetails(numberDetails)
	if err != nil {
		logs.Logger.Error(err)
		logs.Logger.Info("Number details send failed")
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "id": numData.Id, "number": numData.Number})
}

func ExtractNumber(server string, apiURLRequest ApiRequest) (NumberData, error) {
	switch server {
	case "1":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "2":
		// Multiple OTP server with same url
		number, id, err := serverscalc.ExtractNumberServer2(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "3":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "4":
		// Single OTP server
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "5":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "6":
		// Single OTP server
		// Done
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "7":
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "8":
		// Done
		// Multiple OTP server with same url
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "9":
		// Single OTP server
		// Done
		number, id, err := serverscalc.ExtractNumberServer9(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			logs.Logger.Error(err)
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "10":
		// Single OTP server
		id, number, err := serverscalc.ExtractNumberServerFromAccess(apiURLRequest.URL, apiURLRequest.Headers)
		if err != nil {
			logs.Logger.Error(err)
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	case "11":
		// Multiple OTP servers with different URLs
		number, id, err := serverscalc.ExtractNumberServer11(apiURLRequest.URL)
		if err != nil {
			if strings.Contains(err.Error(), "no_channels") {
				logs.Logger.Warn("No channels available. The channel limit has been reached.")
				return NumberData{}, fmt.Errorf("no stock")
			}
			return NumberData{}, fmt.Errorf("no stock")
		}
		return NumberData{
			Id:     id,
			Number: number,
		}, nil
	}
	return NumberData{}, nil
}
func FetchDiscount(ctx context.Context, db *mongo.Database, userId, sname string, server int) (float64, error) {
	totalDiscount := 0.0
	userIdObject, _ := primitive.ObjectIDFromHex(userId)

	// User-specific discount
	userDiscountCollection := models.InitializeUserDiscountCollection(db)
	var userDiscount models.UserDiscount
	err := userDiscountCollection.FindOne(ctx, bson.M{"userId": userIdObject, "service": sname, "server": server}).Decode(&userDiscount)
	if err != nil && err != mongo.ErrNoDocuments {
		return 0, err
	}

	totalDiscount += round(userDiscount.Discount, 2)

	// Service discount
	serviceDiscountCollection := models.InitializeServiceDiscountCollection(db)
	var serviceDiscount models.ServiceDiscount
	err = serviceDiscountCollection.FindOne(ctx, bson.M{"service": sname, "server": server}).Decode(&serviceDiscount)
	if err != nil && err != mongo.ErrNoDocuments {
		return 0, err
	}
	totalDiscount += round(serviceDiscount.Discount, 2)

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

func FormatDateTime() string {
	return time.Now().In(time.FixedZone("IST", 5*3600+30*60)).Format("2006-01-02T15:04:05")
}

func removeHTMLTags(input string) string {
	result := strings.ReplaceAll(input, "<br>", " ")
	return result
}

func HandleGetOtp(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	ctx := context.Background()
	id := c.QueryParam("id")
	apiKey := c.QueryParam("apikey")
	server := c.QueryParam("server")

	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "empty id"})
	}
	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "empty api key"})
	}
	if server == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"errror": "empty server"})
	}

	serverCollection := models.InitializeServerCollection(db)
	var server0 models.Server
	err := serverCollection.FindOne(ctx, bson.M{"server": 0}).Decode(&server0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if server0.Maintenance == true {
		return c.JSON(http.StatusOK, map[string]string{"error": "under maintenance"})
	}

	var apiWalletUser models.ApiWalletUser
	apiWalletColl := models.InitializeApiWalletuserCollection(db)
	err = apiWalletColl.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid api key"})
	}

	var transaction models.TransactionHistory
	transactionCollection := models.InitializeTransactionHistoryCollection(db)
	err = transactionCollection.FindOne(ctx, bson.M{"id": id, "server": server}).Decode(&transaction)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if transaction.Status == "CANCELLED" {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok",
			"otp":    "number cancelled",
		})
	}
	var userData models.User
	userCollection := models.InitializeUserCollection(db)
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&userData)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid api key"})
	}

	serverData, err := getServerDataWithMaintenanceCheck(db, server)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if serverData.Block == true {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid server number"})
	}

	constructedOTPRequest, err := constructOtpUrl(server, serverData.APIKey, serverData.Token, id)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	validOtpList, err := fetchOTP(server, id, constructedOTPRequest)
	if err != nil && err.Error() == "ACCESS_CANCEL" {
		formattedData := FormatDateTime()

		var transaction models.TransactionHistory
		transactionCollection := models.InitializeTransactionHistoryCollection(db)
		err = transactionCollection.FindOne(ctx, bson.M{"id": id, "server": server}).Decode(&transaction)
		if err == mongo.ErrEmptySlice || err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid server number"})
		}
		if err != nil {
			logs.Logger.Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}

		if len(transaction.OTP) == 0 {
			transactionUpdateFilter := bson.M{"id": id, "server": server}
			transactionpdate := bson.M{
				"$set": bson.M{
					"status":    "CANCELLED",
					"date_time": formattedData,
				},
			}
			_, err = transactionCollection.UpdateOne(ctx, transactionUpdateFilter, transactionpdate)
			if err != nil {
				logs.Logger.Error(err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
		}
	}

	for _, validOtp := range validOtpList {
		transactionCollection := models.InitializeTransactionHistoryCollection(db)
		filter := bson.M{"id": id, "otp": validOtp, "server": server}
		var existingEntry models.TransactionHistory
		err = transactionCollection.FindOne(ctx, filter).Decode(&existingEntry)
		if err == mongo.ErrNoDocuments {
			formattedDateTime := FormatDateTime()
			update := bson.M{
				"$addToSet": bson.M{"otp": validOtp},
				"$set": bson.M{
					"status":    "SUCCESS",
					"date_time": formattedDateTime,
				},
			}

			filter := bson.M{"id": id}
			_, err = transactionCollection.UpdateOne(ctx, filter, update)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			ipDetail, err := utils.ExtractIpDetails(c)
			if err != nil {
				logs.Logger.Error(err)
			}

			otpDetail := services.OTPDetails{
				Email:       userData.Email,
				ServiceName: transaction.Service,
				ServiceCode: existingEntry.Service,
				Price:       transaction.Price,
				Server:      transaction.Server,
				Number:      transaction.Number,
				OTP:         validOtp,
				Ip:          ipDetail,
			}

			err = services.OtpGetDetails(otpDetail)
			if err != nil {
				logs.Logger.Error(err)
				logs.Logger.Error("Unable to send message")
			}

			go func(otp string) {
				err := triggerNextOtp(db, server, transaction.Service, id)
				if err != nil {
					log.Printf("Error triggering next OTP for ID: %s, OTP: %s - %v", id, otp, err)
				} else {
					log.Printf("Successfully triggered next OTP for ID: %s, OTP: %s", id, otp)
				}
			}(validOtp)

			recentOtpCollection := models.InitializeVerifyRecentOTPCollection(db)
			recentOtpFilter := bson.M{"transaction_id": id}
			recentOtpUpdate := bson.M{
				"$set": bson.M{
					"otp":       validOtp,
					"updatedAt": time.Now(),
				},
				"$setOnInsert": bson.M{
					"transaction_id": id,
					"createdAt":      time.Now(),
				},
			}
			_, err = recentOtpCollection.UpdateOne(ctx, recentOtpFilter, recentOtpUpdate, options.Update().SetUpsert(true))
			if err != nil {
				logs.Logger.Error("Failed to insert or update recent OTP:", err)
			}
		}
	}

	recentOtpCollection := models.InitializeVerifyRecentOTPCollection(db)
	var recentOtp models.RecentOTP
	err = recentOtpCollection.FindOne(ctx, bson.M{"transaction_id": id}).Decode(&recentOtp)
	if err == mongo.ErrEmptySlice || err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok",
			"otp":    "waiting for otp",
		})
	}
	if err != nil {
		logs.Logger.Error("Failed to fetch recent OTP:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ok",
		"otp":    recentOtp.OTP,
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
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty id"})
	}

	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty id"})
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
	encodedOtp := url.QueryEscape(otp)
	url := fmt.Sprintf("https://fastsms.su/stubs/handler_api.php?api_key=%s&action=getOtp&sms=%s", serverData.APIKey, encodedOtp)
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
	logs.Logger.Info(otpData)
	if len(otpData) == 0 {
		fmt.Println("DEBUG: No OTP data found")
		return c.JSON(http.StatusOK, echo.Map{"results": []string{}})
	}
	otpKey := otpData[0]
	otpkeys := strings.Split(otpKey, "|")
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
	ServicesNames := []string{}
	for _, service_name := range otpkeys {
		if serviceName, exists := servicesData[service_name]; exists {
			ServicesNames = append(ServicesNames, serviceName)
		} else {
			fmt.Println("DEBUG: Key not found in servicesData:", service_name)
		}
	}
	logs.Logger.Info(ServicesNames)
	return c.JSON(http.StatusOK, ServicesNames)
}

func HandleNumberCancel(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	apiKey := c.QueryParam("apikey")
	server := c.QueryParam("server")
	id := c.QueryParam("id")

	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty key"})
	}
	if server == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty server number"})
	}
	if id == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty id"})
	}
	serverNumber, _ := strconv.Atoi(server)

	serverCollection := models.InitializeServerCollection(db)
	var server0 models.Server
	err := serverCollection.FindOne(context.TODO(), bson.M{"server": 0}).Decode(&server0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if server0.Maintenance == true {
		return c.JSON(http.StatusOK, map[string]string{"error": "site is under maintenance"})
	}

	var apiWalletUser models.ApiWalletUser
	apiWalletColl := models.InitializeApiWalletuserCollection(db)
	err = apiWalletColl.FindOne(context.TODO(), bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid api key"})
	}

	var user models.User
	userCollection := models.InitializeUserCollection(db)
	userCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = userCollection.FindOne(userCtx, bson.M{"_id": apiWalletUser.UserID}).Decode(&user)
	if err != nil || err == mongo.ErrEmptySlice {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user not found"})
	}

	var existingOrder models.Order
	orderCollection := models.InitializeOrderCollection(db)
	err = orderCollection.FindOne(context.TODO(), bson.M{"numberId": id}).Decode(&existingOrder)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"errror": "number already cancelled"})
	}

	var serverList models.ServerList
	serverListollection := models.InitializeServerListCollection(db)
	sererListCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = serverListollection.FindOne(sererListCtx, bson.M{
		"name":           existingOrder.Service,
		"servers.server": serverNumber,
	}).Decode(&serverList)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	var serverDataInfo models.ServerData
	for _, s := range serverList.Servers {
		if s.Server == serverNumber {
			serverDataInfo = models.ServerData{
				Code: s.Code,
			}
		}
	}

	// timeDifference := time.Now().Sub(existingOrder.OrderTime)
	// if timeDifference < 2*time.Minute {
	// 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "wait 2 mints before cancel"})
	// }

	serverData, err := getServerDataWithMaintenanceCheck(db, server)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "under maintenance"})
	}
	if serverData.Block == true {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid server number"})
	}

	var transactionData models.TransactionHistory
	transactionCollection := models.InitializeTransactionHistoryCollection(db)

	filter := bson.M{
		"userId": apiWalletUser.UserID.Hex(),
		"id":     id,
		"server": server,
	}
	err = transactionCollection.FindOne(context.TODO(), filter).Decode(&transactionData)
	if err == mongo.ErrEmptySlice || err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid server"})
	}
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "FAILED_TO_FETCH_TRANSACTION_HISTORY_DATA"})
	}

	otpArrived := false
	if len(transactionData.OTP) != 0 {
		otpArrived = true
		return c.JSON(http.StatusNotFound, map[string]string{"error": "TRANSACTION_HISTORY_NOT_FOUND"})
	}
	if otpArrived == true {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "otp already come"})
	}

	constructedNumberRequest, err := ConstructNumberUrl(server, serverData.APIKey, serverData.Token, id, existingOrder.Number)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = CancelNumberThirdParty(constructedNumberRequest.URL, server, id, db, constructedNumberRequest.Headers)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	formattedData := FormatDateTime()
	var transaction models.TransactionHistory
	err = transactionCollection.FindOne(context.TODO(), bson.M{"id": id}).Decode(&transaction)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logs.Logger.Infof("handled request %+v", id)

	price, err := strconv.ParseFloat(transaction.Price, 64)
	if err != nil {
		logs.Logger.Error(err)
	}
	newBalance := apiWalletUser.Balance + price
	newBalance = math.Round(newBalance*100) / 100
	price = math.Round(price*100) / 100

	session, err := db.Client().StartSession()
	if err != nil {
		logs.Logger.Error("Failed to start session:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to start transaction session"})
	}
	defer session.EndSession(context.Background())
	_, err = session.WithTransaction(context.Background(), func(sc mongo.SessionContext) (interface{}, error) {
		const maxRetries = 3
		const retryInterval = time.Second * 2

		for attempt := 1; attempt <= maxRetries; attempt++ {
			balanceUpdate := bson.M{
				"$inc": bson.M{"balance": price},
			}
			balanceFilter := bson.M{"userId": apiWalletUser.UserID}
			balanceResult, err := apiWalletColl.UpdateOne(sc, balanceFilter, balanceUpdate)
			if err != nil {
				logs.Logger.Errorf("Attempt %d: Error updating balance: %v", attempt, err)
				if attempt < maxRetries {
					time.Sleep(retryInterval)
					continue
				}
				return nil, err
			}

			if balanceResult.ModifiedCount == 1 {
				logs.Logger.Infof("Balance successfully updated for user %s after %d attempt(s)", apiWalletUser.UserID, attempt)
				break
			} else {
				logs.Logger.Warnf("Attempt %d: No document modified for user %s", attempt, apiWalletUser.UserID)
				if attempt < maxRetries {
					time.Sleep(retryInterval)
					continue
				}
				return nil, errors.New("failed to update balance after multiple retries")
			}
		}

		for attempt := 1; attempt <= maxRetries; attempt++ {
			transactionUpdateFilter := bson.M{"id": id, "server": server}
			transactionUpdate := bson.M{
				"$set": bson.M{
					"status":    "CANCELLED",
					"date_time": formattedData,
				},
			}

			transactionResult, err := transactionCollection.UpdateOne(sc, transactionUpdateFilter, transactionUpdate)
			if err != nil {
				logs.Logger.Errorf("Attempt %d: Error updating transaction status: %v", attempt, err)
				if attempt < maxRetries {
					time.Sleep(retryInterval)
					continue
				}
				return nil, err
			}

			if transactionResult.ModifiedCount == 1 {
				logs.Logger.Infof("Transaction successfully updated for ID %s after %d attempt(s)", id, attempt)
				break
			} else {
				logs.Logger.Warnf("Attempt %d: No document modified for ID %s with filter %v", attempt, id, transactionUpdateFilter)
				if attempt < maxRetries {
					time.Sleep(retryInterval)
					continue
				}
				return nil, errors.New("failed to update transaction status after multiple retries")
			}
		}

		return nil, nil
	})
	if err != nil {
		logs.Logger.Error("Transaction failed:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	logs.Logger.Infof("Successfully updated balance and transaction status for ID %s", id)

	_, err = orderCollection.DeleteOne(context.TODO(), bson.M{"numberId": id})
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "ORDER_NOT_FOUND"})
	}

	ipDetail, err := utils.ExtractIpDetails(c)
	if err != nil {
		logs.Logger.Error(err)
	}
	cancelDetail := services.CancelDetails{
		Email:       user.Email,
		ServiceName: transaction.Service,
		ServiceCode: serverDataInfo.Code,
		Price:       transaction.Price,
		Server:      transaction.Server,
		Balance:     fmt.Sprintf("%.2f", newBalance),
		Number:      transaction.Number,
		IP:          ipDetail,
	}

	err = services.NumberCancelDetails(cancelDetail)
	if err != nil {
		logs.Logger.Error(err)
	}
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
	logs.Logger.Infof("Number Cancel Response %+v", responseData)

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
		} else if strings.HasPrefix(responseData, "STATUS_CANCEL") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))

	case "2":
		if strings.Contains(responseData, "order has sms") {
			return nil
		} else if strings.Contains(responseData, "order not found") {
			return nil
		}
		var responseDataJSON map[string]interface{}
		err = json.Unmarshal(body, &responseDataJSON)
		if err != nil {
			return fmt.Errorf("failed to parse JSON response: %w", err)
		}
		if responseDataJSON["status"] == "CANCELED" {
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
			return errors.New("BAD_STATUS")
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FROM_THIRD_PARTY_SERVER_%s", server))
	case "5":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") {
			return nil
		} else if strings.HasPrefix(responseData, "BAD_ACTION") {
			return errors.New("BAD_ACTION")
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FROM_THIRD_PARTY_SERVER_%s", server))
	case "6":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") {
			return nil
		} else if strings.HasPrefix(responseData, "NO_ACTIVATION") {
			return errors.New("NO_ACTIVATION")
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	case "8":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") {
			return nil
		} else if strings.HasPrefix(responseData, "BAD_STATUS") {
			return errors.New("BAD_STATUS")
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	case "7":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") {
			return nil
		} else if strings.HasPrefix(responseData, "BAD_STATUS") {
			return errors.New("BAD_STATUS")
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	case "9":
		if strings.HasPrefix(responseData, "success") {
			return nil
		}
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	case "10":
		if strings.HasPrefix(responseData, "ACCESS_CANCEL") {
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
		return errors.New(fmt.Sprintf("NUMBER_REQUEST_FAILED_FOR_THIRD_PARTY_SERVER_%s", server))
	default:
		return errors.New("INVALID_SERVER_VALUE")
	}
}

func getServerDataWithMaintenanceCheck(db *mongo.Database, server string) (models.Server, error) {
	serverNumber, _ := strconv.Atoi(server)
	var serverData models.Server
	collection := models.InitializeServerCollection(db)
	sererCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := collection.FindOne(sererCtx, bson.M{"server": serverNumber}).Decode(&serverData)
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
		if err != nil && err.Error() == "ACCESS_CANCEL" {
			logs.Logger.Error(err)
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
