package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Raylynd6299/babel/internal/shared/config"
	"github.com/Raylynd6299/babel/pkg/gateway"
)

func main() {
	cfg := config.Load()

	router := gin.Default()

	// Middleware
	router.Use(gateway.CORSMiddleware())
	router.Use(gateway.LoggingMiddleware())

	// Parse type
	gatewayRateLimit := gateway.RateLimitConfig{
		RequestsPerSecond: cfg.RateLimit.RequestsPerSecond,
		Burst:             cfg.RateLimit.Burst,
	}

	router.Use(gateway.RateLimitMiddleware(gatewayRateLimit))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Setup service proxies
	gateway.SetupRoutes(router, cfg)

	log.Printf("API Gateway starting on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}
