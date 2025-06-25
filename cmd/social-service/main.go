package main

import (
	"log"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/shared/config"
	"github.com/Raylynd6299/babel/internal/shared/database"
	"github.com/Raylynd6299/babel/internal/social"
)

func main() {
	cfg := config.Load()

	// Actualizar config para social service
	if cfg.Port == "8001" || cfg.Port == "8002" || cfg.Port == "8003" || cfg.Port == "8005" {
		cfg.Port = "8006" // Social service port
	}

	db := database.NewConnection(cfg.DatabaseURL)

	// Migrar modelos sociales
	log.Println("Running social service migration...")
	if err := migrateSocialDatabase(db); err != nil {
		log.Fatalf("Failed to migrate social database: %v", err)
	}
	log.Println("Social database migration completed successfully")

	socialService := social.NewService(db, cfg.JWTSecret)
	router := social.NewRouter(socialService)

	log.Printf("Social service starting on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}

func migrateSocialDatabase(db *gorm.DB) error {
	log.Println("Checking social database schema...")

	// Auto-migrate social models
	if err := db.AutoMigrate(
		&social.UserProfile{},
		&social.UserFollow{},
		&social.StudyGroup{},
		&social.GroupMembership{},
		&social.ActivityFeed{},
		&social.LanguageExchange{},
		&social.Mentorship{},
		&social.UserInteraction{},
		&social.Leaderboard{},
	); err != nil {
		return err
	}

	// Add indexes for performance
	if err := addSocialIndexes(db); err != nil {
		log.Printf("Warning: Failed to add some indexes: %v", err)
	}

	log.Println("Social migration completed successfully")
	return nil
}

func addSocialIndexes(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_user_follows_follower ON user_follows(follower_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_follows_following ON user_follows(following_id)",
		"CREATE INDEX IF NOT EXISTS idx_activity_feeds_user_public ON activity_feeds(user_id, is_public)",
		"CREATE INDEX IF NOT EXISTS idx_activity_feeds_created_at ON activity_feeds(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_group_memberships_user ON group_memberships(user_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_group_memberships_group ON group_memberships(group_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_study_groups_language ON study_groups(language_id, is_public)",
		"CREATE INDEX IF NOT EXISTS idx_user_profiles_public ON user_profiles(is_public)",
		"CREATE INDEX IF NOT EXISTS idx_user_interactions_target ON user_interactions(target_type, target_id, type)",
		"CREATE INDEX IF NOT EXISTS idx_language_exchanges_users ON language_exchanges(user1_id, user2_id)",
		"CREATE INDEX IF NOT EXISTS idx_mentorships_mentor ON mentorships(mentor_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_mentorships_mentee ON mentorships(mentee_id, status)",
	}

	for _, indexSQL := range indexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			log.Printf("Failed to create index: %s - %v", indexSQL, err)
		}
	}

	return nil
}
