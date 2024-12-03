package handlers

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
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
)

func GetNumberHandlerApi(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	apiKey := c.QueryParam("apikey")
	server := c.QueryParam("server")
	code := c.QueryParam("code")
	otp := c.QueryParam("otp")

	ctx := context.TODO()

	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty api key"})
	}
	if server == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty server value"})
	}
	if otp == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty otp value"})
	}
	if code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty code value"})
	}
	serverNumber, _ := strconv.Atoi(server)

	// Maintenance check
	serverCollection := models.InitializeServerCollection(db)
	var server0 models.Server
	err := serverCollection.FindOne(ctx, bson.M{"server": 0}).Decode(server0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if server0.Maintenance == true {
		return c.JSON(http.StatusOK, map[string]string{"error": "site is under maintenance"})
	}

	var apiWalletUser models.ApiWalletUser
	apiWalletCollection := models.InitializeApiWalletuserCollection(db)
	err = apiWalletCollection.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid api key"})
		}
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	userCollection := models.InitializeUserCollection(db)
	var user models.User
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&user)
	if user.Blocked {
		return c.JSON(http.StatusForbidden, echo.Map{"error": "your account is blocked, contact the admin"})
	}

	// Fetch server information
	var serverInfo models.Server
	err = serverCollection.FindOne(ctx, bson.M{"server": serverNumber}).Decode(&serverInfo)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "server not found"})
	}

	if serverInfo.Maintenance {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "server under maintenance"})
	}

	serverListCollection := models.InitializeServerListCollection(db)
	var serviceList models.ServerList
	err = serverListCollection.FindOne(ctx, bson.M{
		"servers.server": serverNumber,
		"servers.code":   code,
	}).Decode(&serviceList)
	if err != nil {
		logs.Logger.Error("service not found for given server and code")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "service not found"})
	}

	// Identify the specific service based on server and code
	var serviceName string
	for _, s := range serviceList.Servers {
		if s.Server == serverNumber && s.Code == code {
			serviceName = serviceList.Name
			break
		}
	}

	isMultiple := "false"
	if otp == "single" {
		isMultiple = "true"
	}

	if serviceName == "" {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "service name could not be resolved"})
	}

	// Process server data
	var serverData models.ServerData
	for _, s := range serviceList.Servers {
		if s.Server == serverNumber {
			serverData = models.ServerData{
				Price:  s.Price,
				Code:   s.Code,
				Otp:    s.Otp,
				Server: serverNumber,
			}
			break
		}
	}

	price, _ := strconv.ParseFloat(serverData.Price, 64)
	discount, err := FetchDiscount(ctx, db, user.ID.Hex(), serviceName, serverNumber)
	price += discount

	if apiWalletUser.Balance < price {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "low balance"})
	}

	apiURLRequest, err := constructApiUrl(db, server, serverInfo.APIKey, serverInfo.Token, serverData, isMultiple)
	if err != nil {
		logs.Logger.Error("failed to construct API URL")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}
	numData, err := ExtractNumber(server, apiURLRequest)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	newBalance := math.Round((apiWalletUser.Balance-price)*100) / 100
	_, err = apiWalletCollection.UpdateOne(ctx, bson.M{"userId": user.ID}, bson.M{"$set": bson.M{"balance": newBalance}})
	if err != nil {
		logs.Logger.Error("failed to update user balance")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

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
	}
	_, err = transactionHistoryCollection.InsertOne(ctx, transaction)
	if err != nil {
		logs.Logger.Error("failed to save transaction history")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
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
		ExpirationTime: time.Now().Add(19 * time.Minute), // Adjust expiration time as needed
	}
	_, err = orderCollection.InsertOne(ctx, order)
	if err != nil {
		logs.Logger.Error("failed to create order")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	if numData.Id == "" || numData.Number == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "no stock available"})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"status": "ok",
		"id":     numData.Id,
		"number": numData.Number,
	})
}

