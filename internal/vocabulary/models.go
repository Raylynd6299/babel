// internal/vocabulary/models.go
package vocabulary

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Vocabulary struct {
	ID                    string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Word                  string    `json:"word" gorm:"not null"`
	LanguageID            int       `json:"language_id" gorm:"not null"`
	Translation           string    `json:"translation"`
	PhoneticTranscription string    `json:"phonetic_transcription"`
	Definition            string    `json:"definition"`
	ExampleSentence       string    `json:"example_sentence"`
	FrequencyRank         int       `json:"frequency_rank"`
	DifficultyLevel       string    `json:"difficulty_level"`
	CreatedBy             string    `json:"created_by"`
	CreatedAt             time.Time `json:"created_at"`
}

type UserVocabulary struct {
	ID              string     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID          string     `json:"user_id" gorm:"not null"`
	VocabularyID    string     `json:"vocabulary_id" gorm:"not null"`
	AddedAt         time.Time  `json:"added_at" gorm:"default:CURRENT_TIMESTAMP"`
	ContextSentence string     `json:"context_sentence"`
	PersonalNote    string     `json:"personal_note"`
	SourceContentID *string    `json:"source_content_id"`
	MasteryLevel    int        `json:"mastery_level" gorm:"default:0;check:mastery_level >= 0 AND mastery_level <= 10"`
	NextReviewAt    *time.Time `json:"next_review_at"`
	ReviewCount     int        `json:"review_count" gorm:"default:0"`
	CorrectCount    int        `json:"correct_count" gorm:"default:0"`
	LastReviewedAt  *time.Time `json:"last_reviewed_at"`
	EaseFactor      float64    `json:"ease_factor" gorm:"default:2.50"`
	IntervalDays    int        `json:"interval_days" gorm:"default:1"`

	// Relations
	Vocabulary Vocabulary `json:"vocabulary,omitempty" gorm:"foreignKey:VocabularyID"`
}

