package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterRoutes sets up the routes for the application
func RegisterUserRoutes(e *echo.Echo) {
	e.POST("/api/google-signup", handlers.GoogleSignup)
	e.POST("/api/google-login", handlers.GoogleLogin)
}
