package phonetic

import (
	"time"

	"gorm.io/gorm"

	content "github.com/Raylynd6299/babel/internal/content"
)

// Phoneme represents a sound unit in a language
type Phoneme struct {
	ID                   int            `json:"id" gorm:"primary_key"`
	LanguageID           int            `json:"language_id" gorm:"not null"`
	Symbol               string         `json:"symbol" gorm:"not null"` // IPA symbol
	Description          string         `json:"description"`
	Category             string         `json:"category"` // vowel, consonant, diphthong
	PlaceOfArticulation  string         `json:"place_of_articulation"`
	MannerOfArticulation string         `json:"manner_of_articulation"`
	Voicing              string         `json:"voicing"` // voiced, voiceless
	AudioURL             string         `json:"audio_url"`
	DiagramURL           string         `json:"diagram_url"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Language content.Language `json:"language,omitempty" gorm:"foreignKey:LanguageID"`
}

// UserPhoneticProgress tracks user's phonetic learning progress
type UserPhoneticProgress struct {
	ID                  string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID              string         `json:"user_id" gorm:"not null"`
	PhonemeID           int            `json:"phoneme_id" gorm:"not null"`
	DiscriminationScore int            `json:"discrimination_score" gorm:"default:0;check:discrimination_score >= 0 AND discrimination_score <= 100"`
	ProductionScore     int            `json:"production_score" gorm:"default:0;check:production_score >= 0 AND production_score <= 100"`
	LastPracticedAt     *time.Time     `json:"last_practiced_at"`
	PracticeCount       int            `json:"practice_count" gorm:"default:0"`
	MasteryLevel        int            `json:"mastery_level" gorm:"default:0;check:mastery_level >= 0 AND mastery_level <= 5"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Phoneme Phoneme `json:"phoneme,omitempty" gorm:"foreignKey:PhonemeID"`
}

// PhoneticExercise represents different types of phonetic exercises
type PhoneticExercise struct {
	ID           string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PhonemeID    int            `json:"phoneme_id" gorm:"not null"`
	Type         string         `json:"type" gorm:"not null"` // discrimination, production, minimal_pairs
	Title        string         `json:"title" gorm:"not null"`
	Description  string         `json:"description"`
	Difficulty   int            `json:"difficulty" gorm:"check:difficulty >= 1 AND difficulty <= 5"`
	AudioURL     string         `json:"audio_url"`
	Instructions string         `json:"instructions"`
	Data         string         `json:"data" gorm:"type:text"` // JSON data for exercise content
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Phoneme Phoneme `json:"phoneme,omitempty" gorm:"foreignKey:PhonemeID"`
}

