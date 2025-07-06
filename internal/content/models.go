// internal/content/models.go
package content

import (
	"time"

	"gorm.io/gorm"
)

type Content struct {
	ID                     string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title                  string         `json:"title" gorm:"not null"`
	ContentType            string         `json:"content_type" gorm:"not null"` // series, movie, podcast, book
	LanguageID             int            `json:"language_id" gorm:"not null"`
	TotalEpisodes          int            `json:"total_episodes" gorm:"default:1"`
	AverageEpisodeDuration int            `json:"average_episode_duration"` // minutes
	YearReleased           int            `json:"year_released"`
	Country                string         `json:"country"`
	Genre                  string         `json:"genre"`
	Description            string         `json:"description"`
	PosterURL              string         `json:"poster_url"`
	IMDbRating             float32        `json:"imdb_rating"`
	DifficultyLevel        string         `json:"difficulty_level"`
	CreatedBy              string         `json:"created_by"`
	CreatedAt              time.Time      `json:"created_at"`
	DeletedAt              gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggerignore:"true"`
	UpdatedAt              time.Time      `json:"updated_at"`
	IsVerified             bool           `json:"is_verified" gorm:"default:false"`
	ViewCount              int            `json:"view_count" gorm:"default:0"`

	// Relations
	Episodes []ContentEpisode `json:"episodes,omitempty" gorm:"foreignKey:ContentID"`
	Language Language         `json:"language,omitempty" gorm:"foreignKey:LanguageID"`
	Ratings  []ContentRating  `json:"ratings,omitempty" gorm:"foreignKey:ContentID"`
}

type ContentEpisode struct {
	ID              string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ContentID       string         `json:"content_id" gorm:"not null"`
	EpisodeNumber   int            `json:"episode_number" gorm:"not null"`
	Title           string         `json:"title"`
	DurationMinutes int            `json:"duration_minutes"`
	SeasonNumber    int            `json:"season_number" gorm:"default:1"`
	Description     string         `json:"description"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

type Language struct {
	ID               int            `json:"id" gorm:"primary_key"`
	Code             string         `json:"code" gorm:"uniqueIndex;not null"`
	Name             string         `json:"name" gorm:"not null"`
	NativeName       string         `json:"native_name"`
	IsActive         bool           `json:"is_active" gorm:"default:true"`
	PhoneticAlphabet string         `json:"phonetic_alphabet"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

type ContentRating struct {
	ID                  string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID              string         `json:"user_id" gorm:"not null"`
	ContentID           string         `json:"content_id" gorm:"not null"`
	DifficultyRating    int            `json:"difficulty_rating" gorm:"check:difficulty_rating >= 1 AND difficulty_rating <= 5"`
	UsefulnessRating    int            `json:"usefulness_rating" gorm:"check:usefulness_rating >= 1 AND usefulness_rating <= 5"`
	EntertainmentRating int            `json:"entertainment_rating" gorm:"check:entertainment_rating >= 1 AND entertainment_rating <= 5"`
	ReviewText          string         `json:"review_text"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

type CreateContentRequest struct {
	Title                  string  `json:"title" validate:"required,min=1,max=255"`
	ContentType            string  `json:"content_type" validate:"required,oneof=series movie podcast book"`
	LanguageID             int     `json:"language_id" validate:"required"`
	TotalEpisodes          int     `json:"total_episodes" validate:"min=1"`
	AverageEpisodeDuration int     `json:"average_episode_duration" validate:"min=1"`
	YearReleased           int     `json:"year_released" validate:"min=1900,max=2030"`
	Country                string  `json:"country" validate:"max=100"`
	Genre                  string  `json:"genre" validate:"max=100"`
	Description            string  `json:"description"`
	PosterURL              string  `json:"poster_url" validate:"url"`
	IMDbRating             float32 `json:"imdb_rating" validate:"min=0,max=10"`
}

type ContentFilter struct {
	LanguageID    int      `json:"language_id"`
	ContentType   string   `json:"content_type"`
	Genre         string   `json:"genre"`
	Country       string   `json:"country"`
	MinRating     float32  `json:"min_rating"`
	MaxRating     float32  `json:"max_rating"`
	Difficulty    []string `json:"difficulty"`
	YearFrom      int      `json:"year_from"`
	YearTo        int      `json:"year_to"`
	Search        string   `json:"search"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
	SortBy        string   `json:"sort_by"`        // title, year, rating, difficulty, created_at
	SortDirection string   `json:"sort_direction"` // asc, desc
}

type RateContentRequest struct {
	DifficultyRating    int    `json:"difficulty_rating" validate:"required,min=1,max=5"`
	UsefulnessRating    int    `json:"usefulness_rating" validate:"required,min=1,max=5"`
	EntertainmentRating int    `json:"entertainment_rating" validate:"required,min=1,max=5"`
	ReviewText          string `json:"review_text" validate:"max=1000"`
}

type CreateEpisodeRequest struct {
	EpisodeNumber   int    `json:"episode_number" validate:"required,min=1"`
	Title           string `json:"title" validate:"max=255"`
	DurationMinutes int    `json:"duration_minutes" validate:"required,min=1"`
	SeasonNumber    int    `json:"season_number" validate:"min=1"`
	Description     string `json:"description"`
}

type UpdateContentRequest struct {
	Title                  string  `json:"title" validate:"omitempty,min=1,max=255"`
	TotalEpisodes          int     `json:"total_episodes" validate:"omitempty,min=1"`
	AverageEpisodeDuration int     `json:"average_episode_duration" validate:"omitempty,min=1"`
	YearReleased           int     `json:"year_released" validate:"omitempty,min=1900,max=2030"`
	Country                string  `json:"country" validate:"omitempty,max=100"`
	Genre                  string  `json:"genre" validate:"omitempty,max=100"`
	Description            string  `json:"description"`
	PosterURL              string  `json:"poster_url" validate:"omitempty,url"`
	IMDbRating             float32 `json:"imdb_rating" validate:"omitempty,min=0,max=10"`
}
