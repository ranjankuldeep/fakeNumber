package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterRoutes sets up the routes for the application
func RegisterRoutes(e *echo.Echo) {
	e.GET("/api/get-service-data", handlers.GetServiceData)
	// e.GET("/get-service", handlers.GetUserServiceData)
	// e.GET("/get-service-data-admin", handlers.GetServiceDataAdmin)
	// e.GET("/get-service-data-server", handlers.GetServersData)
	// e.GET("/total-recharge-balance", handlers.TotalRecharge)
	// e.GET("/total-user-count", handlers.GetTotalUserCount)
}
