package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port        string
	Environment string

	// Database
	DatabaseURL string
	RedisURL    string
	InfluxDBURL string

	// Services
	AuthServiceURL       string
	ContentServiceURL    string
	ProgressServiceURL   string
	VocabularyServiceURL string
	PhoneticServiceURL   string
	SocialServiceURL     string
	GameServiceURL       string
	AnalyticsServiceURL  string

	// Security
	JWTSecret          string
	JWTExpiration      int
	PasswordSaltRounds int

	// File Storage
	MinIOEndpoint   string
	MinIOAccessKey  string
	MinIOSecretKey  string
	MinIOBucketName string

	// Rate Limiting
	RateLimit RateLimitSettings

	// Email
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string

	// External APIs
	InfluxDBToken  string
	InfluxDBOrg    string
	InfluxDBBucket string
}

type RateLimitSettings struct {
	RequestsPerSecond int
	Burst             int
}

func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{
		Port:        getEnvOrDefault("PORT", "8080"),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),

		DatabaseURL: getEnvOrDefault("DATABASE_URL", "postgres://postgres:password@localhost:5432/language_learning?sslmode=disable"),
		RedisURL:    getEnvOrDefault("REDIS_URL", "redis://localhost:6379"),
		InfluxDBURL: getEnvOrDefault("INFLUXDB_URL", "http://localhost:8086"),

		AuthServiceURL:       getEnvOrDefault("AUTH_SERVICE_URL", "http://localhost:8001"),
		ContentServiceURL:    getEnvOrDefault("CONTENT_SERVICE_URL", "http://localhost:8002"),
		ProgressServiceURL:   getEnvOrDefault("PROGRESS_SERVICE_URL", "http://localhost:8003"),
		VocabularyServiceURL: getEnvOrDefault("VOCABULARY_SERVICE_URL", "http://localhost:8004"),
		PhoneticServiceURL:   getEnvOrDefault("PHONETIC_SERVICE_URL", "http://localhost:8005"),
		SocialServiceURL:     getEnvOrDefault("SOCIAL_SERVICE_URL", "http://localhost:8006"),
		GameServiceURL:       getEnvOrDefault("GAME_SERVICE_URL", "http://localhost:8007"),
		AnalyticsServiceURL:  getEnvOrDefault("ANALYTICS_SERVICE_URL", "http://localhost:8008"),

		JWTSecret:          getEnvOrDefault("JWT_SECRET", "your-super-secret-jwt-key"),
		JWTExpiration:      getEnvIntOrDefault("JWT_EXPIRATION", 3600),
		PasswordSaltRounds: getEnvIntOrDefault("PASSWORD_SALT_ROUNDS", 12),

		MinIOEndpoint:   getEnvOrDefault("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:  getEnvOrDefault("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:  getEnvOrDefault("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucketName: getEnvOrDefault("MINIO_BUCKET_NAME", "language-learning"),

		RateLimit: RateLimitSettings{
			RequestsPerSecond: getEnvIntOrDefault("RATE_LIMIT_RPS", 100),
			Burst:             getEnvIntOrDefault("RATE_LIMIT_BURST", 200),
		},

		SMTPHost:     getEnvOrDefault("SMTP_HOST", ""),
		SMTPPort:     getEnvIntOrDefault("SMTP_PORT", 587),
		SMTPUsername: getEnvOrDefault("SMTP_USERNAME", ""),
		SMTPPassword: getEnvOrDefault("SMTP_PASSWORD", ""),

		InfluxDBToken:  getEnvOrDefault("INFLUXDB_TOKEN", ""),
		InfluxDBOrg:    getEnvOrDefault("INFLUXDB_ORG", "language-learning"),
		InfluxDBBucket: getEnvOrDefault("INFLUXDB_BUCKET", "metrics"),
	}

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
