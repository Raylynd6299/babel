package main

import (
	"log"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/phonetic"
	"github.com/Raylynd6299/babel/internal/shared/config"
	"github.com/Raylynd6299/babel/internal/shared/database"
)

func main() {
	cfg := config.Load()

	// Actualizar config para phonetic service
	if cfg.Port == "8001" || cfg.Port == "8002" || cfg.Port == "8003" || cfg.Port == "8004" {
		cfg.Port = "8005" // Phonetic service port
	}

	db := database.NewConnection(cfg.DatabaseURL)

	// Migrar modelos de fonética
	log.Println("Running phonetic service migration...")
	if err := migratePhoneticDatabase(db); err != nil {
		log.Fatalf("Failed to migrate phonetic database: %v", err)
	}
	log.Println("Phonetic database migration completed successfully")

	phoneticService := phonetic.NewService(db, cfg.JWTSecret)
	router := phonetic.NewRouter(phoneticService)

	log.Printf("Phonetic service starting on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}

func migratePhoneticDatabase(db *gorm.DB) error {
	log.Println("Checking phonetic database schema...")

	// Auto-migrate phonetic models
	if err := db.AutoMigrate(
		&phonetic.Phoneme{},
		&phonetic.UserPhoneticProgress{},
		&phonetic.PhoneticExercise{},
		&phonetic.UserExerciseSession{},
		&phonetic.MinimalPair{},
	); err != nil {
		return err
	}

	// Seed basic phonemes for common languages if needed
	if err := seedPhonemes(db); err != nil {
		return err
	}

	log.Println("Phonetic migration completed successfully")
	return nil
}

func seedPhonemes(db *gorm.DB) error {
	// Check if phonemes already exist
	var count int64
	db.Model(&phonetic.Phoneme{}).Count(&count)
	if count > 0 {
		return nil // Already seeded
	}

	// Seed basic English phonemes (simplified set)
	englishPhonemes := []phonetic.Phoneme{
		// Vowels
		{LanguageID: 1, Symbol: "/iː/", Description: "Long 'ee' sound", Category: "vowel", Difficulty: 1},
		{LanguageID: 1, Symbol: "/ɪ/", Description: "Short 'i' sound", Category: "vowel", Difficulty: 1},
		{LanguageID: 1, Symbol: "/e/", Description: "'e' sound", Category: "vowel", Difficulty: 1},
		{LanguageID: 1, Symbol: "/æ/", Description: "'a' as in cat", Category: "vowel", Difficulty: 2},
		{LanguageID: 1, Symbol: "/ɑː/", Description: "'a' as in father", Category: "vowel", Difficulty: 2},
		{LanguageID: 1, Symbol: "/ɔː/", Description: "'o' as in thought", Category: "vowel", Difficulty: 2},
		{LanguageID: 1, Symbol: "/ʊ/", Description: "'u' as in book", Category: "vowel", Difficulty: 2},
		{LanguageID: 1, Symbol: "/uː/", Description: "Long 'oo' sound", Category: "vowel", Difficulty: 1},

		// Consonants
		{LanguageID: 1, Symbol: "/θ/", Description: "'th' as in think", Category: "consonant", Difficulty: 4},
		{LanguageID: 1, Symbol: "/ð/", Description: "'th' as in this", Category: "consonant", Difficulty: 4},
		{LanguageID: 1, Symbol: "/ʃ/", Description: "'sh' sound", Category: "consonant", Difficulty: 3},
		{LanguageID: 1, Symbol: "/ʒ/", Description: "'s' as in pleasure", Category: "consonant", Difficulty: 4},
		{LanguageID: 1, Symbol: "/tʃ/", Description: "'ch' sound", Category: "consonant", Difficulty: 2},
		{LanguageID: 1, Symbol: "/dʒ/", Description: "'j' sound", Category: "consonant", Difficulty: 2},
		{LanguageID: 1, Symbol: "/r/", Description: "'r' sound", Category: "consonant", Difficulty: 3},
		{LanguageID: 1, Symbol: "/l/", Description: "'l' sound", Category: "consonant", Difficulty: 2},
	}

	for _, phoneme := range englishPhonemes {
		if err := db.Create(&phoneme).Error; err != nil {
			log.Printf("Error creating phoneme %s: %v", phoneme.Symbol, err)
		}
	}

	log.Println("Seeded basic English phonemes")
	return nil
}
