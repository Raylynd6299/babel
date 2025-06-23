package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db        *gorm.DB
	jwtSecret string
}

func NewService(db *gorm.DB, jwtSecret string) *Service {
	return &Service{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) generateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 2).Unix(), // 2 hours
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) generateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":     time.Now().Unix(),
		"type":    "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	// Check if user exists
	var existingUser User
	if err := s.db.Unscoped().Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {

		// If user was soft deleted, we can suggest recovery
		if existingUser.DeletedAt.Valid {
			return nil, errors.New("account with this email/username was deleted. Contact support for recovery")
		}

		return nil, errors.New("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create User
	user := User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
	}

	// Try to creat User
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	// Generate Tokens
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600 * 2, // 2 hours
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	var user User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Update Last Login
	now := time.Now()
	user.LastLogin = &now
	s.db.Save(&user)

	// Generate Tokens
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600 * 2, // 2 hours
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResponse, error) {
	// Parse and validate refresh token
	token, err := jwt.ParseWithClaims(refreshToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		// Verificar que es un refresh token
		if claims.ID != "refresh" {
			return nil, errors.New("not a refresh token")
		}

		// Verificar que el usuario existe y no está eliminado
		var user User
		if err := s.db.Where("id = ? AND is_active = ?", claims.Subject, true).First(&user).Error; err != nil {
			return nil, errors.New("user not found or inactive")
		}

		// Generar nuevo access token
		accessToken, err := s.generateAccessToken(claims.Subject)
		if err != nil {
			return nil, err
		}

		return &RefreshTokenResponse{
			AccessToken: accessToken,
			ExpiresIn:   3600,
		}, nil
	}

	return nil, errors.New("invalid token")
}

func (s *Service) GetUserByID(ctx context.Context, userID string) (*User, error) {
	var user User
	err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (s *Service) DeleteAccount(ctx context.Context, userID string) error {
	var user User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return errors.New("user not found")
	}

	// Soft delete by setting is_active to false
	user.IsActive = false
	if err := s.db.Save(&user).Error; err != nil {
		return err
	}

	// GORM soft delete - esto automáticamente establece deleted_at
	if err := s.db.Delete(&user).Error; err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*User, error) {
	var user User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}

	// Update only provided fields
	updates := make(map[string]interface{})

	if req.FirstName != "" {
		updates["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		updates["last_name"] = req.LastName
	}
	if req.Username != "" {
		// Check if username is already taken by another non-deleted user
		var existingUser User
		err := s.db.Where("username = ? AND id != ?", req.Username, userID).First(&existingUser).Error
		if err == nil {
			return nil, errors.New("username already taken")
		}
		updates["username"] = req.Username
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Reload user to get updated data
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
