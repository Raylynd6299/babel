package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims structure for JWT tokens
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Config for JWT settings
type Config struct {
	SecretKey            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Issuer               string
}

// Service handles JWT operations
type Service struct {
	config Config
}

// NewService creates a new JWT service
func NewService(config Config) *Service {
	// Set default values if not provided
	if config.AccessTokenDuration == 0 {
		config.AccessTokenDuration = time.Hour
	}
	if config.RefreshTokenDuration == 0 {
		config.RefreshTokenDuration = time.Hour * 24 * 7 // 7 days
	}
	if config.Issuer == "" {
		config.Issuer = "polyfy"
	}

	return &Service{config: config}
}

// GenerateTokenPair creates both access and refresh tokens
func (s *Service) GenerateTokenPair(userID, email string) (*TokenPair, error) {
	accessToken, err := s.GenerateAccessToken(userID, email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateRefreshToken(userID, email)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.config.AccessTokenDuration.Seconds()),
	}, nil
}

// GenerateAccessToken creates a new access token
func (s *Service) GenerateAccessToken(userID, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.config.Issuer,
			ID:        generateJTI(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.SecretKey))
}

// GenerateRefreshToken creates a new refresh token
func (s *Service) GenerateRefreshToken(userID, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.config.Issuer,
			ID:        generateJTI(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.SecretKey))
}

// ValidateToken validates and parses a JWT token
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateAccessToken specifically validates access tokens
func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != "access" {
		return nil, errors.New("not an access token")
	}

	if s.IsTokenExpired(tokenString) {
		return nil, errors.New("access denied, token expired")
	}

	return claims, nil
}

// ValidateRefreshToken specifically validates refresh tokens
func (s *Service) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != "refresh" {
		return nil, errors.New("not a refresh token")
	}

	return claims, nil
}

// RefreshAccessToken creates a new access token from a valid refresh token
func (s *Service) RefreshAccessToken(refreshTokenString string) (string, error) {
	claims, err := s.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	// Generate new access token with same user info
	return s.GenerateAccessToken(claims.UserID, claims.Email)
}

// ExtractUserID extracts user ID from token without full validation (for middleware)
func (s *Service) ExtractUserID(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.SecretKey), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims.UserID, nil
	}

	return "", errors.New("invalid token claims")
}

// IsTokenExpired checks if a token is expired without full validation
func (s *Service) IsTokenExpired(tokenString string) bool {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.SecretKey), nil
	})

	if err != nil {
		return true
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims.ExpiresAt.Time.Before(time.Now())
	}

	return true
}

// GetTokenTTL returns the remaining time until token expiration
func (s *Service) GetTokenTTL(tokenString string) (time.Duration, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl < 0 {
		return 0, errors.New("token expired")
	}

	return ttl, nil
}

// RevokeToken adds token to a blacklist (requires external blacklist implementation)
// This is a placeholder - you would implement this with Redis or database
func (s *Service) RevokeToken(tokenString string) error {
	// TODO: Implement token blacklisting
	// This could involve storing the JTI in Redis with the token's expiration time
	return nil
}

// Helper functions

// generateJTI generates a unique token ID
func generateJTI() string {
	return time.Now().Format("20060102150405") + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
