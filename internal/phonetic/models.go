package phonetic

import (
	"time"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/content"
)

// Phoneme represents a phonetic sound in a language
type Phoneme struct {
	ID                   int            `json:"id" gorm:"primary_key;autoIncrement"`
	LanguageID           int            `json:"language_id" gorm:"not null"`
	Symbol               string         `json:"symbol" gorm:"not null"` // IPA symbol like /θ/, /ð/
	Description          string         `json:"description"`
	Category             string         `json:"category"`               // vowel, consonant, diphthong
	PlaceOfArticulation  string         `json:"place_of_articulation"`  // bilabial, alveolar, etc.
	MannerOfArticulation string         `json:"manner_of_articulation"` // stop, fricative, etc.
	Voicing              string         `json:"voicing"`                // voiced, voiceless
	AudioURL             string         `json:"audio_url"`
	DiagramURL           string         `json:"diagram_url"`
	Examples             string         `json:"examples"`                    // JSON array of example words
	Difficulty           int            `json:"difficulty" gorm:"default:1"` // 1-5 scale
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Language content.Language       `json:"language,omitempty" gorm:"foreignKey:LanguageID"`
	Progress []UserPhoneticProgress `json:"progress,omitempty" gorm:"foreignKey:PhonemeID"`
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
	LanguageID   int            `json:"language_id" gorm:"not null"`
	PhonemeID    int            `json:"phoneme_id"`                    // Optional, for phoneme-specific exercises
	ExerciseType string         `json:"exercise_type" gorm:"not null"` // discrimination, production, minimal_pairs
	Title        string         `json:"title" gorm:"not null"`
	Description  string         `json:"description"`
	Instructions string         `json:"instructions"`
	Data         string         `json:"data"` // JSON data specific to exercise type
	Difficulty   int            `json:"difficulty" gorm:"default:1;check:difficulty >= 1 AND difficulty <= 5"`
	Duration     int            `json:"duration"` // Expected duration in seconds
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Language content.Language `json:"language,omitempty" gorm:"foreignKey:LanguageID"`
	Phoneme  *Phoneme         `json:"phoneme,omitempty" gorm:"foreignKey:PhonemeID"`
}

// UserExerciseSession tracks user's performance in exercises
type UserExerciseSession struct {
	ID          string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      string         `json:"user_id" gorm:"not null"`
	ExerciseID  string         `json:"exercise_id" gorm:"not null"`
	Score       int            `json:"score" gorm:"check:score >= 0 AND score <= 100"`
	Duration    int            `json:"duration"`  // Actual duration in seconds
	Responses   string         `json:"responses"` // JSON array of user responses
	Feedback    string         `json:"feedback"`  // JSON feedback data
	CompletedAt time.Time      `json:"completed_at"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Exercise PhoneticExercise `json:"exercise,omitempty" gorm:"foreignKey:ExerciseID"`
}

// MinimalPair represents word pairs that differ by one phoneme
type MinimalPair struct {
	ID         string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LanguageID int            `json:"language_id" gorm:"not null"`
	PhonemeID1 int            `json:"phoneme_id_1" gorm:"not null"`
	PhonemeID2 int            `json:"phoneme_id_2" gorm:"not null"`
	Word1      string         `json:"word1" gorm:"not null"`
	Word2      string         `json:"word2" gorm:"not null"`
	AudioURL1  string         `json:"audio_url_1"`
	AudioURL2  string         `json:"audio_url_2"`
	Difficulty int            `json:"difficulty" gorm:"default:1;check:difficulty >= 1 AND difficulty <= 5"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Language content.Language `json:"language,omitempty" gorm:"foreignKey:LanguageID"`
	Phoneme1 Phoneme          `json:"phoneme1,omitempty" gorm:"foreignKey:PhonemeID1"`
	Phoneme2 Phoneme          `json:"phoneme2,omitempty" gorm:"foreignKey:PhonemeID2"`
}

// Request/Response models

// PracticePhonemeRequest represents a phoneme practice session
type PracticePhonemeRequest struct {
	PhonemeID    int    `json:"phoneme_id" validate:"required"`
	ExerciseType string `json:"exercise_type" validate:"required,oneof=discrimination production minimal_pairs"`
	UserResponse string `json:"user_response"`
	ResponseTime int    `json:"response_time"` // milliseconds
	Score        int    `json:"score" validate:"min=0,max=100"`
	Feedback     string `json:"feedback"`
}

