package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var predefinedBlockTypes = []models.Block{
	{BlockType: "Number_Cancel", Status: false},
	{BlockType: "User_Fraud", Status: false},
}

// Handler to save predefined block types to the database
func SavePredefinedBlockTypes(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	blockCol := models.InitializeBlockCollection(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, blockType := range predefinedBlockTypes {
		var existingRecord models.Block
		err := blockCol.FindOne(ctx, bson.M{"block_type": blockType.BlockType}).Decode(&existingRecord)
		if err == mongo.ErrNoDocuments {
			_, err := blockCol.InsertOne(ctx, blockType)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error saving block types"})
			}
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
		}
	}

	return c.JSON(http.StatusCreated, echo.Map{"message": "Block types saved successfully"})
}

func ToggleBlockStatus(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	blockCol := models.InitializeBlockToggler(db)

	var request struct {
		Status bool `json:"status"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "block", Value: request.Status},
			{Key: "updatedAt", Value: time.Now()}, // Update the timestamp
		}},
	}
	result, err := blockCol.UpdateOne(c.Request().Context(), bson.M{}, update)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update block status"})
	}
	if result.MatchedCount == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Block status document not found"})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Block status updated successfully",
		"data": bson.M{
			"block":     request.Status,
			"updatedAt": time.Now(),
		},
	})
}

func GetBlockStatus(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	blockStatus, err := FetchBlockStatus(context.TODO(), db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Unable to Fetch Block Status"})
	}
	return c.JSON(http.StatusOK, echo.Map{"status": blockStatus})
}

func FetchBlockStatus(ctx context.Context, db *mongo.Database) (bool, error) {
	blockTogglerCollection := models.InitializeBlockToggler(db)
	var blockStatus models.ToggleBlock
	err := blockTogglerCollection.FindOne(ctx, bson.M{}).Decode(&blockStatus)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, fmt.Errorf("block status not found in the collection")
		}
		return false, fmt.Errorf("failed to fetch block status: %w", err)
	}
	return blockStatus.Block, nil
}

// Handler to clear fraudulent user data
func BlockFraudClear(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	walletCol := models.InitializeApiWalletuserCollection(db)
	rechargeHistoryCol := models.InitializeRechargeHistoryCollection(db)
	transactionHistoryCol := models.InitializeTransactionHistoryCollection(db)

	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "User ID is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid User ID"})
	}

	var user models.ApiWalletUser
	err = walletCol.FindOne(ctx, bson.M{"userId": objID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "User not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error finding user"})
	}

	// Delete recharge history records for the user
	_, err = rechargeHistoryCol.DeleteMany(ctx, bson.M{"userId": userId})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error clearing recharge history"})
	}

	// Delete transaction history records for the user
	_, err = transactionHistoryCol.DeleteMany(ctx, bson.M{"userId": userId})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error clearing transaction history"})
	}

	// Delete the user from the wallet collection
	_, err = walletCol.DeleteOne(ctx, bson.M{"userId": objID})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error deleting user from the collection"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "User data cleared successfully"})
}