func GetOTPHandlerApi(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	ctx := context.TODO()
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

	// Maintenance check
	serverCollection := models.InitializeServerCollection(db)
	var server0 models.Server
	err := serverCollection.FindOne(ctx, bson.M{"server": 0}).Decode(server0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if server0.Maintenance == true {
		return c.JSON(http.StatusOK, map[string]string{"error": "site is under maintenance"})
	}

	var transaction models.TransactionHistory
	transactionCollection := models.InitializeTransactionHistoryCollection(db)
	err = transactionCollection.FindOne(context.TODO(), bson.M{"id": id}).Decode(&transaction)
	if err != nil {
		logs.Logger.Info("sdf")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}
	serviceName := transaction.Service

	if transaction.Status == "CANCELLED" {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "otp": "number cancelled"})
	}
	if len(transaction.OTP) == 0 && transaction.Status == "PENDING" {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "otp": "waiting for otp"})
	}

	var apiWalletUser models.ApiWalletUser
	apiWalletCollection := models.InitializeApiWalletuserCollection(db)
	err = apiWalletCollection.FindOne(context.TODO(), bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil {
		if err == mongo.ErrEmptySlice {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "invalid api key"})
		}
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	userCollection := models.InitializeUserCollection(db)
	var userData models.User
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&userData)
	if userData.Blocked {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "your account is blocked, contact the admin"})
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

	validOtpList, err := fetchOTP(server, id, constructedOTPRequest)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	for _, validOtp := range validOtpList {
		transactionCollection := models.InitializeTransactionHistoryCollection(db)
		filter := bson.M{"id": id, "otp": validOtp}
		var existingEntry models.TransactionHistory
		err = transactionCollection.FindOne(ctx, filter).Decode(&existingEntry)
		if err == mongo.ErrNoDocuments {
			formattedDateTime := FormatDateTime()
			update := bson.M{
				"$addToSet": bson.M{"otp": validOtp},
				"$set":      bson.M{"date_time": formattedDateTime},
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
				ServiceName: existingEntry.Service,
				Price:       existingEntry.Price,
				Server:      existingEntry.Server,
				Number:      existingEntry.Number,
				OTP:         validOtp,
				Ip:          ipDetail,
			}

			err = services.OtpGetDetails(otpDetail)
			if err != nil {
				logs.Logger.Error(err)
			}

			go func(otp string) {
				err := triggerNextOtp(db, server, serviceName, id)
				if err != nil {
					log.Printf("Error triggering next OTP for ID: %s, OTP: %s - %v", id, otp, err)
				} else {
					log.Printf("Successfully triggered next OTP for ID: %s, OTP: %s", id, otp)
				}
			}(validOtp)
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok", "otp": transaction.OTP})
}

func CancelNumberHandlerApi(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	ctx := context.TODO()
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

	// Maintenance check
	serverCollection := models.InitializeServerCollection(db)
	var server0 models.Server
	err := serverCollection.FindOne(ctx, bson.M{"server": 0}).Decode(server0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if server0.Maintenance == true {
		return c.JSON(http.StatusOK, map[string]string{"error": "site is under maintenance"})
	}

	transactionCollection := models.InitializeTransactionHistoryCollection(db)
	var existingOrder models.Order
	orderCollection := models.InitializeOrderCollection(db)
	err = orderCollection.FindOne(ctx, bson.M{"numberId": id}).Decode(&existingOrder)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"errror": "internal server error"})
	}

	var apiWalletUser models.ApiWalletUser
	apiWalletCollection := models.InitializeApiWalletuserCollection(db)
	err = apiWalletCollection.FindOne(context.TODO(), bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil {
		if err == mongo.ErrEmptySlice {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "invalid api key"})
		}
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	userCollection := models.InitializeUserCollection(db)
	var userData models.User
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&userData)
	if userData.Blocked {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "your account is blocked, contact the admin"})
	}

	serverData, err := getServerDataWithMaintenanceCheck(ctx, db, server)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if serverData.Maintenance == true {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "under maintenance"})
	}

	timeDifference := time.Now().Sub(existingOrder.OrderTime)
	if timeDifference < 2*time.Minute {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "wait 2 mints before cancel"})
	}

	var transactionData models.TransactionHistory
	filter := bson.M{
		"userId": apiWalletUser.UserID.Hex(),
		"id":     id,
	}
	err = transactionCollection.FindOne(ctx, filter).Decode(&transactionData)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	otpArrived := false
	if len(transactionData.OTP) != 0 {
		otpArrived = true
	}
	if otpArrived == true {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "otp already come"})
	}

	constructedNumberRequest, err := ConstructNumberUrl(server, serverData.APIKey, serverData.Token, id, existingOrder.Number)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid server"})
	}

	err = CancelNumberThirdParty(constructedNumberRequest.URL, server, id, db, constructedNumberRequest.Headers)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	_, err = orderCollection.DeleteOne(ctx, bson.M{"numberId": id})
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	formattedData := FormatDateTime()

	var transaction models.TransactionHistory
	err = transactionCollection.FindOne(ctx, bson.M{"id": id}).Decode(&transaction)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	transactionUpdateFilter := bson.M{"id": id}
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
	price, err := strconv.ParseFloat(transaction.Price, 64)
	newBalance := apiWalletUser.Balance + price
	newBalance = math.Round(newBalance*100) / 100

	update := bson.M{
		"$set": bson.M{"balance": newBalance},
	}
	balanceFilter := bson.M{"_id": apiWalletUser.UserID}

	_, err = apiWalletCollection.UpdateOne(ctx, balanceFilter, update)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// ipDetails, err := utils.GetIpDetails(c)
	// if err != nil {
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	// }
	// services.NumberCancelDetails(userData.Email, transaction.Service, price, server, int64(price), apiWalletUser.Balance, ipDetails)
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func GetServiceDataApi(c echo.Context) error {
	apiKey := c.QueryParam("api_key")
	if apiKey == "" {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	db := c.Get("db").(*mongo.Database)
	serverCollection := models.InitializeServerCollection(db)
	serviceCollection := models.InitializeServerListCollection(db)
	serviceDiscountCollection := models.InitializeServiceDiscountCollection(db)
	serverDiscountCollection := models.InitializeServerDiscountCollection(db)
	userDiscountCollection := models.InitializeUserDiscountCollection(db)
	var maintenanceStatus struct {
		Maintenance bool `bson:"maintainance"`
	}
	// Maintenance check
	var server0 models.Server
	err := serverCollection.FindOne(context.TODO(), bson.M{"server": 0}).Decode(server0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if server0.Maintenance == true {
		return c.JSON(http.StatusOK, map[string]string{"error": "site is under maintenance"})
	}

	var apiWalletUser models.ApiWalletUser
	apiWalletCollection := models.InitializeApiWalletuserCollection(db)
	err = apiWalletCollection.FindOne(context.TODO(), bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "User not found"})
	}

	err = serverCollection.FindOne(context.Background(), bson.M{"server": 0}).Decode(&maintenanceStatus)
	if err == nil && maintenanceStatus.Maintenance {
		log.Println(err)
		return c.JSON(http.StatusForbidden, echo.Map{"error": "Site is under maintenance."})
	}
	serversInMaintenance, err := serverCollection.Find(context.Background(), bson.M{"maintainance": true})
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	defer serversInMaintenance.Close(context.Background())

	// Track maintenance servers
	var maintenanceServerNumbers []int
	for serversInMaintenance.Next(context.Background()) {
		var server struct {
			ServerNumber int `bson:"server"`
		}
		if err := serversInMaintenance.Decode(&server); err == nil {
			maintenanceServerNumbers = append(maintenanceServerNumbers, server.ServerNumber)
		}
	}

	cursor, err := serviceCollection.Find(context.Background(), bson.D{})
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	defer cursor.Close(context.Background())

	var services []models.ServerList
	for cursor.Next(context.Background()) {
		var service models.ServerList
		if err := cursor.Decode(&service); err != nil {
			logs.Logger.Error(err)
		}
		services = append(services, service)
	}

	serviceDiscounts, serverDiscounts, userDiscounts, err := loadDiscounts(serviceDiscountCollection, serverDiscountCollection, userDiscountCollection, apiWalletUser.UserID.Hex())
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Deduplication logic
	filteredData := []ServiceResponse{}
	uniqueServices := make(map[string]bool)

	for _, service := range services {
		if uniqueServices[service.Name] {
			continue // Skip duplicate services
		}
		uniqueServices[service.Name] = true

		serverDetails := []ServerDetail{}
		seenServers := make(map[int]bool)

		for _, server := range service.Servers {
			if contains(maintenanceServerNumbers, server.Server) || seenServers[server.Server] {
				continue // Skip maintenance or duplicate servers
			}
			seenServers[server.Server] = true

			// Calculate discounts
			discount := CalculateDiscount(serviceDiscounts, serverDiscounts, userDiscounts, service.Name, server.Server, apiWalletUser.UserID.Hex())
			price, _ := strconv.ParseFloat(server.Price, 64)
			adjustedPrice := strconv.FormatFloat(price+discount, 'f', 2, 64)

			// Normalize the OTP field
			var otpType string
			switch server.Otp {
			case "Multiple Otp":
				otpType = "multiple"
			case "Single Otp & Fresh Number", "Single Otp":
				otpType = "single"
			default:
				otpType = "unknown"
			}

			serverDetails = append(serverDetails, ServerDetail{
				Server: strconv.Itoa(server.Server),
				Price:  adjustedPrice,
				Code:   server.Code,
				Otp:    otpType,
			})
		}

		sort.Slice(serverDetails, func(i, j int) bool {
			return serverDetails[i].Server < serverDetails[j].Server
		})
		filteredData = append(filteredData, ServiceResponse{
			Name:    service.Name,
			Servers: serverDetails,
		})
	}

	sort.Slice(filteredData, func(i, j int) bool {
		return filteredData[i].Name < filteredData[j].Name
	})
	return c.JSON(http.StatusOK, filteredData)
}