// GetPhonemeRequest represents request for phonemes
type GetPhonemeRequest struct {
	LanguageID int    `json:"language_id" validate:"required"`
	Category   string `json:"category"`   // vowel, consonant, diphthong
	Difficulty int    `json:"difficulty"` // 1-5
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

// CreateExerciseRequest represents request to create a new exercise
type CreateExerciseRequest struct {
	LanguageID   int    `json:"language_id" validate:"required"`
	PhonemeID    int    `json:"phoneme_id"`
	ExerciseType string `json:"exercise_type" validate:"required"`
	Title        string `json:"title" validate:"required,min=1,max=255"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
	Data         string `json:"data" validate:"required"`
	Difficulty   int    `json:"difficulty" validate:"min=1,max=5"`
	Duration     int    `json:"duration" validate:"min=10,max=3600"`
}

// ExerciseSessionRequest represents starting an exercise session
type ExerciseSessionRequest struct {
	ExerciseID string `json:"exercise_id" validate:"required"`
	Responses  string `json:"responses" validate:"required"`
	Duration   int    `json:"duration" validate:"required,min=1"`
}

// CreateMinimalPairRequest represents request to create minimal pairs
type CreateMinimalPairRequest struct {
	LanguageID int    `json:"language_id" validate:"required"`
	PhonemeID1 int    `json:"phoneme_id_1" validate:"required"`
	PhonemeID2 int    `json:"phoneme_id_2" validate:"required"`
	Word1      string `json:"word1" validate:"required,min=1,max=100"`
	Word2      string `json:"word2" validate:"required,min=1,max=100"`
	AudioURL1  string `json:"audio_url_1" validate:"omitempty,url"`
	AudioURL2  string `json:"audio_url_2" validate:"omitempty,url"`
	Difficulty int    `json:"difficulty" validate:"min=1,max=5"`
}

// Response models

// PhoneticProgressResponse represents user's phonetic progress
type PhoneticProgressResponse struct {
	OverallScore        float64                 `json:"overall_score"`
	TotalPhonemes       int                     `json:"total_phonemes"`
	MasteredPhonemes    int                     `json:"mastered_phonemes"`
	InProgressPhonemes  int                     `json:"in_progress_phonemes"`
	WeakPhonemes        []Phoneme               `json:"weak_phonemes"`
	RecentProgress      []UserPhoneticProgress  `json:"recent_progress"`
	NextRecommendations []PhonemeRecommendation `json:"next_recommendations"`
}

// PhonemeRecommendation represents suggested phonemes to practice
type PhonemeRecommendation struct {
	Phoneme       Phoneme `json:"phoneme"`
	Reason        string  `json:"reason"`
	Priority      int     `json:"priority"`       // 1-5, 5 being highest
	EstimatedTime int     `json:"estimated_time"` // minutes
}

// ExerciseProgressResponse represents progress in exercises
type ExerciseProgressResponse struct {
	TotalExercises     int        `json:"total_exercises"`
	CompletedExercises int        `json:"completed_exercises"`
	AverageScore       float64    `json:"average_score"`
	TotalTime          int        `json:"total_time"` // seconds
	StreakDays         int        `json:"streak_days"`
	LastPracticeDate   *time.Time `json:"last_practice_date"`
}

// PhoneticLessonPlan represents a structured learning plan
type PhoneticLessonPlan struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	LanguageID  int                    `json:"language_id"`
	Level       string                 `json:"level"` // beginner, intermediate, advanced
	Lessons     []PhoneticLesson       `json:"lessons"`
	Progress    PhoneticLessonProgress `json:"progress"`
}

// PhoneticLesson represents a single lesson in a plan
type PhoneticLesson struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	Phonemes      []Phoneme          `json:"phonemes"`
	Exercises     []PhoneticExercise `json:"exercises"`
	MinimalPairs  []MinimalPair      `json:"minimal_pairs"`
	EstimatedTime int                `json:"estimated_time"` // minutes
	Prerequisites []string           `json:"prerequisites"`  // lesson IDs
}

// PhoneticLessonProgress represents user progress in lesson plans
type PhoneticLessonProgress struct {
	CompletedLessons []string  `json:"completed_lessons"`
	CurrentLesson    string    `json:"current_lesson"`
	OverallProgress  float64   `json:"overall_progress"` // percentage
	StartedAt        time.Time `json:"started_at"`
	LastAccessedAt   time.Time `json:"last_accessed_at"`
}

// PhoneticStatistics represents comprehensive phonetic learning stats
type PhoneticStatistics struct {
	TotalPracticeTime  int                         `json:"total_practice_time"` // minutes
	SessionsCompleted  int                         `json:"sessions_completed"`
	AverageSessionTime float64                     `json:"average_session_time"` // minutes
	OverallAccuracy    float64                     `json:"overall_accuracy"`
	WeakestPhonemes    []PhonemeWithScore          `json:"weakest_phonemes"`
	StrongestPhonemes  []PhonemeWithScore          `json:"strongest_phonemes"`
	ProgressByCategory map[string]CategoryProgress `json:"progress_by_category"`
	WeeklyProgress     []WeeklyPhoneticProgress    `json:"weekly_progress"`
	AchievementHistory []PhoneticAchievement       `json:"achievement_history"`
}

// PhonemeWithScore represents a phoneme with its performance score
type PhonemeWithScore struct {
	Phoneme Phoneme `json:"phoneme"`
	Score   float64 `json:"score"`
}

// CategoryProgress represents progress in a phonetic category
type CategoryProgress struct {
	Category      string  `json:"category"`
	TotalPhonemes int     `json:"total_phonemes"`
	MasteredCount int     `json:"mastered_count"`
	AverageScore  float64 `json:"average_score"`
	TimeSpent     int     `json:"time_spent"` // minutes
}

// WeeklyPhoneticProgress represents weekly learning progress
type WeeklyPhoneticProgress struct {
	WeekStart         string  `json:"week_start"`
	WeekEnd           string  `json:"week_end"`
	SessionsCount     int     `json:"sessions_count"`
	TotalTime         int     `json:"total_time"` // minutes
	AverageScore      float64 `json:"average_score"`
	NewPhonemes       int     `json:"new_phonemes"`
	PerfectedPhonemes int     `json:"perfected_phonemes"`
}

// PhoneticAchievement represents achievements in phonetic learning
type PhoneticAchievement struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	IconURL     string    `json:"icon_url"`
	EarnedAt    time.Time `json:"earned_at"`
	Category    string    `json:"category"`
	Points      int       `json:"points"`
}
