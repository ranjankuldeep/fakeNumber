package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddServer(c echo.Context) error {
	// Retrieve the database instance
	db := c.Get("db").(*mongo.Database)
	serverCollection := db.Collection("servers")

	// Define a struct to map the incoming JSON payload
	type RequestBody struct {
		Server string `json:"server"`
		APIKey string `json:"api_key"`
	}

	var input RequestBody

	// Bind the JSON payload to the struct
	if err := c.Bind(&input); err != nil {
		log.Println("ERROR: Failed to parse request body:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// Parse the server field into an integer
	server, err := strconv.Atoi(input.Server)
	if err != nil {
		log.Println("ERROR: Server must be a valid number:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server must be a valid number"})
	}

	// Validate API key
	if input.APIKey == "" {
		log.Println("ERROR: Missing API key in the request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "API key is required"})
	}

	log.Printf("INFO: Received request to add/update server. Server: %d, API Key: %s\n", server, input.APIKey)

	// Check if the server already exists
	filter := bson.M{"server": server}
	existingServer := models.Server{}
	err = serverCollection.FindOne(context.Background(), filter).Decode(&existingServer)
	if err == nil {
		// Server exists, update the API key if provided
		update := bson.M{"$set": bson.M{"api_key": input.APIKey}}
		_, err := serverCollection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Println("ERROR: Failed to update API key:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update API key"})
		}
		log.Printf("INFO: API key updated successfully for server %d\n", server)
		return c.JSON(http.StatusOK, map[string]string{"message": "API key updated successfully"})
	} else if err == mongo.ErrNoDocuments || err == mongo.ErrEmptySlice {
		// Server doesn't exist, create a new entry
		newServer := models.Server{
			ServerNumber: server,
			APIKey:       input.APIKey,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_, err := serverCollection.InsertOne(context.Background(), newServer)
		if err != nil {
			log.Println("ERROR: Failed to add new server:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add server"})
		}
		log.Printf("INFO: Server %d added successfully\n", server)
		return c.JSON(http.StatusCreated, map[string]string{"message": "Server added successfully"})
	} else {
		log.Println("ERROR: Unexpected error while querying server collection:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
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
	serverCollection := db.Collection("servers")
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

	// Initialize the collection
	serverCollection := db.Collection("servers")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"server": input.Server}

	// Check if the server exists
	var existingServer bson.M
	err := serverCollection.FindOne(ctx, filter).Decode(&existingServer)
	if err == mongo.ErrNoDocuments {
		// Add a new server if not found
		log.Println("INFO: Server not found. Adding new server.")
		_, err := serverCollection.InsertOne(ctx, bson.M{
			"server":       input.Server,
			"maintainance": input.Maintainance,
		})
		if err != nil {
			log.Println("ERROR: Failed to add new server:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		}
		return c.JSON(http.StatusCreated, map[string]string{"message": "Server added successfully."})
	} else if err != nil {
		log.Println("ERROR: Database error while checking for server:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	// Toggle or explicitly set maintenance status
	var newStatus bool
	if input.Server == 0 {
		// For server 0, explicitly set the maintenance status from input
		newStatus = input.Maintainance
		log.Printf("INFO: Explicitly setting maintenance status for server 0 to %t\n", newStatus)
	} else {
		// For other servers, toggle the maintenance status
		currentStatus := existingServer["maintainance"].(bool)
		newStatus = !currentStatus
		log.Printf("INFO: Toggling maintenance status for server %d from %t to %t\n", input.Server, currentStatus, newStatus)
	}

	// Perform the update
	update := bson.M{"$set": bson.M{"maintainance": newStatus}}
	_, err = serverCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println("ERROR: Failed to update maintenance status:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	log.Printf("INFO: Maintenance status updated successfully for server: %d\n", input.Server)
	return c.JSON(http.StatusOK, map[string]string{
		"message":      "Maintenance status updated successfully.",
		"maintainance": fmt.Sprintf("Server %d is now %t", input.Server, newStatus),
	})
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
	// Retrieve the database instance
	db := c.Get("db").(*mongo.Database)
	serverCollection := db.Collection("servers")

	// Define a struct to map the expected JSON payload
	type RequestBody struct {
		Server       string `json:"server"`
		ExchangeRate string `json:"exchangeRate,omitempty"` // Allow string input
		Margin       string `json:"margin,omitempty"`       // Allow string input
	}

	var input RequestBody

	// Bind the JSON payload
	if err := c.Bind(&input); err != nil {
		log.Println("ERROR: Failed to parse request body:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// Parse server as integer
	server, err := strconv.Atoi(input.Server)
	if err != nil {
		log.Println("ERROR: Server must be a valid number:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server must be a valid number"})
	}

	// Parse and validate exchangeRate
	var exchangeRate *float64
	if input.ExchangeRate != "" {
		rate, err := strconv.ParseFloat(input.ExchangeRate, 64)
		if err != nil {
			log.Println("ERROR: ExchangeRate must be a valid number:", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "ExchangeRate must be a valid number"})
		}
		exchangeRate = &rate
	}

	// Parse and validate margin
	var margin *float64
	if input.Margin != "" {
		mg, err := strconv.ParseFloat(input.Margin, 64)
		if err != nil {
			log.Println("ERROR: Margin must be a valid number:", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Margin must be a valid number"})
		}
		margin = &mg
	}

	// Ensure at least one field is provided
	updateFields := bson.M{}
	if exchangeRate != nil {
		updateFields["exchangeRate"] = *exchangeRate
	}
	if margin != nil {
		updateFields["margin"] = *margin
	}
	if len(updateFields) == 0 {
		log.Println("ERROR: No fields provided to update")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "At least one field (exchangeRate or margin) must be provided"})
	}

	// Update the server document
	filter := bson.M{"server": server}
	update := bson.M{"$set": updateFields}
	result, err := serverCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Println("ERROR: Failed to update server:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	if result.MatchedCount == 0 {
		log.Printf("ERROR: Server %d not found\n", server)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Server not found"})
	}

	log.Printf("INFO: Successfully updated server %d\n", server)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Exchange rate and/or margin updated successfully.",
		"server":       server,
		"exchangeRate": exchangeRate,
		"margin":       margin,
	})
}

func BlocKServer(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	type RequestPayload struct {
		Name         string `json:"name" validate:"required"`
		ServerNumber string `json:"serverNumber" validate:"required"`
		Block        bool   `json:"block"`
	}
	var payload RequestPayload
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request payload"})
	}

	if payload.Name == "" || payload.ServerNumber == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing required fields"})
	}

	serverNumber, err := strconv.Atoi(payload.ServerNumber)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid serverNumber value"})
	}

	serverListCollection := models.InitializeServerListCollection(db)
	filter := bson.M{"name": payload.Name, "servers.server": serverNumber}

	update := bson.M{
		"$set": bson.M{
			"servers.$.block": payload.Block,
			"updatedAt":       time.Now(),
		},
	}

	result, err := serverListCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update server block status"})
	}

	if result.MatchedCount == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "server or service not found"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "server block status updated successfully"})
}
