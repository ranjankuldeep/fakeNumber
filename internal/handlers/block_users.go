package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// Handler to toggle the status of a block type
func ToggleBlockStatus(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	blockCol := models.InitializeBlockCollection(db)

	var request struct {
		BlockType string `json:"blockType"`
		Status    bool   `json:"status"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"status": request.Status}}
	result := blockCol.FindOneAndUpdate(ctx, bson.M{"block_type": request.BlockType}, update, options.FindOneAndUpdate().SetReturnDocument(options.After))

	var updatedRecord models.Block
	err := result.Decode(&updatedRecord)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "Block type not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error updating block status"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Block status updated successfully", "data": updatedRecord})
}

// Handler to retrieve the status of a block type
func GetBlockStatus(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	blockCol := models.InitializeBlockCollection(db)

	blockType := c.QueryParam("blockType")
	if blockType == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "blockType is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var record models.Block
	err := blockCol.FindOne(ctx, bson.M{"block_type": blockType}).Decode(&record)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "Block type not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error fetching block status"})
	}

	return c.JSON(http.StatusOK, echo.Map{"status": record.Status})
}

// // Handler to clear fraudulent user data
// func BlockFraudClear(c echo.Context) error {
// 	db := c.Get("db").(*mongo.Database)
// 	walletCol := models.InitializeApiWalletuserCollection(db)
// 	rechargeHistoryCol := models.InitializeRechargeHistoryCollection(db)
// 	transactionHistoryCol := models.InitializeTransactionHistoryCollection(db)

// 	userId := c.QueryParam("userId")
// 	if userId == "" {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"message": "User ID is required"})
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	objID, err := primitive.ObjectIDFromHex(userId)
// 	if err != nil {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid User ID"})
// 	}

// 	var user models.ApiWalletUser
// 	err = walletCol.FindOne(ctx, bson.M{"userId": objID}).Decode(&user)
// 	if err == mongo.ErrNoDocuments {
// 		return c.JSON(http.StatusNotFound, echo.Map{"message": "User not found"})
// 	} else if err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error finding user"})
// 	}

// 	// Delete recharge history records for the user
// 	_, err = rechargeHistoryCol.DeleteMany(ctx, bson.M{"userId": objID})
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error clearing recharge history"})
// 	}

// 	// Delete transaction history records for the user
// 	_, err = transactionHistoryCol.DeleteMany(ctx, bson.M{"userId": objID})
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error clearing transaction history"})
// 	}

// 	// Set user's balance to 0
// 	_, err = walletCol.UpdateOne(ctx, bson.M{"userId": objID}, bson.M{"$set": bson.M{"balance": 0}})
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error updating user balance"})
// 	}

// 	return c.JSON(http.StatusOK, echo.Map{"message": "User data cleared successfully"})
// }
