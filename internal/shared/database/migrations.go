// internal/shared/database/migrations.go
package database

import (
	"log"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/auth"
	"github.com/Raylynd6299/babel/internal/content"
	"github.com/Raylynd6299/babel/internal/progress"
)

func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// Auto-migrate all models
	err := db.AutoMigrate(
		// Auth models
		&auth.User{},
		&content.Language{},
		// &auth.UserLanguage{},

		// Content models
		&content.Content{},
		&content.ContentEpisode{},
		&content.ContentRating{},

		// Progress models
		&progress.UserProgress{},
		&progress.UserStats{},

		// Vocabulary models
		// &vocabulary.Vocabulary{},
		// &vocabulary.UserVocabulary{},

		// Phonetic models
		// &phonetic.Phoneme{},
		// &phonetic.UserPhoneticProgress{},

		// Social models
		// &social.UserFollow{},
		// &social.Group{},
		// &social.GroupMembership{},

		// Gamification models
		// &gamification.Achievement{},
		// &gamification.UserAchievement{},
		// &gamification.Challenge{},
		// &gamification.UserChallenge{},
	)

	if err != nil {
		return err
	}

	// Seed initial data
	if err := seedInitialData(db); err != nil {
		return err
	}

	log.Println("Database migrations completed successfully")
	return nil
}

func seedInitialData(db *gorm.DB) error {
	// Seed languages
	languages := []content.Language{
		{Code: "en", Name: "English", NativeName: "English", PhoneticAlphabet: "IPA"},
		{Code: "es", Name: "Spanish", NativeName: "Español", PhoneticAlphabet: "IPA"},
		{Code: "fr", Name: "French", NativeName: "Français", PhoneticAlphabet: "IPA"},
		{Code: "de", Name: "German", NativeName: "Deutsch", PhoneticAlphabet: "IPA"},
		{Code: "it", Name: "Italian", NativeName: "Italiano", PhoneticAlphabet: "IPA"},
		{Code: "pt", Name: "Portuguese", NativeName: "Português", PhoneticAlphabet: "IPA"},
		{Code: "ja", Name: "Japanese", NativeName: "日本語", PhoneticAlphabet: "IPA"},
		{Code: "ko", Name: "Korean", NativeName: "한국어", PhoneticAlphabet: "IPA"},
		{Code: "zh", Name: "Chinese", NativeName: "中文", PhoneticAlphabet: "Pinyin"},
		{Code: "ru", Name: "Russian", NativeName: "Русский", PhoneticAlphabet: "IPA"},
	}

	for _, lang := range languages {
		var existingLang content.Language
		if err := db.Where("code = ?", lang.Code).First(&existingLang).Error; err != nil {
			if err := db.Create(&lang).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
