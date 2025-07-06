// @title           Polyfy Content Service API
// @version         1.0
// @description     Content management service for Polyfy language learning platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8002
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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

	// Actualizar config para content service
	if cfg.Port == "8001" || cfg.Port == "8003" {
		cfg.Port = "8002" // Content service port
	}

	db := database.NewConnection(cfg.DatabaseURL)

	// Migrar modelos de contenido
	log.Println("Running content service migration...")
	if err := migrateContentDatabase(db); err != nil {
		log.Fatalf("Failed to migrate content database: %v", err)
	}
	log.Println("Content database migration completed successfully")

	// Seed initial data
	contentService := content.NewService(db, cfg.JWTSecret)
	if err := seedInitialData(contentService); err != nil {
		log.Printf("Warning: Failed to seed initial data: %v", err)
	}

	router := content.NewRouter(contentService)

	log.Printf("Content service starting on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}

func migrateContentDatabase(db *gorm.DB) error {
	log.Println("Checking content database schema...")

	// Auto-migrate content models
	if err := db.AutoMigrate(
		&content.Language{},
		&content.Content{},
		&content.ContentEpisode{},
		&content.ContentRating{},
	); err != nil {
		return err
	}

	log.Println("Content migration completed successfully")
	return nil
}

func seedInitialData(service *content.Service) error {
	log.Println("Seeding initial content data...")

	// Seed languages
	languages := []content.Language{
		{Code: "en", Name: "English", NativeName: "English"},
		{Code: "es", Name: "Spanish", NativeName: "Español"},
		{Code: "fr", Name: "French", NativeName: "Français"},
		{Code: "de", Name: "German", NativeName: "Deutsch"},
		{Code: "it", Name: "Italian", NativeName: "Italiano"},
		{Code: "pt", Name: "Portuguese", NativeName: "Português"},
		{Code: "ja", Name: "Japanese", NativeName: "日本語"},
		{Code: "ko", Name: "Korean", NativeName: "한국어"},
		{Code: "zh", Name: "Chinese", NativeName: "中文"},
		{Code: "ru", Name: "Russian", NativeName: "Русский"},
	}

	for _, lang := range languages {
		var existing content.Language
		err := service.GetDB().Where("code = ?", lang.Code).First(&existing).Error
		if err != nil {
			if err := service.GetDB().Create(&lang).Error; err != nil {
				log.Printf("Failed to create language %s: %v", lang.Code, err)
			}
		}
	}

	log.Println("Initial content data seeded successfully")
	return nil
}
