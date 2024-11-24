package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddDiscount(c echo.Context) error {
	// Log: Start of the function
	log.Println("INFO: Starting AddDiscount handler")

	// Retrieve the database instance
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	log.Println("INFO: Database instance retrieved successfully")

	// Define a struct to parse the JSON input
	type RequestBody struct {
		Server   string  `json:"server"`
		Discount float64 `json:"discount"`
	}

	var input RequestBody

	// Bind the JSON input to the struct
	if err := c.Bind(&input); err != nil {
		log.Println("ERROR: Failed to bind JSON input:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input format"})
	}

	// Convert `server` to an integer
	server, err := strconv.Atoi(input.Server)
	if err != nil {
		log.Println("ERROR: Server number must be an integer:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server number must be an integer"})
	}

	// Log received parameters
	log.Printf("INFO: Received parameters - server: %d, discount: %.2f\n", server, input.Discount)

	// Validate input
	if server <= 0 {
		log.Println("ERROR: Server number is required and must be greater than 0")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server number is required and must be greater than 0"})
	}

	// Initialize collection and set up filter, update, and options
	serverDiscountCol := db.Collection("server-discounts")
	filter := bson.M{"server": server}
	update := bson.M{"$set": bson.M{"server": server, "discount": input.Discount}}
	opts := options.Update().SetUpsert(true)

	// Perform the update operation
	_, err = serverDiscountCol.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		log.Println("ERROR: Failed to add or update discount:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
	}

	// Log success
	log.Println("INFO: Discount added or updated successfully")
	return c.JSON(http.StatusOK, map[string]string{"message": "Discount added or updated successfully"})
}

// Handler to get all server discounts
func GetDiscount(c echo.Context) error {
	// Log: Start of the function
	log.Println("INFO: Starting GetDiscount handler")

	// Fetch the database instance
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	log.Println("INFO: Database instance retrieved successfully")

	// Initialize a slice to store server discounts
	var serverDiscounts []models.ServerDiscount

	// Log: Querying the database
	log.Println("INFO: Fetching discounts from the 'server-discount' collection")
	cursor, err := db.Collection("server-discounts").Find(context.Background(), bson.M{})
	if err != nil {
		log.Println("ERROR: Error fetching discounts from database:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error fetching discounts"})
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			log.Println("ERROR: Error closing cursor:", err)
		} else {
			log.Println("INFO: Cursor closed successfully")
		}
	}()

	// Log: Decoding fetched data
	log.Println("INFO: Decoding fetched discounts")
	if err = cursor.All(context.Background(), &serverDiscounts); err != nil {
		log.Println("ERROR: Error decoding discounts:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error parsing discounts"})
	}

	// Handle case when no discounts are found
	if len(serverDiscounts) == 0 {
		log.Println("INFO: No discounts found in the 'server_discount' collection")
		return c.JSON(http.StatusOK, []models.ServerDiscount{})
	}

	// Log: Successfully fetched discounts
	log.Printf("INFO: Successfully fetched %d discounts\n", len(serverDiscounts))

	// Return the discounts
	log.Println("INFO: Returning fetched discounts")
	return c.JSON(http.StatusOK, serverDiscounts)
}

// Handler to delete a server discount
func DeleteDiscount(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	serverStr := c.QueryParam("server")
	if serverStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server number is required."})
	}

	server, err := strconv.Atoi(serverStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server number must be an integer."})
	}

	result, err := db.Collection("server-discounts").DeleteOne(context.Background(), bson.M{"server": server})
	if err != nil {
		log.Println("Error deleting discount:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	if result.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Server discount not found."})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Server discount deleted successfully"})
}
