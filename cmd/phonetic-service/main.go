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
	if cfg.Port == "8001" || cfg.Port == "8002" || cfg.Port == "8003" {
		cfg.Port = "8005" // Phonetic service port
	}

	db := database.NewConnection(cfg.DatabaseURL)

	// Migrar modelos fonéticos
	log.Println("Running phonetic service migration...")
	if err := migratePhoneticDatabase(db); err != nil {
		log.Fatalf("Failed to migrate phonetic database: %v", err)
	}
	log.Println("Phonetic database migration completed successfully")

	// Seed initial phonetic data
	phoneticService := phonetic.NewService(db, cfg.JWTSecret)
	if err := seedInitialPhoneticData(phoneticService); err != nil {
		log.Printf("Warning: Failed to seed phonetic data: %v", err)
	}

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
		&phonetic.MinimalPair{},
		&phonetic.UserExerciseSession{},
		&phonetic.PracticePlan{},
	); err != nil {
		return err
	}

	log.Println("Phonetic migration completed successfully")
	return nil
}

func seedInitialPhoneticData(service *phonetic.Service) error {
	log.Println("Seeding initial phonetic data...")

	// Seed English phonemes
	englishPhonemes := []phonetic.Phoneme{
		// Vowels
		{Symbol: "iː", Description: "Long i sound", Category: "vowel", PlaceOfArticulation: "front", MannerOfArticulation: "close", Voicing: "voiced"},
		{Symbol: "ɪ", Description: "Short i sound", Category: "vowel", PlaceOfArticulation: "front", MannerOfArticulation: "near-close", Voicing: "voiced"},
		{Symbol: "e", Description: "E sound", Category: "vowel", PlaceOfArticulation: "front", MannerOfArticulation: "close-mid", Voicing: "voiced"},
		{Symbol: "æ", Description: "A sound as in cat", Category: "vowel", PlaceOfArticulation: "front", MannerOfArticulation: "near-open", Voicing: "voiced"},
		{Symbol: "ɑː", Description: "Long a sound", Category: "vowel", PlaceOfArticulation: "back", MannerOfArticulation: "open", Voicing: "voiced"},
		{Symbol: "ɒ", Description: "Short o sound", Category: "vowel", PlaceOfArticulation: "back", MannerOfArticulation: "open", Voicing: "voiced"},
		{Symbol: "ɔː", Description: "Long o sound", Category: "vowel", PlaceOfArticulation: "back", MannerOfArticulation: "open-mid", Voicing: "voiced"},
		{Symbol: "ʊ", Description: "Short u sound", Category: "vowel", PlaceOfArticulation: "back", MannerOfArticulation: "near-close", Voicing: "voiced"},
		{Symbol: "uː", Description: "Long u sound", Category: "vowel", PlaceOfArticulation: "back", MannerOfArticulation: "close", Voicing: "voiced"},
		{Symbol: "ʌ", Description: "Short u sound as in cup", Category: "vowel", PlaceOfArticulation: "back", MannerOfArticulation: "open-mid", Voicing: "voiced"},
		{Symbol: "ɜː", Description: "Er sound", Category: "vowel", PlaceOfArticulation: "central", MannerOfArticulation: "open-mid", Voicing: "voiced"},
		{Symbol: "ə", Description: "Schwa sound", Category: "vowel", PlaceOfArticulation: "central", MannerOfArticulation: "mid", Voicing: "voiced"},

		// Consonants
		{Symbol: "p", Description: "P sound", Category: "consonant", PlaceOfArticulation: "bilabial", MannerOfArticulation: "plosive", Voicing: "voiceless"},
		{Symbol: "b", Description: "B sound", Category: "consonant", PlaceOfArticulation: "bilabial", MannerOfArticulation: "plosive", Voicing: "voiced"},
		{Symbol: "t", Description: "T sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "plosive", Voicing: "voiceless"},
		{Symbol: "d", Description: "D sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "plosive", Voicing: "voiced"},
		{Symbol: "k", Description: "K sound", Category: "consonant", PlaceOfArticulation: "velar", MannerOfArticulation: "plosive", Voicing: "voiceless"},
		{Symbol: "g", Description: "G sound", Category: "consonant", PlaceOfArticulation: "velar", MannerOfArticulation: "plosive", Voicing: "voiced"},
		{Symbol: "f", Description: "F sound", Category: "consonant", PlaceOfArticulation: "labiodental", MannerOfArticulation: "fricative", Voicing: "voiceless"},
		{Symbol: "v", Description: "V sound", Category: "consonant", PlaceOfArticulation: "labiodental", MannerOfArticulation: "fricative", Voicing: "voiced"},
		{Symbol: "θ", Description: "Th sound as in think", Category: "consonant", PlaceOfArticulation: "dental", MannerOfArticulation: "fricative", Voicing: "voiceless"},
		{Symbol: "ð", Description: "Th sound as in this", Category: "consonant", PlaceOfArticulation: "dental", MannerOfArticulation: "fricative", Voicing: "voiced"},
		{Symbol: "s", Description: "S sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "fricative", Voicing: "voiceless"},
		{Symbol: "z", Description: "Z sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "fricative", Voicing: "voiced"},
		{Symbol: "ʃ", Description: "Sh sound", Category: "consonant", PlaceOfArticulation: "postalveolar", MannerOfArticulation: "fricative", Voicing: "voiceless"},
		{Symbol: "ʒ", Description: "Zh sound as in measure", Category: "consonant", PlaceOfArticulation: "postalveolar", MannerOfArticulation: "fricative", Voicing: "voiced"},
		{Symbol: "h", Description: "H sound", Category: "consonant", PlaceOfArticulation: "glottal", MannerOfArticulation: "fricative", Voicing: "voiceless"},
		{Symbol: "m", Description: "M sound", Category: "consonant", PlaceOfArticulation: "bilabial", MannerOfArticulation: "nasal", Voicing: "voiced"},
		{Symbol: "n", Description: "N sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "nasal", Voicing: "voiced"},
		{Symbol: "ŋ", Description: "Ng sound", Category: "consonant", PlaceOfArticulation: "velar", MannerOfArticulation: "nasal", Voicing: "voiced"},
		{Symbol: "l", Description: "L sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "lateral", Voicing: "voiced"},
		{Symbol: "r", Description: "R sound", Category: "consonant", PlaceOfArticulation: "postalveolar", MannerOfArticulation: "approximant", Voicing: "voiced"},
		{Symbol: "j", Description: "Y sound", Category: "consonant", PlaceOfArticulation: "palatal", MannerOfArticulation: "approximant", Voicing: "voiced"},
		{Symbol: "w", Description: "W sound", Category: "consonant", PlaceOfArticulation: "labio-velar", MannerOfArticulation: "approximant", Voicing: "voiced"},

		// Diphthongs
		{Symbol: "eɪ", Description: "A sound as in face", Category: "diphthong", PlaceOfArticulation: "front-front", MannerOfArticulation: "close-mid to close", Voicing: "voiced"},
		{Symbol: "aɪ", Description: "I sound as in price", Category: "diphthong", PlaceOfArticulation: "front-front", MannerOfArticulation: "open to close", Voicing: "voiced"},
		{Symbol: "ɔɪ", Description: "Oy sound as in choice", Category: "diphthong", PlaceOfArticulation: "back-front", MannerOfArticulation: "open-mid to close", Voicing: "voiced"},
		{Symbol: "aʊ", Description: "Ow sound as in mouth", Category: "diphthong", PlaceOfArticulation: "front-back", MannerOfArticulation: "open to close", Voicing: "voiced"},
		{Symbol: "əʊ", Description: "O sound as in goat", Category: "diphthong", PlaceOfArticulation: "central-back", MannerOfArticulation: "mid to close", Voicing: "voiced"},
		{Symbol: "ɪə", Description: "Ear sound", Category: "diphthong", PlaceOfArticulation: "front-central", MannerOfArticulation: "near-close to mid", Voicing: "voiced"},
		{Symbol: "eə", Description: "Air sound", Category: "diphthong", PlaceOfArticulation: "front-central", MannerOfArticulation: "close-mid to mid", Voicing: "voiced"},
		{Symbol: "ʊə", Description: "Ure sound", Category: "diphthong", PlaceOfArticulation: "back-central", MannerOfArticulation: "near-close to mid", Voicing: "voiced"},
	}

	// Seed English phonemes (language_id = 1 for English)
	if err := service.SeedPhonemes(nil, 1, englishPhonemes); err != nil {
		return err
	}

	// Seed Spanish phonemes
	spanishPhonemes := []phonetic.Phoneme{
		// Spanish vowels
		{Symbol: "a", Description: "A sound", Category: "vowel", PlaceOfArticulation: "central", MannerOfArticulation: "open", Voicing: "voiced"},
		{Symbol: "e", Description: "E sound", Category: "vowel", PlaceOfArticulation: "front", MannerOfArticulation: "close-mid", Voicing: "voiced"},
		{Symbol: "i", Description: "I sound", Category: "vowel", PlaceOfArticulation: "front", MannerOfArticulation: "close", Voicing: "voiced"},
		{Symbol: "o", Description: "O sound", Category: "vowel", PlaceOfArticulation: "back", MannerOfArticulation: "close-mid", Voicing: "voiced"},
		{Symbol: "u", Description: "U sound", Category: "vowel", PlaceOfArticulation: "back", MannerOfArticulation: "close", Voicing: "voiced"},

		// Spanish consonants
		{Symbol: "p", Description: "P sound", Category: "consonant", PlaceOfArticulation: "bilabial", MannerOfArticulation: "plosive", Voicing: "voiceless"},
		{Symbol: "b", Description: "B sound", Category: "consonant", PlaceOfArticulation: "bilabial", MannerOfArticulation: "plosive", Voicing: "voiced"},
		{Symbol: "t", Description: "T sound", Category: "consonant", PlaceOfArticulation: "dental", MannerOfArticulation: "plosive", Voicing: "voiceless"},
		{Symbol: "d", Description: "D sound", Category: "consonant", PlaceOfArticulation: "dental", MannerOfArticulation: "plosive", Voicing: "voiced"},
		{Symbol: "k", Description: "K sound", Category: "consonant", PlaceOfArticulation: "velar", MannerOfArticulation: "plosive", Voicing: "voiceless"},
		{Symbol: "g", Description: "G sound", Category: "consonant", PlaceOfArticulation: "velar", MannerOfArticulation: "plosive", Voicing: "voiced"},
		{Symbol: "f", Description: "F sound", Category: "consonant", PlaceOfArticulation: "labiodental", MannerOfArticulation: "fricative", Voicing: "voiceless"},
		{Symbol: "β", Description: "Beta sound (soft b)", Category: "consonant", PlaceOfArticulation: "bilabial", MannerOfArticulation: "fricative", Voicing: "voiced"},
		{Symbol: "θ", Description: "Theta sound (Spain)", Category: "consonant", PlaceOfArticulation: "dental", MannerOfArticulation: "fricative", Voicing: "voiceless"},
		{Symbol: "ð", Description: "Eth sound (soft d)", Category: "consonant", PlaceOfArticulation: "dental", MannerOfArticulation: "fricative", Voicing: "voiced"},
		{Symbol: "s", Description: "S sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "fricative", Voicing: "voiceless"},
		{Symbol: "x", Description: "J sound", Category: "consonant", PlaceOfArticulation: "velar", MannerOfArticulation: "fricative", Voicing: "voiceless"},
		{Symbol: "ɣ", Description: "Gamma sound (soft g)", Category: "consonant", PlaceOfArticulation: "velar", MannerOfArticulation: "fricative", Voicing: "voiced"},
		{Symbol: "tʃ", Description: "Ch sound", Category: "consonant", PlaceOfArticulation: "postalveolar", MannerOfArticulation: "affricate", Voicing: "voiceless"},
		{Symbol: "m", Description: "M sound", Category: "consonant", PlaceOfArticulation: "bilabial", MannerOfArticulation: "nasal", Voicing: "voiced"},
		{Symbol: "n", Description: "N sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "nasal", Voicing: "voiced"},
		{Symbol: "ɲ", Description: "Ñ sound", Category: "consonant", PlaceOfArticulation: "palatal", MannerOfArticulation: "nasal", Voicing: "voiced"},
		{Symbol: "l", Description: "L sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "lateral", Voicing: "voiced"},
		{Symbol: "ʎ", Description: "Ll sound", Category: "consonant", PlaceOfArticulation: "palatal", MannerOfArticulation: "lateral", Voicing: "voiced"},
		{Symbol: "r", Description: "Single R sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "tap", Voicing: "voiced"},
		{Symbol: "rr", Description: "Double R sound", Category: "consonant", PlaceOfArticulation: "alveolar", MannerOfArticulation: "trill", Voicing: "voiced"},
		{Symbol: "j", Description: "Y sound", Category: "consonant", PlaceOfArticulation: "palatal", MannerOfArticulation: "approximant", Voicing: "voiced"},
		{Symbol: "w", Description: "W sound", Category: "consonant", PlaceOfArticulation: "labio-velar", MannerOfArticulation: "approximant", Voicing: "voiced"},
	}

	// Seed Spanish phonemes (language_id = 2 for Spanish)
	if err := service.SeedPhonemes(nil, 2, spanishPhonemes); err != nil {
		return err
	}

	log.Println("Initial phonetic data seeded successfully")
	return nil
}
