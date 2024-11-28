package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MinimumRecharge represents the schema for the minimum recharge document.
type MinimumRecharge struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	MinimumRecharge float64            `bson:"minimumRecharge"`
	CreatedAt       time.Time          `bson:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt"`
}

// AddMinimumRecharge handles setting a new minimum recharge amount.
func AddMinimumRecharge(c echo.Context) error {
	// Parse the request body using json.NewDecoder
	var req struct {
		MinimumRecharge float64 `json:"minimumRecharge"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input format"})
	}

	// Validate the minimumRecharge field
	if req.MinimumRecharge < 1 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Minimum recharge must be at least 1"})
	}

	// Connect to MongoDB
	db := c.Get("db").(*mongo.Database)
	collection := db.Collection("minimum_recharge")

	// Upsert the minimum recharge amount (overwrite if it exists)
	filter := bson.M{}
	update := bson.M{
		"$set": bson.M{
			"minimumRecharge": req.MinimumRecharge,
			"updatedAt":       time.Now(),
		},
		"$setOnInsert": bson.M{
			"createdAt": time.Now(),
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to set minimum recharge"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Minimum recharge amount set successfully"})
}

// GetMinimumRecharge handles fetching the minimum recharge amount.
func GetMinimumRecharge(c echo.Context) error {
	// Connect to MongoDB
	db := c.Get("db").(*mongo.Database)
	collection := db.Collection("minimum_recharge")

	// Find the minimum recharge document
	var result MinimumRecharge
	err := collection.FindOne(context.Background(), bson.M{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return a default value if no document exists
			return c.JSON(http.StatusOK, echo.Map{
				"minimumRecharge": nil,
				"message":         "No minimum recharge amount set.",
			})
		}
		// Handle other errors
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch minimum recharge"})
	}

	// Return the minimum recharge amount
	return c.JSON(http.StatusOK, echo.Map{
		"minimumRecharge": result.MinimumRecharge,
		"message":         "Minimum recharge amount fetched successfully",
	})
}

// DeleteMinimumRecharge handles deleting the minimum recharge amount.
func DeleteMinimumRecharge(c echo.Context) error {
	// Connect to MongoDB
	db := c.Get("db").(*mongo.Database)
	collection := db.Collection("minimum_recharge")

	// Delete the minimum recharge document
	_, err := collection.DeleteOne(context.Background(), bson.M{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to delete minimum recharge",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "Minimum recharge amount deleted successfully",
	})
}
