package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterRoutes sets up the routes for the application
func RegisterDataRoutes(e *echo.Echo) {
	e.GET("/api/get-service-data", handlers.GetServiceData)
}
