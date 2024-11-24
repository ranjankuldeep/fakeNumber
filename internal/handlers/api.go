package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetNumberHandlerApi(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	apiKey := c.QueryParam("apikey")
	server := c.QueryParam("server")
	serviceNameWithSpaces := c.QueryParam("servicename")
	serviceName := strings.ReplaceAll(serviceNameWithSpaces, "%", " ")
	ctx := context.TODO()

	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty key"})
	}
	if server == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty server number"})
	}
	serverNumber, _ := strconv.Atoi(server)
	if serviceName == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empty service name"})
	}

	var apiWalletUser models.ApiWalletUser
	apiWalletCollection := models.InitializeApiWalletuserCollection(db)
	err := apiWalletCollection.FindOne(context.TODO(), bson.M{"api_key": apiKey}).Decode(&apiWalletUser)
	if err != nil {
		if err == mongo.ErrEmptySlice {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "invalid api key"})
		}
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	userCollection := models.InitializeUserCollection(db)
	var user models.User
	err = userCollection.FindOne(ctx, bson.M{"_id": apiWalletUser.UserID}).Decode(&user)
	if user.Blocked {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "your account is blocked, contact the admin"})
	}

	var serverInfo models.Server
	serverCollection := models.InitializeServerCollection(db)
	err = serverCollection.FindOne(ctx, bson.M{"server": serverNumber}).Decode(&serverInfo)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	if serverInfo.Maintenance == true {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "under maintenance"})
	}

	serverListollection := models.InitializeServerListCollection(db)
	var serverList models.ServerList
	err = serverListollection.FindOne(ctx, bson.M{
		"name":           serviceName,
		"servers.server": serverNumber,
	}).Decode(&serverList)
	if err != nil {
		logs.Logger.Error("couldn't find server list")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

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

	isMultiple := "false"
	apiURLRequest, err := constructApiUrl(server, serverInfo.APIKey, serverInfo.Token, serverData, isMultiple)
	if err != nil {
		logs.Logger.Error("Couldn't construcrt api url")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	logs.Logger.Info(fmt.Sprintf("url-%s", apiURLRequest.URL))
	numData, err := ExtractNumber(server, apiURLRequest)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	logs.Logger.Info(fmt.Sprintf("id-%s number-%s", numData.Id, numData.Number))

	price, _ := strconv.ParseFloat(serverData.Price, 64)
	discount, err := FetchDiscount(ctx, db, user.ID.Hex(), serviceName, serverNumber)
	price += discount

	// Check user balance
	if apiWalletUser.Balance < price {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "low balance"})
	}

	newBalance := apiWalletUser.Balance - price
	roundedBalance := math.Round(newBalance*100) / 100
	_, err = apiWalletCollection.UpdateOne(ctx, bson.M{"userId": user.ID}, bson.M{"$set": bson.M{"balance": roundedBalance}})
	if err != nil {
		logs.Logger.Error("FAILED_TO_UPDATE_USER_BALANCE")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
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
		logs.Logger.Error("error saving transaction history")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
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
		logs.Logger.Error("failed to create order")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server errror"})
	}

	go func(id, number, userId string, db *mongo.Database, ctx context.Context) {
		defer func() {
			if r := recover(); r != nil {
				logs.Logger.Error("Recovered from panic in OTP handling goroutine:", r)
			}
		}()

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

		constructedNumberRequest, err := constructNumberUrl(server, serverData.APIKey, serverData.Token, id, number)
		if err != nil {
			logs.Logger.Error(err)
			return
		}

		err = CancelNumberThirdParty(constructedNumberRequest.URL, server, id, db, constructedNumberRequest.Headers)
		if err != nil {
			logs.Logger.Error(err)
			return
		}
	}(numData.Id, numData.Number, apiWalletUser.UserID.Hex(), db, ctx)

	if numData.Id == "" || numData.Number == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no stock"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "id": numData.Id, "number": numData.Number})
}
