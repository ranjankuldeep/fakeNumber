package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Database collection
var unsendTrxCollection *mongo.Collection

// GetAllUnsendTrx retrieves all unsent transactions
func GetAllUnsendTrx(c echo.Context) error {
	var allUnsendTrx []models.UnsendTrx

	cursor, err := unsendTrxCollection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Println("Error fetching unsent transactions:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var trx models.UnsendTrx
		if err := cursor.Decode(&trx); err != nil {
			log.Println("Error decoding transaction:", err)
			continue
		}
		allUnsendTrx = append(allUnsendTrx, trx)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": allUnsendTrx})
}

// DeleteUnsendTrx deletes an unsent transaction by ID
func DeleteUnsendTrx(c echo.Context) error {
	id := c.QueryParam("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID is required"})
	}

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	result, err := unsendTrxCollection.DeleteOne(context.Background(), bson.M{"_id": objectId})
	if err != nil {
		log.Println("Error deleting transaction:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	if result.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Document not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Document deleted successfully"})
}
