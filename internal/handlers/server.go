package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddServer(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	serverCollection := db.Collection("server")

	server, err := strconv.Atoi(c.FormValue("server"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server is required and must be a number."})
	}
	apiKey := c.FormValue("api_key")

	existingServer := models.Server{}
	filter := bson.M{"server": server}
	err = serverCollection.FindOne(context.Background(), filter).Decode(&existingServer)

	if err == nil {
		if apiKey != "" {
			update := bson.M{"$set": bson.M{"api_key": apiKey}}
			_, err := serverCollection.UpdateOne(context.Background(), filter, update)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
			}
			return c.JSON(http.StatusOK, map[string]string{"message": "API key updated successfully."})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Server already exists. No changes made."})
	} else if err == mongo.ErrNoDocuments {
		newServer := models.Server{ServerNumber: server, APIKey: apiKey}
		_, err := serverCollection.InsertOne(context.Background(), newServer)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		}
		return c.JSON(http.StatusCreated, map[string]string{"message": "Server added successfully."})
	}
	log.Println("Error:", err)
	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
}

// Get all servers
func GetServer(c echo.Context) error {
	fmt.Println("GetServer")
	db := c.Get("db").(*mongo.Database)
	serverCollection := db.Collection("servers")

	var servers []models.Server
	cursor, err := serverCollection.Find(context.Background(), bson.M{}, options.Find().SetSort(bson.M{"server": 1}))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	defer cursor.Close(context.Background())
	if err := cursor.All(context.Background(), &servers); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	return c.JSON(http.StatusOK, servers)
}

// Get maintenance status for server 0
func GetServerZero(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	serverCollection := db.Collection("server")
	var server models.Server
	err := serverCollection.FindOne(context.Background(), bson.M{"server": 0}).Decode(&server)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"maintainance": server.Maintenance})
}

// Delete a server
func DeleteServer(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	serverCollection := db.Collection("server")

	server, err := strconv.Atoi(c.QueryParam("server"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server number is required and must be an integer."})
	}

	result, err := serverCollection.DeleteOne(context.Background(), bson.M{"server": server})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	if result.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Server not found."})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Server deleted successfully"})
}

// Update maintenance status for a server
func MaintainanceServer(c echo.Context) error {
	// Log the start of the function
	log.Println("INFO: Starting MaintainanceServer handler")

	// Retrieve the database instance
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	log.Println("INFO: Database instance retrieved successfully")

	// Define a struct to parse the JSON input
	type RequestBody struct {
		Server       int  `json:"server"`
		Maintainance bool `json:"maintainance"`
	}

	var input RequestBody

	// Bind the JSON input to the struct
	log.Println("INFO: Binding JSON input from request body")
	if err := c.Bind(&input); err != nil {
		log.Println("ERROR: Failed to bind JSON input:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input format"})
	}

	// Log the received input
	log.Printf("INFO: Received input - server: %d, maintainance: %t\n", input.Server, input.Maintainance)

	// Validate input
	if input.Server == 0 {
		log.Println("ERROR: Server number is required and must be greater than 0")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server is required and must be greater than 0"})
	}

	// Initialize the collection and define the filter and update
	serverCollection := db.Collection("servers")
	filter := bson.M{"server": input.Server}
	update := bson.M{"$set": bson.M{"maintainance": input.Maintainance}}

	// Log the update operation
	log.Printf("INFO: Updating document in 'server' collection with filter: %+v and update: %+v\n", filter, update)
	result, err := serverCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Println("ERROR: Failed to update maintenance status:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	// Check if the update operation matched any document
	if result.MatchedCount == 0 {
		log.Printf("INFO: No matching document found for filter: %+v\n", filter)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Server not found"})
	}

	// Log the successful update
	log.Printf("INFO: Maintenance status updated successfully for server: %d\n", input.Server)
	return c.JSON(http.StatusOK, map[string]string{"message": "Maintenance status updated successfully"})
}

// Add token for server 9
func AddTokenForServer9(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	serverCollection := db.Collection("server")

	token := c.FormValue("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Token is required"})
	}

	filter := bson.M{"server": 9}
	update := bson.M{"$set": bson.M{"token": token}}
	result, err := serverCollection.UpdateOne(context.Background(), filter, update)
	if err != nil || result.MatchedCount == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Server 9 not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Token added successfully"})
}

// Get token for server 9
func GetTokenForServer9(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	serverCollection := db.Collection("server")

	var server models.Server
	err := serverCollection.FindOne(context.Background(), bson.M{"server": 9}).Decode(&server)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Server 9 not found"})
	}
	return c.JSON(http.StatusOK, map[string]string{"token": server.Token})
}

// Update exchange rate and margin for a server
func UpdateExchangeRateAndMargin(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	serverCollection := db.Collection("server")

	server, err := strconv.Atoi(c.FormValue("server"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server is required and must be an integer."})
	}

	exchangeRate, err := strconv.ParseFloat(c.FormValue("exchangeRate"), 64)
	if err != nil && c.FormValue("exchangeRate") != "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Exchange rate must be a valid number"})
	}

	margin, err := strconv.ParseFloat(c.FormValue("margin"), 64)
	if err != nil && c.FormValue("margin") != "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Margin must be a valid number"})
	}

	filter := bson.M{"server": server}
	update := bson.M{"$set": bson.M{"exchangeRate": exchangeRate, "margin": margin}}
	result, err := serverCollection.UpdateOne(context.Background(), filter, update)
	if err != nil || result.MatchedCount == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Server not found"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Exchange rate and/or margin updated successfully.",
		"exchangeRate": exchangeRate,
		"margin":       margin,
	})
}
