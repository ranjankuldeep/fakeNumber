package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterRechargeRoutes sets up routes for recharge-related operations.
func RegisterRechargeRoutes(e *echo.Echo) {
	rechargeGroup := e.Group("/api/")

	// Define GET routes
	rechargeGroup.GET("recharge-upi-transaction", handlers.RechargeUpiApi)
	rechargeGroup.GET("recharge-trx-transaction", handlers.RechargeTrxApi)
	rechargeGroup.GET("exchange-rate", handlers.ExchangeRate)
	rechargeGroup.GET("get-recharge-maintenance", handlers.GetMaintenanceStatus)
	// rechargeGroup.GET("get-minimum-recharge", handlers.GetMinimumRecharge)

	// Define POST routes
	rechargeGroup.POST("recharge-maintenance-toggle", handlers.ToggleMaintenance)
	// rechargeGroup.POST("add-minimum-recharge", handlers.AddMinimumRecharge)

	// Define DELETE routes
	// rechargeGroup.DELETE("delete-minimum-recharge", handlers.DeleteMinimumRecharge)

}
