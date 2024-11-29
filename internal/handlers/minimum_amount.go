package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddMinimumRecharge(c echo.Context) error {
	var req struct {
		MinimumRecharge float64 `json:"minimumRecharge"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input format"})
	}

	if req.MinimumRecharge < 1 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Minimum recharge must be at least 1"})
	}

	db := c.Get("db").(*mongo.Database)
	collection := models.InitializeMinimumCollection(db)

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

func GetMinimumRecharge(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	collection := models.InitializeMinimumCollection(db)

	var result models.MinimumRecharge
	err := collection.FindOne(context.Background(), bson.M{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusOK, echo.Map{
				"minimumRecharge": nil,
				"message":         "No minimum recharge amount set.",
			})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch minimum recharge"})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"minimumRecharge": result.MinimumRecharge,
		"message":         "Minimum recharge amount fetched successfully",
	})
}

func DeleteMinimumRecharge(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	collection := models.InitializeMinimumCollection(db)

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
