// internal/progress/models.go
package progress

import (
	"time"

	"gorm.io/gorm"
)

type UserProgress struct {
	ID                      string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID                  string    `json:"user_id" gorm:"not null"`
	ContentID               string    `json:"content_id" gorm:"not null"`
	EpisodeID               *string   `json:"episode_id"`
	WatchedAt               time.Time `json:"watched_at" gorm:"default:CURRENT_TIMESTAMP"`
	DurationMinutes         int       `json:"duration_minutes" gorm:"not null"`
	ComprehensionPercentage int       `json:"comprehension_percentage" gorm:"check:comprehension_percentage >= 0 AND comprehension_percentage <= 100"`
	DifficultyRating        int       `json:"difficulty_rating" gorm:"check:difficulty_rating >= 1 AND difficulty_rating <= 5"`
	EnjoymentRating         int       `json:"enjoyment_rating" gorm:"check:enjoyment_rating >= 1 AND enjoyment_rating <= 5"`
	Notes                   string    `json:"notes"`
	Completed               bool      `json:"completed" gorm:"default:false"`
}

type UserStats struct {
	ID                   string `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID               string `json:"user_id" gorm:"not null"`
	LanguageID           int    `json:"language_id" gorm:"not null"`
	TotalInputMinutes    int    `json:"total_input_minutes" gorm:"default:0"`
	CurrentStreakDays    int    `json:"current_streak_days" gorm:"default:0"`
	LongestStreakDays    int    `json:"longest_streak_days" gorm:"default:0"`
	TotalVocabularyWords int    `json:"total_vocabulary_words" gorm:"default:0"`
	TotalPoints          int    `json:"total_points" gorm:"default:0"`
	CurrentLevel         int    `json:"current_level" gorm:"default:1"`

	DailyGoalMinutes int `json:"daily_goal_minutes" gorm:"default:60"`
	WeeklyGoalHours  int `json:"weekly_goal_hours" gorm:"default:7"`
	MonthlyGoalHours int `json:"monthly_goal_hours" gorm:"default:30"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggerignore:"true"`
}

type ProgressAnalytics struct {
	TotalInputHours       float64             `json:"total_input_hours"`
	AverageSessionMinutes float64             `json:"average_session_minutes"`
	CurrentStreak         int                 `json:"current_streak"`
	LongestStreak         int                 `json:"longest_streak"`
	WeeklyProgress        []WeeklyData        `json:"weekly_progress"`
	ContentTypeBreakdown  map[string]int      `json:"content_type_breakdown"`
	ComprehensionTrend    []ComprehensionData `json:"comprehension_trend"`
	MostWatchedContent    []ContentSummary    `json:"most_watched_content"`
	StudyTimeByHour       map[int]int         `json:"study_time_by_hour"`
}

type WeeklyData struct {
	Week     string `json:"week"`
	Minutes  int    `json:"minutes"`
	Sessions int    `json:"sessions"`
}

type ComprehensionData struct {
	Date          string  `json:"date"`
	Comprehension float64 `json:"comprehension"`
}

type ContentSummary struct {
	ContentID    string `json:"content_id"`
	Title        string `json:"title"`
	TotalMinutes int    `json:"total_minutes"`
	Sessions     int    `json:"sessions"`
}

type LogInputRequest struct {
	ContentID               string `json:"content_id" validate:"required"`
	EpisodeID               string `json:"episode_id"`
	LanguageID              int    `json:"language_id"` // Add this field
	DurationMinutes         int    `json:"duration_minutes" validate:"required,min=1"`
	ComprehensionPercentage int    `json:"comprehension_percentage" validate:"min=0,max=100"`
	DifficultyRating        int    `json:"difficulty_rating" validate:"min=1,max=5"`
	EnjoymentRating         int    `json:"enjoyment_rating" validate:"min=1,max=5"`
	Notes                   string `json:"notes"`
	Completed               bool   `json:"completed"`
}

type SetGoalsRequest struct {
	LanguageID       int `json:"language_id" validate:"required"`
	DailyGoalMinutes int `json:"daily_goal_minutes" validate:"required,min=1,max=1440"`
	WeeklyGoalHours  int `json:"weekly_goal_hours" validate:"required,min=1,max=168"`
	MonthlyGoalHours int `json:"monthly_goal_hours" validate:"required,min=1,max=744"`
}

type GoalsResponse struct {
	LanguageID             int     `json:"language_id"`
	DailyGoalMinutes       int     `json:"daily_goal_minutes"`
	WeeklyGoalHours        int     `json:"weekly_goal_hours"`
	MonthlyGoalHours       int     `json:"monthly_goal_hours"`
	DailyProgress          int     `json:"daily_progress_minutes"`
	WeeklyProgress         float64 `json:"weekly_progress_hours"`
	MonthlyProgress        float64 `json:"monthly_progress_hours"`
	DailyProgressPercent   float64 `json:"daily_progress_percent"`
	WeeklyProgressPercent  float64 `json:"weekly_progress_percent"`
	MonthlyProgressPercent float64 `json:"monthly_progress_percent"`
}

type StreakInfo struct {
	CurrentStreak    int       `json:"current_streak"`
	LongestStreak    int       `json:"longest_streak"`
	LastActivityDate time.Time `json:"last_activity_date"`
	StreakStartDate  time.Time `json:"streak_start_date"`
	IsActiveToday    bool      `json:"is_active_today"`
}

type CalendarDay struct {
	Date    string `json:"date"`
	Minutes int    `json:"minutes"`
	HasGoal bool   `json:"has_goal"`
	MetGoal bool   `json:"met_goal"`
}

type StudySession struct {
	Date             string         `json:"date"`
	TotalMinutes     int            `json:"total_minutes"`
	SessionCount     int            `json:"session_count"`
	AvgComprehension float64        `json:"avg_comprehension"`
	ContentTypes     map[string]int `json:"content_types"`
}

type WeeklyReport struct {
	WeekStart          time.Time              `json:"week_start"`
	WeekEnd            time.Time              `json:"week_end"`
	TotalMinutes       int                    `json:"total_minutes"`
	TotalHours         float64                `json:"total_hours"`
	SessionCount       int                    `json:"session_count"`
	AvgSessionLength   float64                `json:"avg_session_length"`
	StreakDays         int                    `json:"streak_days"`
	GoalMet            bool                   `json:"goal_met"`
	GoalProgress       float64                `json:"goal_progress"`
	TopContent         []ContentSummary       `json:"top_content"`
	DailyBreakdown     []DailyProgressSummary `json:"daily_breakdown"`
	ComprehensionTrend []ComprehensionData    `json:"comprehension_trend"`
}

type MonthlyReport struct {
	MonthStart         time.Time               `json:"month_start"`
	MonthEnd           time.Time               `json:"month_end"`
	TotalMinutes       int                     `json:"total_minutes"`
	TotalHours         float64                 `json:"total_hours"`
	SessionCount       int                     `json:"session_count"`
	AvgSessionLength   float64                 `json:"avg_session_length"`
	ActiveDays         int                     `json:"active_days"`
	LongestStreak      int                     `json:"longest_streak"`
	GoalMet            bool                    `json:"goal_met"`
	GoalProgress       float64                 `json:"goal_progress"`
	TopContent         []ContentSummary        `json:"top_content"`
	WeeklyBreakdown    []WeeklyProgressSummary `json:"weekly_breakdown"`
	ComprehensionTrend []ComprehensionData     `json:"comprehension_trend"`
}

type DailyProgressSummary struct {
	Date          string  `json:"date"`
	Minutes       int     `json:"minutes"`
	Sessions      int     `json:"sessions"`
	Comprehension float64 `json:"comprehension"`
	MetGoal       bool    `json:"met_goal"`
}

type WeeklyProgressSummary struct {
	WeekStart     string  `json:"week_start"`
	WeekEnd       string  `json:"week_end"`
	Minutes       int     `json:"minutes"`
	Hours         float64 `json:"hours"`
	Sessions      int     `json:"sessions"`
	Comprehension float64 `json:"comprehension"`
	MetGoal       bool    `json:"met_goal"`
}
