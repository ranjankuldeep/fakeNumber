package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

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
		Service  string  `json:"service"`  // JSON field name
		Server   string  `json:"server"`   // JSON field name
		Discount float64 `json:"discount"` // JSON field name
	}

	// Initialize an instance of the struct
	var input RequestBody
	if err := c.Bind(input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}
	serverNumber, _ := strconv.Atoi(input.Server)

	if input.Service == "" || input.Server == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Service and server are required."})
	}

	discount, err := strconv.ParseFloat(fmt.Sprintf("%v", input.Discount), 64)
	if err != nil || discount < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Discount must be a valid number."})
	}

	// Check if the service discount exists
	filter := bson.M{"service": input.Service, "server": input.Server}
	servicedDiscountCollection := models.InitializeServiceDiscountCollection(db)
	var existingService models.ServiceDiscount
	err = servicedDiscountCollection.FindOne(context.TODO(), filter).Decode(&existingService)

	if err == mongo.ErrNoDocuments {
		// Add new discount
		_, err = servicedDiscountCollection.InsertOne(context.TODO(), models.ServiceDiscount{
			Service:  input.Service,
			Server:   serverNumber,
			Discount: discount,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add service discount."})
		}
		return c.JSON(http.StatusCreated, map[string]string{"message": "Discount added successfully."})
	} else if err == nil {
		// Update existing discount
		update := bson.M{"$set": bson.M{"discount": discount}}
		_, err = servicedDiscountCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update service discount."})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Discount updated successfully."})
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}
}

// getServiceDiscount handles fetching all service discounts.
func GetServiceDiscount(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	// Define a struct to map the expected request body
	type RequestBody struct {
		Service  string  `json:"service"`  // JSON field name
		Server   string  `json:"server"`   // JSON field name
		Discount float64 `json:"discount"` // JSON field name
	}

	// Initialize an instance of the struct
	var input RequestBody
	if err := c.Bind(input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	servicedDiscountCollection := models.InitializeServiceDiscountCollection(db)
	var serviceDiscounts []models.ServiceDiscount
	cursor, err := servicedDiscountCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch service discounts."})
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var serviceDiscount models.ServiceDiscount
		if err := cursor.Decode(&serviceDiscount); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error decoding service discount."})
		}
		serviceDiscounts = append(serviceDiscounts, serviceDiscount)
	}

	return c.JSON(http.StatusOK, serviceDiscounts)
}

// deleteServiceDiscount handles deleting a specific service discount.
func DeleteServiceDiscount(c echo.Context) error {
	service := c.QueryParam("service")
	server := c.QueryParam("server")

	if service == "" || server == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Service and server are required."})
	}
	db := c.Get("db").(*mongo.Database)

	servicedDiscountCollection := models.InitializeServiceDiscountCollection(db)
	filter := bson.M{"service": service, "server": server}
	result, err := servicedDiscountCollection.DeleteOne(context.TODO(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete service discount."})
	}

	if result.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Service discount not found."})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Service discount deleted successfully."})
}
