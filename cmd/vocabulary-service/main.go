package main

import (
	"log"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/shared/config"
	"github.com/Raylynd6299/babel/internal/shared/database"
	"github.com/Raylynd6299/babel/internal/vocabulary"
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
		&vocabulary.UserSRSConfig{},
		&vocabulary.VocabularyList{},
		&vocabulary.VocabularyListItem{},
	); err != nil {
		return err
	}

	log.Println("Vocabulary migration completed successfully")
	return nil
}
