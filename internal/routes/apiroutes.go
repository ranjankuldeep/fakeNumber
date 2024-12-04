package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

func RegisterApisRoutes(e *echo.Echo) {
	apiGroup := e.Group("/api")
	apiGroup.GET("/get-service", handlers.GetServiceDataApi)
}
