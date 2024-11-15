package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterServiceRoutes sets up the routes for the application
func RegisterServiceRoutes(e *echo.Echo) {
	e.GET("/api/get-number", handlers.HandleGetNumberRequest)
	e.GET("/api/check-otp", handlers.HandleCheckOTP)
	e.GET("/api/get-otp", handlers.HandleGetOtp)
	e.GET("/api/number-cancel", handlers.HandleNumberCancel)
}
