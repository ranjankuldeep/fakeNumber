package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterRoutes sets up the routes for the application
func RegisterUserRoutes(e *echo.Echo) {
	e.POST("/google-signup", handlers.GoogleSignup)
	e.POST("/google-login", handlers.GoogleLogin)
}
