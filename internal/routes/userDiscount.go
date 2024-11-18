package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterUserRoutes sets up the user-related routes.
func RegisterUserDiscountRoutes(e *echo.Echo) {
	userGroup := e.Group("/api/users/")

	// Define routes and link them to controller functions
	userGroup.POST("add-discount", handlers.AddUserDiscount)
	userGroup.GET("get-discount", handlers.GetUserDiscount)
	userGroup.DELETE("delete-discount", handlers.DeleteUserDiscount)
	userGroup.GET("get-all-discounts", handlers.GetAllUserDiscounts)
}
