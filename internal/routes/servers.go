package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

func RegisterServerRoutes(e *echo.Echo) {
	serverGroup := e.Group("/api/")

	serverGroup.POST("add-server", handlers.AddServer)
	serverGroup.GET("get-server", handlers.GetServer)
	serverGroup.DELETE("delete-server", handlers.DeleteServer)
	serverGroup.POST("maintainance-server", handlers.MaintainanceServer)
	serverGroup.GET("maintainance-check", handlers.GetServerZero)
	serverGroup.POST("add-token-server9", handlers.AddTokenForServer9)
	serverGroup.GET("get-token-server9", handlers.GetTokenForServer9)
	serverGroup.POST("add-exchange-rate-margin-server", handlers.UpdateExchangeRateAndMargin)
	serverGroup.POST("service-data-block-unblock", handlers.BlocKServer)
}
