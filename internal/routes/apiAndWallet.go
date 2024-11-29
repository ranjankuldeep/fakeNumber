package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterApiWalletRoutes sets up routes for API and Wallet operations.
func RegisterApiWalletRoutes(e *echo.Echo) {
	apiWalletGroup := e.Group("/api/")
	// Define routes
	apiWalletGroup.GET("api_key", handlers.ApiKey)
	apiWalletGroup.GET("balance", handlers.BalanceHandler)
	apiWalletGroup.GET("change_api_key", handlers.ChangeAPIKeyHandler)
	apiWalletGroup.POST("edit-balance", handlers.UpdateWalletBalanceHandler)
	apiWalletGroup.POST("update-qr", handlers.UpiQRUpdateHandler)
	apiWalletGroup.GET("get-qr", handlers.GetUpiQR)
	apiWalletGroup.POST("add-recharge-api", handlers.CreateOrUpdateAPIKeyHandler)
	apiWalletGroup.GET("get-recharge-api", handlers.GetAPIKeyHandler)
}
