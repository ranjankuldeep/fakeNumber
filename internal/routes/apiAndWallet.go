package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterRoutes sets up the routes for the application
func RegisterWalletRoutes(e *echo.Echo) {
	e.GET("/api/api_key", handlers.APIKeyHandler)
	e.GET("/api/balance", handlers.BalanceHandler)
}
