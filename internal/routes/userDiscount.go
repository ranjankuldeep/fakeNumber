package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

func RegisterUserDiscountRoutes(e *echo.Echo) {
	userGroup := e.Group("/api/users/")

	userGroup.POST("add-discount", handlers.AddUserDiscount)
	userGroup.GET("get-discount", handlers.GetUserDiscount)
	userGroup.DELETE("delete-discount", handlers.DeleteUserDiscount)
	userGroup.GET("get-all-discounts", handlers.GetAllUserDiscounts)
}
