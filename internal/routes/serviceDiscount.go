package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterServiceDiscountRoutes sets up routes for service discounts.
func RegisterServiceDiscountRoutes(e *echo.Echo) {
	serviceGroup := e.Group("/service")

	// Define routes and link them to handler functions
	serviceGroup.POST("/add-discount", handlers.AddServiceDiscount)
	serviceGroup.GET("/get-discount", handlers.GetServiceDiscount)
	serviceGroup.DELETE("/delete-discount", handlers.DeleteServiceDiscount)
}