type UserSRSConfig struct {
	ID               string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID           string         `json:"user_id" gorm:"not null;uniqueIndex"`
	EasyBonus        float64        `json:"easy_bonus" gorm:"default:1.3"`
	HardPenalty      float64        `json:"hard_penalty" gorm:"default:0.85"`
	FailurePenalty   float64        `json:"failure_penalty" gorm:"default:0.2"`
	MinEaseFactor    float64        `json:"min_ease_factor" gorm:"default:1.3"`
	MaxEaseFactor    float64        `json:"max_ease_factor" gorm:"default:2.5"`
	GraduationSteps  string         `json:"-" gorm:"default:'1,6'"` // Stored as comma-separated string
	NewWordsPerDay   int            `json:"new_words_per_day" gorm:"default:20"`
	MaxReviewsPerDay int            `json:"max_reviews_per_day" gorm:"default:200"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// UpdateSRSConfigRequest represents the request to update SRS config
type UpdateSRSConfigRequest struct {
	EasyBonus        float64 `json:"easy_bonus" validate:"min=1.0,max=3.0"`
	HardPenalty      float64 `json:"hard_penalty" validate:"min=0.1,max=1.0"`
	FailurePenalty   float64 `json:"failure_penalty" validate:"min=0.1,max=1.0"`
	MinEaseFactor    float64 `json:"min_ease_factor" validate:"min=1.0,max=3.0"`
	MaxEaseFactor    float64 `json:"max_ease_factor" validate:"min=1.0,max=5.0"`
	GraduationSteps  []int   `json:"graduation_steps" validate:"min=1,max=10,dive,min=1,max=30"`
	NewWordsPerDay   int     `json:"new_words_per_day" validate:"min=1,max=100"`
	MaxReviewsPerDay int     `json:"max_reviews_per_day" validate:"min=10,max=1000"`
}

type ReviewRequest struct {
	VocabularyID string `json:"vocabulary_id" validate:"required"`
	Correct      bool   `json:"correct"`
	ResponseTime int    `json:"response_time"` // milliseconds
}

// SRSConfigResponse represents the response with current config and presets
type SRSConfigResponse struct {
	CurrentConfig SRSConfig         `json:"current_config"`
	Presets       []SRSConfigPreset `json:"presets"`
	Statistics    SRSStatistics     `json:"statistics"`
}

// SRSConfigPreset represents predefined SRS configurations
type SRSConfigPreset struct {
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	EasyBonus        float64 `json:"easy_bonus"`
	HardPenalty      float64 `json:"hard_penalty"`
	FailurePenalty   float64 `json:"failure_penalty"`
	MinEaseFactor    float64 `json:"min_ease_factor"`
	MaxEaseFactor    float64 `json:"max_ease_factor"`
	GraduationSteps  []int   `json:"graduation_steps"`
	NewWordsPerDay   int     `json:"new_words_per_day"`
	MaxReviewsPerDay int     `json:"max_reviews_per_day"`
}

// SRSStatistics represents statistics about the user's SRS performance
type SRSStatistics struct {
	AverageRetention float64 `json:"average_retention"`
	AverageInterval  float64 `json:"average_interval"`
	TotalReviews     int64   `json:"total_reviews"`
	StreakDays       int     `json:"streak_days"`
	ReviewsPerDay    float64 `json:"reviews_per_day"`
	TimeToMaturity   float64 `json:"time_to_maturity_days"`
}

// Update the original SRSConfig to include new fields
type SRSConfig struct {
	EasyBonus        float64 `json:"easy_bonus"`
	HardPenalty      float64 `json:"hard_penalty"`
	FailurePenalty   float64 `json:"failure_penalty"`
	MinEaseFactor    float64 `json:"min_ease_factor"`
	MaxEaseFactor    float64 `json:"max_ease_factor"`
	GraduationSteps  []int   `json:"graduation_steps"`
	NewWordsPerDay   int     `json:"new_words_per_day"`   // NEW
	MaxReviewsPerDay int     `json:"max_reviews_per_day"` // NEW
}
type VocabularyStats struct {
	TotalWords    int64   `json:"total_words"`
	ReviewsDue    int64   `json:"reviews_due"`
	NewWords      int64   `json:"new_words"`
	LearningWords int64   `json:"learning_words"`
	MatureWords   int64   `json:"mature_words"`
	ReviewsToday  int64   `json:"reviews_today"`
	AccuracyRate  float64 `json:"accuracy_rate"`
}

// Request
type AddVocabularyRequest struct {
	Word                  string `json:"word" validate:"required,min=1,max=255"`
	Translation           string `json:"translation" validate:"required,min=1,max=255"`
	PhoneticTranscription string `json:"phonetic_transcription"`
	Definition            string `json:"definition"`
	ExampleSentence       string `json:"example_sentence"`
	ContextSentence       string `json:"context_sentence"`
	PersonalNote          string `json:"personal_note"`
	SourceContentID       string `json:"source_content_id"`
	DifficultyLevel       string `json:"difficulty_level"`
}

type UpdateVocabularyRequest struct {
	Translation           string `json:"translation" validate:"omitempty,min=1,max=255"`
	PhoneticTranscription string `json:"phonetic_transcription" validate:"omitempty,max=255"`
	Definition            string `json:"definition" validate:"omitempty"`
	ExampleSentence       string `json:"example_sentence" validate:"omitempty"`
	ContextSentence       string `json:"context_sentence" validate:"omitempty"`
	PersonalNote          string `json:"personal_note" validate:"omitempty"`
}

type BatchReviewRequest struct {
	Reviews []ReviewRequest `json:"reviews" validate:"required,min=1,max=50"`
}

type BatchReviewResult struct {
	Processed int                      `json:"processed"`
	Results   []ReviewVocabularyResult `json:"results"`
	Errors    []string                 `json:"errors,omitempty"`
}

type ReviewVocabularyResult struct {
	VocabularyID string `json:"vocabulary_id"`
	Success      bool   `json:"success"`
	NextReview   string `json:"next_review"`
	Error        string `json:"error,omitempty"`
}

type VocabularyFilter struct {
	LanguageID    int    `json:"language_id" validate:"required"`
	MasteryLevels []int  `json:"mastery_levels"`
	SearchQuery   string `json:"search_query"`
	SortBy        string `json:"sort_by"`        // added_at, mastery_level, word, next_review
	SortDirection string `json:"sort_direction"` // asc, desc
	Limit         int    `json:"limit" validate:"min=1,max=100"`
	Offset        int    `json:"offset" validate:"min=0"`
}

type ImportVocabularyRequest struct {
	LanguageID int           `json:"language_id" validate:"required"`
	Format     string        `json:"format" validate:"required,oneof=csv json anki"`
	Data       string        `json:"data" validate:"required"`
	Options    ImportOptions `json:"options"`
}

type ImportOptions struct {
	SkipDuplicates bool `json:"skip_duplicates"`
	UpdateExisting bool `json:"update_existing"`
}

type ImportResult struct {
	Total    int      `json:"total"`
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Updated  int      `json:"updated"`
	Errors   []string `json:"errors,omitempty"`
}

type VocabularyProgress struct {
	Period           string                    `json:"period"`
	TotalWords       int64                     `json:"total_words"`
	WordsLearned     int64                     `json:"words_learned"`
	WordsReviewed    int64                     `json:"words_reviewed"`
	AccuracyRate     float64                   `json:"accuracy_rate"`
	DailyProgress    []DailyVocabularyProgress `json:"daily_progress"`
	MasteryBreakdown map[string]int64          `json:"mastery_breakdown"`
}

type DailyVocabularyProgress struct {
	Date          string  `json:"date"`
	WordsAdded    int     `json:"words_added"`
	WordsReviewed int     `json:"words_reviewed"`
	AccuracyRate  float64 `json:"accuracy_rate"`
}

// GetGraduationSteps converts the comma-separated string to int slice
func (u *UserSRSConfig) GetGraduationSteps() []int {
	if u.GraduationSteps == "" {
		return []int{1, 6}
	}

	parts := strings.Split(u.GraduationSteps, ",")
	steps := make([]int, 0, len(parts))

	for _, part := range parts {
		if step, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			steps = append(steps, step)
		}
	}

	if len(steps) == 0 {
		return []int{1, 6}
	}

	return steps
}

// SetGraduationSteps converts int slice to comma-separated string
func (u *UserSRSConfig) SetGraduationSteps(steps []int) {
	if len(steps) == 0 {
		u.GraduationSteps = "1,6"
		return
	}

	strSteps := make([]string, len(steps))
	for i, step := range steps {
		strSteps[i] = strconv.Itoa(step)
	}

	u.GraduationSteps = strings.Join(strSteps, ",")
}

// ToSRSConfig converts UserSRSConfig to SRSConfig
func (u *UserSRSConfig) ToSRSConfig() SRSConfig {
	return SRSConfig{
		EasyBonus:        u.EasyBonus,
		HardPenalty:      u.HardPenalty,
		FailurePenalty:   u.FailurePenalty,
		MinEaseFactor:    u.MinEaseFactor,
		MaxEaseFactor:    u.MaxEaseFactor,
		GraduationSteps:  u.GetGraduationSteps(),
		NewWordsPerDay:   u.NewWordsPerDay,
		MaxReviewsPerDay: u.MaxReviewsPerDay,
	}
}

// Validate validates the SRS config request
func (r *UpdateSRSConfigRequest) Validate() error {
	if r.MinEaseFactor >= r.MaxEaseFactor {
		return errors.New("min_ease_factor must be less than max_ease_factor")
	}

	if len(r.GraduationSteps) < 1 {
		return errors.New("graduation_steps must have at least one step")
	}

	// Check graduation steps are in ascending order
	for i := 1; i < len(r.GraduationSteps); i++ {
		if r.GraduationSteps[i] <= r.GraduationSteps[i-1] {
			return errors.New("graduation_steps must be in ascending order")
		}
	}

	return nil
}
