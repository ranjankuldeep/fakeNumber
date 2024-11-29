package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/services"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/mongo"
)

func SendSellingUpdate(c echo.Context) error {
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	sellingDetails, err := services.FetchSellingUpdate(ctx, db)
	if err != nil {
		logs.Logger.Error(err)
		logs.Logger.Info("Unable to get selling update")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Unable to Fetch selling details"})
	}
	err = services.SellingTeleBot(sellingDetails)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Unable to send telebot sell message")
	}
	return nil
}
