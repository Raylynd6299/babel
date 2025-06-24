package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// GenerateSecretKey generates a cryptographically secure secret key
func GenerateSecretKey() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateToken is a standalone function for simple token validation
func ValidateToken(tokenString, secretKey string) (string, error) {
	config := Config{
		SecretKey: secretKey,
	}
	service := NewService(config)

	claims, err := service.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	return claims.UserID, nil
}

// ExtractUserID is a standalone function to extract user ID from token
func ExtractUserID(tokenString, secretKey string) (string, error) {
	config := Config{
		SecretKey: secretKey,
	}
	service := NewService(config)

	return service.ExtractUserID(tokenString)
}

// TokenInfo represents information about a token
type TokenInfo struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Type      string    `json:"type"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	Issuer    string    `json:"issuer"`
	IsExpired bool      `json:"is_expired"`
	TTL       int64     `json:"ttl_seconds"`
}

// GetTokenInfo extracts detailed information from a token
func (s *Service) GetTokenInfo(tokenString string) (*TokenInfo, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := claims.ExpiresAt.Time
	isExpired := expiresAt.Before(now)
	ttl := int64(0)
	if !isExpired {
		ttl = int64(expiresAt.Sub(now).Seconds())
	}

	return &TokenInfo{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Type:      claims.Type,
		ExpiresAt: expiresAt,
		IssuedAt:  claims.IssuedAt.Time,
		Issuer:    claims.Issuer,
		IsExpired: isExpired,
		TTL:       ttl,
	}, nil
}

// BatchValidateTokens validates multiple tokens at once
func (s *Service) BatchValidateTokens(tokens []string) map[string]error {
	results := make(map[string]error)

	for _, token := range tokens {
		_, err := s.ValidateToken(token)
		results[token] = err
	}

	return results
}
