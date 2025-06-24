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
