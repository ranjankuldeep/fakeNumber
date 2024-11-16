package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterRoutes sets up the routes for the application
func RegisterUserRoutes(e *echo.Echo) {
	e.POST("/api/signup", handlers.Signup)
	e.POST("/api/verify-otp", handlers.VerifyOTP)
	// e.POST("/api/resend-otp", handlers.ResendOTP)
	// e.POST("/api/login", handlers.Login)
	e.POST("/api/forgot-password", handlers.ForgotPassword)
	e.POST("/api/resend-forgot-otp", handlers.ResendForgotOTP)
	e.POST("/api/verify-forgot-otp", handlers.ForgotVerifyOTP)
	e.POST("/api/change-password-unauthenticated", handlers.ChangePasswordUnauthenticated)
	e.POST("/api/change-password-authenticated", handlers.ChangePasswordAuthenticated)
	e.POST("/api/google-login", handlers.GoogleLogin)
	e.POST("/api/google-signup", handlers.GoogleSignup)

	// Admin APIs with `/api` prefix
	e.GET("/api/get-all-users", handlers.GetAllUsers)
	e.GET("/api/get-user", handlers.GetUser)
	e.POST("/api/user", handlers.BlockUnblockUser)
	e.GET("/api/blocked-user", handlers.BlockedUser)
	e.GET("/api/get-all-blocked-users", handlers.GetAllBlockedUsers)
	e.GET("/api/orders", handlers.GetOrdersByUserId)
}
