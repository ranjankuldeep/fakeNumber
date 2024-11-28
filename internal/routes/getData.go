package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterGetDataRoutes sets up routes for data retrieval operations.
func RegisterGetDataRoutes(e *echo.Echo) {
	dataGroup := e.Group("/api/")

	// Define routes
	dataGroup.GET("get-service-data", handlers.GetServiceData)
	dataGroup.GET("get-service", handlers.GetUserServiceData)
	dataGroup.GET("get-service-data-admin", handlers.GetServiceDataAdmin)
	dataGroup.GET("get-service-data-server", handlers.GetServersData)
	dataGroup.GET("total-recharge-balance", handlers.TotalRecharge)
	dataGroup.GET("total-user-count", handlers.GetTotalUserCount)
	dataGroup.GET("get-server-balance", handlers.GetServerBalanceHandler)
}
