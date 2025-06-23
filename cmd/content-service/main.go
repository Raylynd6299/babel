package main

import (
	"log"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/content"
	"github.com/Raylynd6299/babel/internal/shared/config"
	"github.com/Raylynd6299/babel/internal/shared/database"
)

func main() {
	cfg := config.Load()

	if cfg.Port == "8001" {
		cfg.Port = "8002"
	}

	db := database.NewConnection(cfg.DatabaseURL)

	// Auto migrate contente
	log.Println("Running content service migration...")
	if err := migrateContentDatabase(db); err != nil {
		log.Fatalf("Failed to migrate content database: %v", err)
	}
	log.Println("Content database migration completed successfully")

	contentService := content.NewService(db, cfg.JWTSecret)
	router := content.NewRouter(contentService)

	log.Printf("Content service starting on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}

func migrateContentDatabase(db *gorm.DB) error {
	// Auto-migrate content models
	err := db.AutoMigrate(
		&content.Language{},
		&content.Content{},
		&content.ContentEpisode{},
		&content.ContentRating{},
	)

	if err != nil {
		log.Fatal("warning: error migration")
		return err
	}

	return seedLanguages(db)
}

func seedLanguages(db *gorm.DB) error {
	languages := []content.Language{
		{Code: "en", Name: "English", NativeName: "English", IsActive: true},
		{Code: "es", Name: "Spanish", NativeName: "Español", IsActive: true},
		{Code: "fr", Name: "French", NativeName: "Français", IsActive: true},
		{Code: "de", Name: "German", NativeName: "Deutsch", IsActive: true},
		{Code: "it", Name: "Italian", NativeName: "Italiano", IsActive: true},
		{Code: "pt", Name: "Portuguese", NativeName: "Português", IsActive: true},
		{Code: "ja", Name: "Japanese", NativeName: "日本語", IsActive: true},
		{Code: "ko", Name: "Korean", NativeName: "한국어", IsActive: true},
		{Code: "zh", Name: "Chinese", NativeName: "中文", IsActive: true},
		{Code: "ru", Name: "Russian", NativeName: "Русский", IsActive: true},
	}

	for _, lang := range languages {
		var existingLang content.Language
		err := db.Where("code = ?", lang.Code).First(&existingLang).Error
		if err != nil {
			// No existe, crear
			if err := db.Create(&lang).Error; err != nil {
				return err
			}
			log.Printf("Created language: %s", lang.Name)
		}
	}

	return nil
}