// MinimalPair represents word pairs that differ by one phoneme
type MinimalPair struct {
	ID           string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LanguageID   int            `json:"language_id" gorm:"not null"`
	Phoneme1ID   int            `json:"phoneme1_id" gorm:"not null"`
	Phoneme2ID   int            `json:"phoneme2_id" gorm:"not null"`
	Word1        string         `json:"word1" gorm:"not null"`
	Word2        string         `json:"word2" gorm:"not null"`
	Translation1 string         `json:"translation1"`
	Translation2 string         `json:"translation2"`
	Audio1URL    string         `json:"audio1_url"`
	Audio2URL    string         `json:"audio2_url"`
	Difficulty   int            `json:"difficulty" gorm:"check:difficulty >= 1 AND difficulty <= 5"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Language content.Language `json:"language,omitempty" gorm:"foreignKey:LanguageID"`
	Phoneme1 Phoneme          `json:"phoneme1,omitempty" gorm:"foreignKey:Phoneme1ID"`
	Phoneme2 Phoneme          `json:"phoneme2,omitempty" gorm:"foreignKey:Phoneme2ID"`
}

// UserExerciseSession tracks exercise practice sessions
type UserExerciseSession struct {
	ID          string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      string         `json:"user_id" gorm:"not null"`
	ExerciseID  string         `json:"exercise_id" gorm:"not null"`
	StartedAt   time.Time      `json:"started_at" gorm:"default:CURRENT_TIMESTAMP"`
	CompletedAt *time.Time     `json:"completed_at"`
	Score       int            `json:"score" gorm:"check:score >= 0 AND score <= 100"`
	Accuracy    float64        `json:"accuracy" gorm:"check:accuracy >= 0 AND accuracy <= 100"`
	TimeSpent   int            `json:"time_spent"` // seconds
	Attempts    int            `json:"attempts" gorm:"default:1"`
	Responses   string         `json:"responses" gorm:"type:text"` // JSON with detailed responses
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Exercise PhoneticExercise `json:"exercise,omitempty" gorm:"foreignKey:ExerciseID"`
}

// Request/Response models
type PracticePhonemeRequest struct {
	PhonemeID int     `json:"phoneme_id" validate:"required"`
	Type      string  `json:"type" validate:"required,oneof=discrimination production"`
	Score     int     `json:"score" validate:"min=0,max=100"`
	TimeSpent int     `json:"time_spent" validate:"min=1"` // seconds
	Attempts  int     `json:"attempts" validate:"min=1"`
	Accuracy  float64 `json:"accuracy" validate:"min=0,max=100"`
}

type PhoneticProgressResponse struct {
	PhonemeID           int        `json:"phoneme_id"`
	Symbol              string     `json:"symbol"`
	DiscriminationScore int        `json:"discrimination_score"`
	ProductionScore     int        `json:"production_score"`
	MasteryLevel        int        `json:"mastery_level"`
	PracticeCount       int        `json:"practice_count"`
	LastPracticedAt     *time.Time `json:"last_practiced_at"`
	RecommendedNext     bool       `json:"recommended_next"`
}

type ExerciseStartRequest struct {
	ExerciseID string `json:"exercise_id" validate:"required"`
}

type ExerciseCompleteRequest struct {
	SessionID string  `json:"session_id" validate:"required"`
	Score     int     `json:"score" validate:"min=0,max=100"`
	Accuracy  float64 `json:"accuracy" validate:"min=0,max=100"`
	TimeSpent int     `json:"time_spent" validate:"min=1"`
	Responses string  `json:"responses"` // JSON string with detailed responses
}

type PhoneticStatsResponse struct {
	LanguageID          int                        `json:"language_id"`
	TotalPhonemes       int                        `json:"total_phonemes"`
	PracticedPhonemes   int                        `json:"practiced_phonemes"`
	MasteredPhonemes    int                        `json:"mastered_phonemes"`
	AverageScore        float64                    `json:"average_score"`
	TotalPracticeTime   int                        `json:"total_practice_time"` // minutes
	WeakestPhonemes     []PhoneticProgressResponse `json:"weakest_phonemes"`
	RecommendedPractice []PhoneticProgressResponse `json:"recommended_practice"`
}

type ExerciseFilter struct {
	LanguageID int    `json:"language_id"`
	PhonemeID  int    `json:"phoneme_id"`
	Type       string `json:"type"`
	Difficulty []int  `json:"difficulty"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

type CreatePracticePlanRequest struct {
	LanguageID        int      `json:"language_id" validate:"required"`
	Name              string   `json:"name" validate:"required,min=1,max=100"`
	Description       string   `json:"description" validate:"max=500"`
	DurationWeeks     int      `json:"duration_weeks" validate:"required,min=1,max=52"`
	SessionsPerWeek   int      `json:"sessions_per_week" validate:"required,min=1,max=7"`
	MinutesPerSession int      `json:"minutes_per_session" validate:"required,min=5,max=120"`
	FocusAreas        []string `json:"focus_areas" validate:"required,min=1"`
}

type PracticePlan struct {
	ID                string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID            string         `json:"user_id" gorm:"not null"`
	LanguageID        int            `json:"language_id" gorm:"not null"`
	Name              string         `json:"name" gorm:"not null"`
	Description       string         `json:"description"`
	DurationWeeks     int            `json:"duration_weeks" gorm:"not null"`
	SessionsPerWeek   int            `json:"sessions_per_week" gorm:"not null"`
	MinutesPerSession int            `json:"minutes_per_session" gorm:"not null"`
	FocusAreas        string         `json:"focus_areas" gorm:"type:text"` // JSON array
	IsActive          bool           `json:"is_active" gorm:"default:true"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
