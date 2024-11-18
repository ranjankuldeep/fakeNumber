// AddUserDiscount adds or updates a user discount
package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddUserDiscount(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userDiscountCollection := db.Collection("userDiscount")
	userCollection := models.InitializeUserCollection(db)

	var req struct {
		Email    string  `json:"email" validate:"required,email"`
		Service  string  `json:"service" validate:"required"`
		Server   int     `json:"server" validate:"required"`
		Discount float64 `json:"discount" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	var user models.User
	if err := userCollection.FindOne(context.Background(), bson.M{"email": req.Email}).Decode(&user); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	filter := bson.M{"userId": user.ID, "service": req.Service, "server": req.Server}
	update := bson.M{"$set": bson.M{"discount": req.Discount}}

	upsertOpts := options.Update().SetUpsert(true)
	_, err := userDiscountCollection.UpdateOne(context.Background(), filter, update, upsertOpts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error updating discount"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Discount added/updated successfully"})
}

// GetUserDiscount retrieves all discounts for a specific user
func GetUserDiscount(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userDiscountCollection := db.Collection("userDiscount")

	userID := c.QueryParam("userId")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}

	objectId, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid User ID"})
	}

	var userDiscounts []models.UserDiscount
	cursor, err := userDiscountCollection.Find(context.Background(), bson.M{"userId": objectId})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error fetching discounts"})
	}
	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		var discount models.UserDiscount
		if err := cursor.Decode(&discount); err != nil {
			log.Println("Error decoding discount:", err)
		} else {
			userDiscounts = append(userDiscounts, discount)
		}
	}
	return c.JSON(http.StatusOK, userDiscounts)
}

// DeleteUserDiscount deletes a specific user discount by service and server
func DeleteUserDiscount(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userDiscountCollection := db.Collection("userDiscount")

	userID := c.QueryParam("userId")
	service := c.QueryParam("service")
	serverStr := c.QueryParam("server")

	if userID == "" || service == "" || serverStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID, Service, and Server are required"})
	}

	server, err := strconv.Atoi(serverStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid Server"})
	}

	objectId, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid User ID"})
	}

	result, err := userDiscountCollection.DeleteOne(context.Background(), bson.M{"userId": objectId, "service": service, "server": server})
	if err != nil || result.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User discount not found"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "User discount deleted successfully"})
}

// GetAllUserDiscounts retrieves all user discounts
func GetAllUserDiscounts(c echo.Context) error {
	// Log: Start of the function
	log.Println("INFO: Starting GetAllUserDiscounts handler")

	db := c.Get("db").(*mongo.Database)
	userDiscountCollection := db.Collection("user-discounts")

	log.Println("INFO: Fetching all user discounts from the database...")
	var allDiscounts []models.UserDiscount

	// Find all documents in the userDiscount collection
	cursor, err := userDiscountCollection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Println("ERROR: Error fetching all discounts from the database:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error fetching all discounts"})
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			log.Println("ERROR: Error closing the cursor:", err)
		}
	}()

	// Iterate over the cursor to decode each discount
	for cursor.Next(context.Background()) {
		var discount models.UserDiscount
		if err := cursor.Decode(&discount); err != nil {
			log.Println("ERROR: Error decoding discount document:", err)
		} else {
			allDiscounts = append(allDiscounts, discount)
			log.Printf("INFO: Discount added to the list: %+v\n", discount)
		}
	}

	// Check for any errors during iteration
	if err := cursor.Err(); err != nil {
		log.Println("ERROR: Cursor iteration error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error iterating over discounts"})
	}

	// Log the total number of discounts fetched
	log.Printf("INFO: Successfully fetched %d user discounts\n", len(allDiscounts))

	// Return an empty array if no discounts are found
	if len(allDiscounts) == 0 {
		log.Println("INFO: No user discounts found, returning an empty array.")
		return c.JSON(http.StatusOK, []models.UserDiscount{})
	}

	// Return the result
	return c.JSON(http.StatusOK, allDiscounts)
}
