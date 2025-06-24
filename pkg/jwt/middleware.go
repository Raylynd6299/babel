package jwt

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates a JWT authentication middleware
func (s *Service) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractTokenFromHeader(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		claims, err := s.ValidateAccessToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("token_id", claims.ID)

		c.Next()
	}
}

// OptionalAuthMiddleware validates token if present but doesn't require it
func (s *Service) OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractTokenFromHeader(c.GetHeader("Authorization"))
		if tokenString != "" {
			claims, err := s.ValidateAccessToken(tokenString)
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("token_id", claims.ID)
				c.Set("authenticated", true)
			} else {
				c.Set("authenticated", false)
			}
		} else {
			c.Set("authenticated", false)
		}

		c.Next()
	}
}

// RefreshTokenMiddleware validates refresh tokens
func (s *Service) RefreshTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractTokenFromHeader(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		claims, err := s.ValidateRefreshToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("token_id", claims.ID)

		c.Next()
	}
}

// AdminMiddleware requires admin role (placeholder implementation)
func (s *Service) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First validate the token
		tokenString := extractTokenFromHeader(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		claims, err := s.ValidateAccessToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// TODO: Check if user has admin role
		// This would typically involve checking the user's role in the database
		// For now, we'll just set the user info and continue
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("token_id", claims.ID)

		c.Next()
	}
}

// Helper function to extract token from Authorization header
func extractTokenFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	// Check if it starts with "Bearer "
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Return as-is if no Bearer prefix
	return authHeader
}

// GetUserIDFromContext extracts user ID from gin context
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	if id, ok := userID.(string); ok {
		return id, true
	}

	return "", false
}

// GetUserEmailFromContext extracts user email from gin context
func GetUserEmailFromContext(c *gin.Context) (string, bool) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", false
	}

	if e, ok := email.(string); ok {
		return e, true
	}

	return "", false
}

// IsAuthenticated checks if the current request is authenticated
func IsAuthenticated(c *gin.Context) bool {
	authenticated, exists := c.Get("authenticated")
	if !exists {
		// If not set, check if user_id exists
		_, exists := c.Get("user_id")
		return exists
	}

	if auth, ok := authenticated.(bool); ok {
		return auth
	}

	return false
}
