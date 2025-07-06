// @title           Polyfy Vocabulary Service API
// @version         1.0
// @description     Vocabulary management and SRS (Spaced Repetition System) service for Polyfy language learning platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8004
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"log"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/vocabulary"
	"github.com/Raylynd6299/babel/internal/shared/config"
	"github.com/Raylynd6299/babel/internal/shared/database"
)

func main() {
	cfg := config.Load()

	// Actualizar config para vocabulary service
	if cfg.Port == "8001" || cfg.Port == "8002" || cfg.Port == "8003" {
		cfg.Port = "8004" // Vocabulary service port
	}

	db := database.NewConnection(cfg.DatabaseURL)

	// Migrar modelos de vocabulario
	log.Println("Running vocabulary service migration...")
	if err := migrateVocabularyDatabase(db); err != nil {
		log.Fatalf("Failed to migrate vocabulary database: %v", err)
	}
	log.Println("Vocabulary database migration completed successfully")

	vocabularyService := vocabulary.NewService(db, cfg.JWTSecret)

	router := vocabulary.NewRouter(vocabularyService)

	log.Printf("Vocabulary service starting on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}

func migrateVocabularyDatabase(db *gorm.DB) error {
	log.Println("Checking vocabulary database schema...")

	// Auto-migrate vocabulary models
	if err := db.AutoMigrate(
		&vocabulary.Vocabulary{},
		&vocabulary.UserVocabulary{},
		&vocabulary.VocabularyList{},
		&vocabulary.UserSRSConfig{},
	); err != nil {
		return err
	}

	log.Println("Vocabulary migration completed successfully")
	return nil
}
