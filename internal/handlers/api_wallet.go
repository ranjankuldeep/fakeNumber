package handlers

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
func APIKeyHandler(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	// serverCol := models.InitializeServerCollection(db)
	walletCol := models.InitializeApiWalletuserCollection(db)

	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "userId is required"})
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
	objID, _ := primitive.ObjectIDFromHex(userId)
	err := walletCol.FindOne(ctx, bson.M{"userId": objID}).Decode(&user)
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
	file := c.FormValue("file")
	if file == "" {
		return c.String(http.StatusBadRequest, "QR code file is required")
	}

	base64Data := file[strings.IndexByte(file, ',')+1:]
	bufferData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid file format"})
	}

	filePath := filepath.Join("uploads", "upi-qr-code.png")
	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)

	err = ioutil.WriteFile(filePath, bufferData, 0644)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save QR code"})
	}

	return c.String(http.StatusOK, "QR code updated successfully")
}

// Handler to create or update API key for recharge type
func CreateOrUpdateAPIKeyHandler(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	rechargeCol := models.InitializeRechargeAPICollection(db)

	apiKey := c.FormValue("api_key")
	rechargeType := c.FormValue("recharge_type")
	if rechargeType == "" || apiKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "API key and recharge_type are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingAPI models.RechargeAPI
	err := rechargeCol.FindOne(ctx, bson.M{"recharge_type": rechargeType}).Decode(&existingAPI)
	if err == mongo.ErrNoDocuments {
		// Create new API key
		_, err = rechargeCol.InsertOne(ctx, models.RechargeAPI{
			RechargeType: rechargeType,
			APIKey:       apiKey,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create API key"})
		}
		return c.JSON(http.StatusCreated, echo.Map{"message": "API key created successfully"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Update existing API key
	_, err = rechargeCol.UpdateOne(ctx, bson.M{"recharge_type": rechargeType}, bson.M{"$set": bson.M{"api_key": apiKey}})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update API key"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "API key updated successfully"})
}
