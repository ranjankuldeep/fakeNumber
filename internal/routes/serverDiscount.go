package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterServerDiscountRoutes sets up routes for server discounts.
func RegisterServerDiscountRoutes(e *echo.Echo) {
	serverGroup := e.Group("/server")

	// Define routes and link them to handler functions
	serverGroup.POST("/add-discount", handlers.AddDiscount)
	serverGroup.GET("/get-discount", handlers.GetDiscount)
	serverGroup.DELETE("/delete-discount", handlers.DeleteDiscount)
}
