package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

func RegisterBlockUsersRoutes(e *echo.Echo) {
	blockGroup := e.Group("/api/")

	blockGroup.POST("block-status-toggle", handlers.ToggleBlockStatus)
	blockGroup.GET("get-block-status", handlers.GetBlockStatus)
	blockGroup.GET("save-block-types", handlers.SavePredefinedBlockTypes)
	blockGroup.DELETE("block-fraud-clear", handlers.BlockFraudClear)
}
