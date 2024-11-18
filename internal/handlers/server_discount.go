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
	db := c.Get("db").(*mongo.Database)
	server, err := strconv.Atoi(c.FormValue("server"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server number is required and must be an integer."})
	}

	discount, err := strconv.ParseFloat(c.FormValue("discount"), 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Discount must be a valid number."})
	}

	serverDiscountCol := db.Collection("server_discount")
	filter := bson.M{"server": server}
	update := bson.M{"$set": bson.M{"server": server, "discount": discount}}
	opts := options.Update().SetUpsert(true)

	_, err = serverDiscountCol.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		log.Println("Error updating discount:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
	}

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
	log.Println("INFO: Fetching discounts from the 'server_discount' collection")
	cursor, err := db.Collection("server_discount").Find(context.Background(), bson.M{})
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

	// Log: Parsing the data
	log.Println("INFO: Decoding fetched discounts")
	if err = cursor.All(context.Background(), &serverDiscounts); err != nil {
		log.Println("ERROR: Error decoding discounts:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error parsing discounts"})
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

	result, err := db.Collection("server_discount").DeleteOne(context.Background(), bson.M{"server": server})
	if err != nil {
		log.Println("Error deleting discount:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	if result.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Server discount not found."})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Server discount deleted successfully"})
}
