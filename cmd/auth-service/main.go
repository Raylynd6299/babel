package main

import (
	"log"

	"github.com/Raylynd6299/babel/internal/auth"
	"github.com/Raylynd6299/babel/internal/shared/config"
	"github.com/Raylynd6299/babel/internal/shared/database"
)

func main() {
	cfg := config.Load()
	db := database.NewConnection(cfg.DatabaseURL)

	// Auto-migrate
	log.Println("Runnig auto-migration...")
	if err := db.AutoMigrate(&auth.User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Auto migration completed")

	authService := auth.NewService(db, cfg.JWTSecret)
	router := auth.NewRouter(authService)

	log.Printf("Auth service starting on Port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}
