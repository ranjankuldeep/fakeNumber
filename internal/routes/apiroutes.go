package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

func RegisterApisRoutes(e *echo.Echo) {
	e.GET("/number", handlers.GetNumberHandlerApi)
}
