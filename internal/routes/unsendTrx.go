package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
)

// RegisterUnsendTrxRoutes sets up routes for unsend transactions.
func RegisterUnsendTrxRoutes(e *echo.Echo) {
	trxGroup := e.Group("/unsend-trx")

	// Define routes and link them to handler functions
	trxGroup.GET("", handlers.GetAllUnsendTrx)
	trxGroup.DELETE("", handlers.DeleteUnsendTrx)
}
