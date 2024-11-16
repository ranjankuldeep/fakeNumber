package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterHistoryRoutes sets up routes for history-related operations.
func RegisterHistoryRoutes(e *echo.Echo) {
	historyGroup := e.Group("/")

	// Define routes
	historyGroup.GET("recharge-history", handlers.GetRechargeHistory)
	historyGroup.GET("transaction-history", handlers.GetTransactionHistory)
	historyGroup.POST("save-recharge-history", handlers.SaveRechargeHistory)
	historyGroup.GET("transaction-history-count", handlers.TransactionCount)
}
