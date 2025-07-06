// @title           Polyfy Auth Service API
// @version         1.0
// @description     Authentication and user management service for Polyfy language learning platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8001
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"log"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/auth"
	"github.com/Raylynd6299/babel/internal/shared/config"
	"github.com/Raylynd6299/babel/internal/shared/database"
)

func main() {
	cfg := config.Load()

	// Actualizar config para auth service
	if cfg.Port == "8002" || cfg.Port == "8003" {
		cfg.Port = "8001" // Auth service port
	}

	db := database.NewConnection(cfg.DatabaseURL)

	// Migrar modelos de autenticación
	log.Println("Running auth service migration...")
	if err := migrateAuthDatabase(db); err != nil {
		log.Fatalf("Failed to migrate auth database: %v", err)
	}
	log.Println("Auth database migration completed successfully")

	authService := auth.NewService(db, cfg.JWTSecret)
	router := auth.NewRouter(authService)

	log.Printf("Auth service starting on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}

func migrateAuthDatabase(db *gorm.DB) error {
	log.Println("Checking auth database schema...")

	// Auto-migrate auth models
	if err := db.AutoMigrate(
		&auth.User{},
		&auth.RefreshToken{},
	); err != nil {
		return err
	}

	log.Println("Auth migration completed successfully")
	return nil
}
