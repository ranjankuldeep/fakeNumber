package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterServerDataRoutes sets up routes for server data operations.
func RegisterServerDataRoutes(e *echo.Echo) {
	serverGroup := e.Group("/")

	// Define GET routes
	serverGroup.GET("save-server-data-once", handlers.SaveServerDataOnce)
	serverGroup.GET("check-duplicates", handlers.CheckDuplicates)
	serverGroup.GET("merge-duplicates", handlers.MergeDuplicates)
	serverGroup.GET("update-server-prices", handlers.UpdateServerPrices)

	// Define POST routes
	serverGroup.POST("add-new-service-data", handlers.AddNewServiceData)
	serverGroup.POST("add-ccpay-service-name-data", handlers.AddCcpayServiceNameData)
	serverGroup.POST("service-data-block-unblock", handlers.BlockUnblockService)
	serverGroup.POST("delete-service", handlers.DeleteService)
}
