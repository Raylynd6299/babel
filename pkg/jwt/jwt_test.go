package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJWTService(t *testing.T) {
	config := Config{
		SecretKey:            "test-secret-key",
		AccessTokenDuration:  time.Minute * 15,
		RefreshTokenDuration: time.Hour * 24,
		Issuer:               "test-issuer",
	}

	service := NewService(config)

	t.Run("GenerateTokenPair", func(t *testing.T) {
		userID := "test-user-id"
		email := "test@example.com"

		tokenPair, err := service.GenerateTokenPair(userID, email)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenPair.AccessToken)
		assert.NotEmpty(t, tokenPair.RefreshToken)
		assert.Equal(t, int(config.AccessTokenDuration.Seconds()), tokenPair.ExpiresIn)
	})

	t.Run("ValidateAccessToken", func(t *testing.T) {
		userID := "test-user-id"
		email := "test@example.com"

		accessToken, err := service.GenerateAccessToken(userID, email)
		assert.NoError(t, err)

		claims, err := service.ValidateAccessToken(accessToken)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, "access", claims.Type)
	})

	t.Run("ValidateRefreshToken", func(t *testing.T) {
		userID := "test-user-id"
		email := "test@example.com"

		refreshToken, err := service.GenerateRefreshToken(userID, email)
		assert.NoError(t, err)

		claims, err := service.ValidateRefreshToken(refreshToken)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, "refresh", claims.Type)
	})

	t.Run("RefreshAccessToken", func(t *testing.T) {
		userID := "test-user-id"
		email := "test@example.com"

		refreshToken, err := service.GenerateRefreshToken(userID, email)
		assert.NoError(t, err)

		newAccessToken, err := service.RefreshAccessToken(refreshToken)
		assert.NoError(t, err)
		assert.NotEmpty(t, newAccessToken)

		// Validate the new access token
		claims, err := service.ValidateAccessToken(newAccessToken)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		// Create service with very short expiration
		shortConfig := config
		shortConfig.AccessTokenDuration = time.Millisecond
		shortService := NewService(shortConfig)

		userID := "test-user-id"
		email := "test@example.com"

		accessToken, err := shortService.GenerateAccessToken(userID, email)
		assert.NoError(t, err)

		// Wait for token to expire
		time.Sleep(time.Millisecond * 2)

		_, err = shortService.ValidateAccessToken(accessToken)
		assert.Error(t, err)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		invalidToken := "invalid.token.string"

		_, err := service.ValidateAccessToken(invalidToken)
		assert.Error(t, err)
	})

	t.Run("WrongTokenType", func(t *testing.T) {
		userID := "test-user-id"
		email := "test@example.com"

		refreshToken, err := service.GenerateRefreshToken(userID, email)
		assert.NoError(t, err)

		// Try to validate refresh token as access token
		_, err = service.ValidateAccessToken(refreshToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not an access token")
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("generateJTI", func(t *testing.T) {
		jti1 := generateJTI()
		jti2 := generateJTI()

		assert.NotEmpty(t, jti1)
		assert.NotEmpty(t, jti2)
		assert.NotEqual(t, jti1, jti2) // Should be unique
	})

	t.Run("randomString", func(t *testing.T) {
		str1 := randomString(10)
		str2 := randomString(10)

		assert.Len(t, str1, 10)
		assert.Len(t, str2, 10)
		assert.NotEqual(t, str1, str2) // Should be different
	})
}
