package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// addServiceDiscount handles adding or updating a service discount.
func AddServiceDiscount(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)

	// Define a struct to map the expected request body
	type RequestBody struct {
		Service  string  `json:"service"`
		Server   string  `json:"server"`
		Discount float64 `json:"discount"`
	}

	// Log the incoming request for debugging
	body := new(bytes.Buffer)
	body.ReadFrom(c.Request().Body)
	log.Printf("INFO: Incoming request body: %s\n", body.String())

	// Bind the request body
	var input RequestBody
	if err := json.NewDecoder(strings.NewReader(body.String())).Decode(&input); err != nil {
		log.Println("ERROR: Failed to parse request body:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	// Validate input
	if input.Service == "" || input.Server == "" {
		log.Println("ERROR: Missing service or server in the input")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Service and server are required."})
	}

	// Parse server number
	serverNumber, err := strconv.Atoi(input.Server)
	if err != nil {
		log.Println("ERROR: Invalid server number:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server must be a valid number."})
	}

	// Parse and validate discount
	discount, err := strconv.ParseFloat(fmt.Sprintf("%v", input.Discount), 64)
	if err != nil || discount < 0 {
		log.Println("ERROR: Invalid discount value:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Discount must be a valid number."})
	}

	// Initialize collection
	servicedDiscountCollection := models.InitializeServiceDiscountCollection(db)

	// Check if the service discount exists
	filter := bson.M{"service": input.Service, "server": serverNumber}
	var existingService models.ServiceDiscount
	err = servicedDiscountCollection.FindOne(context.TODO(), filter).Decode(&existingService)

	if err == mongo.ErrNoDocuments {
		// Add new discount
		log.Println("INFO: Adding new service discount")
		_, err = servicedDiscountCollection.InsertOne(context.TODO(), models.ServiceDiscount{
			Service:  input.Service,
			Server:   serverNumber,
			Discount: discount,
		})
		if err != nil {
			log.Println("ERROR: Failed to add service discount:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add service discount."})
		}
		return c.JSON(http.StatusCreated, map[string]string{"message": "Discount added successfully."})
	} else if err == nil {
		// Update existing discount
		log.Println("INFO: Updating existing service discount")
		update := bson.M{"$set": bson.M{"discount": discount}}
		_, err = servicedDiscountCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			log.Println("ERROR: Failed to update service discount:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update service discount."})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Discount updated successfully."})
	} else {
		log.Println("ERROR: Unexpected error while processing service discount:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}
}

// getServiceDiscount handles fetching all service discounts.
func GetServiceDiscount(c echo.Context) error {
	// Log: Start of the function
	log.Println("INFO: Starting GetServiceDiscount handler")

	// Retrieve the database instance
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	log.Println("INFO: Database instance retrieved successfully")

	// Initialize the service discount collection
	log.Println("INFO: Initializing service discount collection")
	serviceDiscountCollection := models.InitializeServiceDiscountCollection(db)

	// Fetch service discounts from the collection
	log.Println("INFO: Fetching all service discounts from the database")
	var serviceDiscounts []models.ServiceDiscount
	cursor, err := serviceDiscountCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Println("ERROR: Failed to fetch service discounts:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch service discounts"})
	}
	defer func() {
		if err := cursor.Close(context.TODO()); err != nil {
			log.Println("ERROR: Failed to close cursor:", err)
		} else {
			log.Println("INFO: Cursor closed successfully")
		}
	}()

	// Iterate over the cursor and decode service discounts
	log.Println("INFO: Decoding service discount data")
	for cursor.Next(context.TODO()) {
		var serviceDiscount models.ServiceDiscount
		if err := cursor.Decode(&serviceDiscount); err != nil {
			log.Println("ERROR: Error decoding service discount:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error decoding service discount"})
		}
		serviceDiscounts = append(serviceDiscounts, serviceDiscount)
		log.Printf("INFO: Service discount added: %+v\n", serviceDiscount)
	}

	// Check for any errors during cursor iteration
	if err := cursor.Err(); err != nil {
		log.Println("ERROR: Cursor iteration error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error iterating service discounts"})
	}

	// Log: Successfully fetched discounts
	log.Printf("INFO: Successfully fetched %d service discounts\n", len(serviceDiscounts))

	// Ensure an empty array is returned if no data exists
	if len(serviceDiscounts) == 0 {
		log.Println("INFO: No service discounts found, returning an empty array")
		return c.JSON(http.StatusOK, []models.ServiceDiscount{})
	}

	// Return the fetched service discounts
	log.Println("INFO: Returning service discounts")
	return c.JSON(http.StatusOK, serviceDiscounts)
}

// DeleteServiceDiscount handles deleting a specific service discount.
func DeleteServiceDiscount(c echo.Context) error {
	// Log the start of the function
	log.Println("INFO: Starting DeleteServiceDiscount handler")

	// Retrieve query parameters
	service := c.QueryParam("service")
	server := c.QueryParam("server")

	// Log the received parameters
	log.Printf("INFO: Received parameters - service: %s, server: %s\n", service, server)

	// Validate the input parameters
	if service == "" || server == "" {
		log.Println("ERROR: Missing required parameters - service or server")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Service and server are required."})
	}

	// Convert server to an integer
	serverNumber, err := strconv.Atoi(server)
	if err != nil {
		log.Printf("ERROR: Server parameter is not a valid number: %s\n", server)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server must be a valid number."})
	}

	// Retrieve the database instance
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	// Initialize the service discount collection
	log.Println("INFO: Initializing service discount collection")
	servicedDiscountCollection := models.InitializeServiceDiscountCollection(db)

	// Define the filter for the document to delete
	filter := bson.M{"service": service, "server": serverNumber}
	log.Printf("DEBUG: Filter being used for deletion: %+v\n", filter)

	// Perform the delete operation
	result, err := servicedDiscountCollection.DeleteOne(context.TODO(), filter)
	if err != nil {
		log.Println("ERROR: Failed to delete service discount:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete service discount."})
	}

	// Check if a document was deleted
	if result.DeletedCount == 0 {
		log.Println("INFO: No document found to delete with the given filter")
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Service discount not found."})
	}

	// Log the successful deletion
	log.Printf("INFO: Successfully deleted document. Deleted count: %d\n", result.DeletedCount)

	// Return success response
	return c.JSON(http.StatusOK, map[string]string{"message": "Service discount deleted successfully."})
}
