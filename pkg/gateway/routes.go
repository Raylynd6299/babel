package gateway

import (
	"bytes"
	"io"
	"net/http"
	"time"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/Raylynd6299/babel/internal/shared/config"
	polyfyjwt "github.com/Raylynd6299/babel/pkg/jwt"
)

type ServiceConfig struct {
	AuthServiceURL       string
	ContentServiceURL    string
	ProgressServiceURL   string
	VocabularyServiceURL string
	PhoneticServiceURL   string
	SocialServiceURL     string
	GameServiceURL       string
	AnalyticsServiceURL  string
	jwtService           *polyfyjwt.Service
}

func SetupRoutes(router *gin.Engine, cfg *config.Config) {

	// Create JWT Service
	jwtConfig := polyfyjwt.Config{
		SecretKey:            cfg.JWTSecret,
		AccessTokenDuration:  time.Hour * 2,
		RefreshTokenDuration: time.Hour * 24 * 7,
		Issuer:               "polyfy-auth",
	}

	jwtService := polyfyjwt.NewService(jwtConfig)

	services := ServiceConfig{
		AuthServiceURL:       cfg.AuthServiceURL,
		ContentServiceURL:    cfg.ContentServiceURL,
		ProgressServiceURL:   cfg.ProgressServiceURL,
		VocabularyServiceURL: cfg.VocabularyServiceURL,
		PhoneticServiceURL:   cfg.PhoneticServiceURL,
		SocialServiceURL:     cfg.SocialServiceURL,
		GameServiceURL:       cfg.GameServiceURL,
		AnalyticsServiceURL:  cfg.AnalyticsServiceURL,
		jwtService:           jwtService,
	}

	v1 := router.Group("/api/v1")

	// Auth routes (no auth required)
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", proxyTo(services.AuthServiceURL))
		authGroup.POST("/login", proxyTo(services.AuthServiceURL))
		authGroup.POST("/refresh", proxyTo(services.AuthServiceURL))
		authGroup.POST("/logout", proxyTo(services.AuthServiceURL))
		authGroup.POST("/forgot-password", proxyTo(services.AuthServiceURL))
		authGroup.POST("/reset-password", proxyTo(services.AuthServiceURL))
	}

	// Protected routes
	protected := v1.Group("/")
	protected.Use(services.jwtService.AuthMiddleware())

	// User routes
	userGroup := protected.Group("/users")
	{
		userGroup.GET("/me", proxyTo(services.AuthServiceURL))
		userGroup.PUT("/me", proxyTo(services.AuthServiceURL))
		userGroup.DELETE("/me", proxyTo(services.AuthServiceURL))
		userGroup.GET("/languages", proxyTo(services.AuthServiceURL))
		userGroup.POST("/languages", proxyTo(services.AuthServiceURL))
		userGroup.PUT("/languages/:id", proxyTo(services.AuthServiceURL))
		userGroup.DELETE("/languages/:id", proxyTo(services.AuthServiceURL))
	}

	// Content routes
	contentGroup := protected.Group("/content")
	{
		contentGroup.GET("/", proxyTo(services.ContentServiceURL))
		contentGroup.POST("/", proxyTo(services.ContentServiceURL))
		contentGroup.GET("/:id", proxyTo(services.ContentServiceURL))
		contentGroup.PUT("/:id", proxyTo(services.ContentServiceURL))
		contentGroup.DELETE("/:id", proxyTo(services.ContentServiceURL))
		contentGroup.POST("/:id/rate", proxyTo(services.ContentServiceURL))
		contentGroup.GET("/:id/episodes", proxyTo(services.ContentServiceURL))
		contentGroup.POST("/:id/episodes", proxyTo(services.ContentServiceURL))
		contentGroup.GET("/recommendations", proxyTo(services.ContentServiceURL))
		contentGroup.GET("/languages", proxyTo(services.ContentServiceURL))
	}

	// Progress routes
	progressGroup := protected.Group("/progress")
	{
		progressGroup.POST("/input", proxyTo(services.ProgressServiceURL))
		progressGroup.GET("/stats", proxyTo(services.ProgressServiceURL))
		progressGroup.GET("/analytics", proxyTo(services.ProgressServiceURL))
		progressGroup.GET("/recent", proxyTo(services.ProgressServiceURL))
		progressGroup.GET("/calendar", proxyTo(services.ProgressServiceURL))
	}

	// Vocabulary routes
	vocabGroup := protected.Group("/vocabulary")
	{
		vocabGroup.POST("/", proxyTo(services.VocabularyServiceURL))
		vocabGroup.GET("/", proxyTo(services.VocabularyServiceURL))
		vocabGroup.GET("/reviews", proxyTo(services.VocabularyServiceURL))
		vocabGroup.POST("/reviews", proxyTo(services.VocabularyServiceURL))
		vocabGroup.GET("/stats", proxyTo(services.VocabularyServiceURL))
		vocabGroup.DELETE("/:id", proxyTo(services.VocabularyServiceURL))
		vocabGroup.GET("/search", proxyTo(services.VocabularyServiceURL))
		vocabGroup.POST("/import", proxyTo(services.VocabularyServiceURL))
		vocabGroup.GET("/export", proxyTo(services.VocabularyServiceURL))
	}

	// Phonetic routes
	phoneticGroup := protected.Group("/phonetic")
	{
		phoneticGroup.GET("/languages/:language_id/phonemes", proxyTo(services.PhoneticServiceURL))
		phoneticGroup.GET("/progress", proxyTo(services.PhoneticServiceURL))
		phoneticGroup.POST("/practice", proxyTo(services.PhoneticServiceURL))
		phoneticGroup.GET("/exercises", proxyTo(services.PhoneticServiceURL))
	}

	// Social routes
	socialGroup := protected.Group("/social")
	{
		socialGroup.GET("/profile/:user_id", proxyTo(services.SocialServiceURL))
		socialGroup.POST("/follow/:user_id", proxyTo(services.SocialServiceURL))
		socialGroup.DELETE("/follow/:user_id", proxyTo(services.SocialServiceURL))
		socialGroup.GET("/followers", proxyTo(services.SocialServiceURL))
		socialGroup.GET("/following", proxyTo(services.SocialServiceURL))
		socialGroup.GET("/feed", proxyTo(services.SocialServiceURL))
		socialGroup.GET("/leaderboard", proxyTo(services.SocialServiceURL))
	}

	// Gamification routes
	gameGroup := protected.Group("/gamification")
	{
		gameGroup.GET("/achievements", proxyTo(services.GameServiceURL))
		gameGroup.GET("/leaderboard", proxyTo(services.GameServiceURL))
		gameGroup.GET("/challenges", proxyTo(services.GameServiceURL))
		gameGroup.POST("/challenges/:id/join", proxyTo(services.GameServiceURL))
		gameGroup.GET("/points", proxyTo(services.GameServiceURL))
	}

	// Analytics routes
	analyticsGroup := protected.Group("/analytics")
	{
		analyticsGroup.GET("/dashboard", proxyTo(services.AnalyticsServiceURL))
		analyticsGroup.GET("/reports/:type", proxyTo(services.AnalyticsServiceURL))
		analyticsGroup.POST("/events", proxyTo(services.AnalyticsServiceURL))
	}
}

func proxyTo(serviceURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build target URL
		targetURL := serviceURL + c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			targetURL += "?" + c.Request.URL.RawQuery
		}

		// Create new request
		var body io.Reader
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
				return
			}
			body = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequest(c.Request.Method, targetURL, body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
			return
		}

		// Copy headers
		for key, values := range c.Request.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Add user ID from context if available
		if userID, exists := c.Get("user_id"); exists {
			req.Header.Set("X-User-ID", userID.(string))
		}

		// Make request
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Service unavailable"})
			return
		}
		defer resp.Body.Close()

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// Copy response body
		c.Status(resp.StatusCode)
		io.Copy(c.Writer, resp.Body)
	}
}
