package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func main() {
	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Middlewares: request logging and panic recovery
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Serve static files under /static (e.g., /static/style.css -> public/style.css)
	e.Static("/static", "public")

	// Return the HTML file on root path
	e.GET("/", func(c echo.Context) error {
		return c.File("public/index.html")
	})

	// Health check endpoint
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(200, "ok")
	})

	port := getEnv("PORT", "8080")
	if err := e.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}
