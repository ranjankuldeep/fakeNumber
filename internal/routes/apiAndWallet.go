package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterApiWalletRoutes sets up routes for API and Wallet operations.
func RegisterApiWalletRoutes(e *echo.Echo) {
	apiWalletGroup := e.Group("/api")

	// Define routes
	apiWalletGroup.GET("api_key", handlers.ApiKey)
	apiWalletGroup.GET("balance", handlers.Balance)
	apiWalletGroup.GET("change_api_key", handlers.ChangeApiKey)
	apiWalletGroup.POST("edit-balance", handlers.UpdateBalance)
	apiWalletGroup.POST("update-qr", handlers.UpiQRUpdate)
	apiWalletGroup.GET("get-qr", handlers.GetUpiQR)
	apiWalletGroup.POST("add-recharge-api", handlers.CreateOrUpdateApiKey)
	apiWalletGroup.GET("get-recharge-api", handlers.GetApiKey)
}
