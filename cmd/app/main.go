package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ranjankuldeep/fakeNumber/internal/database"
	"github.com/ranjankuldeep/fakeNumber/internal/routes"
)

func Load(envFile string) {
	err := godotenv.Load(dir(envFile))
	if err != nil {
		panic(fmt.Errorf("Error loading .env file: %w", err))
	}
}
func dir(envFile string) string {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			break
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			panic(fmt.Errorf("go.mod not found"))
		}
		currentDir = parent
	}

	return filepath.Join(currentDir, envFile)
}

var (
	store = sessions.NewCookieStore([]byte("mY FUckingSEcretKey"))
)

func main() {
	Load(".env")
	e := echo.New()

	uri := "mongodb+srv://test2:amardeep885@cluster0.blfflhg.mongodb.net/Express-Backend?retryWrites=true&w=majority"

	// CORS middleware to allow only http://localhost:5173
	// Configure CORS to allow requests from http://localhost:5173 with credentials
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:5173"}, // Allow only localhost:5173
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true, // Enable credentials
	}))

	// Connect to the MongoDB client
	client, err := database.ConnectDB(uri)
	if err != nil {
		log.Fatal("Error initializing MongoDB connection:", err)
	}

	// Select the specific database
	db := client.Database("Express-Backend")

	// Middleware to set the DB in the context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set DB in the context for every request
			c.Set("db", db)
			return next(c)
		}
	})
	routes.RegisterServiceRoutes(e)
	routes.RegisterGetDataRoutes(e)
	routes.RegisterUserRoutes(e)
	routes.RegisterApiWalletRoutes(e)
	routes.RegisterHistoryRoutes(e)
	routes.RegisterRechargeRoutes(e)
	routes.RegisterUserDiscountRoutes(e)

	e.Logger.Fatal(e.Start(":8000"))
}
