// @title           Polyfy Progress Service API
// @version         1.0
// @description     Progress tracking and analytics service for Polyfy language learning platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8003
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"log"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/progress"
	"github.com/Raylynd6299/babel/internal/shared/config"
	"github.com/Raylynd6299/babel/internal/shared/database"
)

func main() {
	cfg := config.Load()

	// Actualizar config para progress service
	if cfg.Port == "8001" || cfg.Port == "8002" {
		cfg.Port = "8003" // Progress service port
	}

	db := database.NewConnection(cfg.DatabaseURL)

	// Migrar modelos de progreso
	log.Println("Running progress service migration...")
	if err := migrateProgressDatabase(db); err != nil {
		log.Fatalf("Failed to migrate progress database: %v", err)
	}
	log.Println("Progress database migration completed successfully")

	progressService := progress.NewService(db, cfg.JWTSecret)

	router := progress.NewRouter(progressService)

	log.Printf("Progress service starting on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}

func migrateProgressDatabase(db *gorm.DB) error {
	log.Println("Checking progress database schema...")

	// Auto-migrate progress models
	if err := db.AutoMigrate(
		&progress.UserProgress{},
		&progress.UserStats{},
	); err != nil {
		return err
	}

	log.Println("Progress migration completed successfully")
	return nil
}
