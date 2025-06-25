package social

import (
	"time"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/content"
)

// UserFollow represents following relationship between users
type UserFollow struct {
	ID          string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	FollowerID  string         `json:"follower_id" gorm:"not null"`
	FollowingID string         `json:"following_id" gorm:"not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Constraints
	_ struct{} `gorm:"uniqueIndex:idx_user_follow,unique"`
}

// UserProfile represents public user profile information
type UserProfile struct {
	UserID            string         `json:"user_id" gorm:"primary_key"`
	Username          string         `json:"username" gorm:"not null"`
	DisplayName       string         `json:"display_name"`
	Bio               string         `json:"bio" gorm:"type:text"`
	AvatarURL         string         `json:"avatar_url"`
	CountryCode       string         `json:"country_code"`
	TimeZone          string         `json:"timezone"`
	IsPublic          bool           `json:"is_public" gorm:"default:true"`
	ShowProgress      bool           `json:"show_progress" gorm:"default:true"`
	ShowStreak        bool           `json:"show_streak" gorm:"default:true"`
	AllowMessages     bool           `json:"allow_messages" gorm:"default:true"`
	NativeLanguages   string         `json:"native_languages" gorm:"type:text"`   // JSON array
	LearningLanguages string         `json:"learning_languages" gorm:"type:text"` // JSON array
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Computed fields (not stored in DB)
	FollowersCount int  `json:"followers_count" gorm:"-"`
	FollowingCount int  `json:"following_count" gorm:"-"`
	IsFollowing    bool `json:"is_following" gorm:"-"`
	IsFollower     bool `json:"is_follower" gorm:"-"`
}

// StudyGroup represents a group for collaborative learning
type StudyGroup struct {
	ID               string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name             string         `json:"name" gorm:"not null"`
	Description      string         `json:"description" gorm:"type:text"`
	LanguageID       int            `json:"language_id" gorm:"not null"`
	TargetLevel      string         `json:"target_level"` // A1, A2, B1, B2, C1, C2
	MaxMembers       int            `json:"max_members" gorm:"default:50"`
	IsPublic         bool           `json:"is_public" gorm:"default:true"`
	RequiresApproval bool           `json:"requires_approval" gorm:"default:false"`
	Tags             string         `json:"tags" gorm:"type:text"` // JSON array
	Rules            string         `json:"rules" gorm:"type:text"`
	CreatedBy        string         `json:"created_by" gorm:"not null"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Language content.Language `json:"language,omitempty" gorm:"foreignKey:LanguageID"`

	// Computed fields
	MemberCount int  `json:"member_count" gorm:"-"`
	IsMember    bool `json:"is_member" gorm:"-"`
	IsAdmin     bool `json:"is_admin" gorm:"-"`
}

// GroupMembership represents user membership in study groups
type GroupMembership struct {
	ID        string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	GroupID   string         `json:"group_id" gorm:"not null"`
	UserID    string         `json:"user_id" gorm:"not null"`
	Role      string         `json:"role" gorm:"default:'member'"`   // admin, moderator, member
	Status    string         `json:"status" gorm:"default:'active'"` // active, pending, banned
	JoinedAt  time.Time      `json:"joined_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Group StudyGroup `json:"group,omitempty" gorm:"foreignKey:GroupID"`

	// Constraints
	_ struct{} `gorm:"uniqueIndex:idx_group_user,unique"`
}

// ActivityFeed represents user activities for social feed
type ActivityFeed struct {
	ID          string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      string         `json:"user_id" gorm:"not null"`
	Type        string         `json:"type" gorm:"not null"` // streak, achievement, content_completed, etc.
	Title       string         `json:"title" gorm:"not null"`
	Description string         `json:"description"`
	Data        string         `json:"data" gorm:"type:text"` // JSON with activity-specific data
	LanguageID  *int           `json:"language_id"`
	IsPublic    bool           `json:"is_public" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Language *content.Language `json:"language,omitempty" gorm:"foreignKey:LanguageID"`

	// Computed fields
	Username  string `json:"username" gorm:"-"`
	AvatarURL string `json:"avatar_url" gorm:"-"`
}

// LanguageExchange represents language exchange partnerships
type LanguageExchange struct {
	ID                 string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	User1ID            string         `json:"user1_id" gorm:"not null"`
	User2ID            string         `json:"user2_id" gorm:"not null"`
	User1TeachLanguage int            `json:"user1_teach_language" gorm:"not null"`
	User1LearnLanguage int            `json:"user1_learn_language" gorm:"not null"`
	User2TeachLanguage int            `json:"user2_teach_language" gorm:"not null"`
	User2LearnLanguage int            `json:"user2_learn_language" gorm:"not null"`
	Status             string         `json:"status" gorm:"default:'pending'"` // pending, active, paused, completed
	StartedAt          *time.Time     `json:"started_at"`
	EndedAt            *time.Time     `json:"ended_at"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// Mentorship represents mentor-mentee relationships
type Mentorship struct {
	ID          string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MentorID    string         `json:"mentor_id" gorm:"not null"`
	MenteeID    string         `json:"mentee_id" gorm:"not null"`
	LanguageID  int            `json:"language_id" gorm:"not null"`
	Status      string         `json:"status" gorm:"default:'pending'"` // pending, active, completed, cancelled
	Description string         `json:"description" gorm:"type:text"`
	Goals       string         `json:"goals" gorm:"type:text"` // JSON array
	StartedAt   *time.Time     `json:"started_at"`
	EndedAt     *time.Time     `json:"ended_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Language content.Language `json:"language,omitempty" gorm:"foreignKey:LanguageID"`
}

// UserInteraction represents various user interactions (likes, comments, etc.)
type UserInteraction struct {
	ID         string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     string         `json:"user_id" gorm:"not null"`
	TargetType string         `json:"target_type" gorm:"not null"` // activity, user, group
	TargetID   string         `json:"target_id" gorm:"not null"`
	Type       string         `json:"type" gorm:"not null"`     // like, comment, react, report
	Content    string         `json:"content" gorm:"type:text"` // For comments
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Constraints
	_ struct{} `gorm:"uniqueIndex:idx_user_target_type,unique"`
}

// Leaderboard represents different types of leaderboards
type Leaderboard struct {
	ID         string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Type       string    `json:"type" gorm:"not null"` // weekly_input, monthly_streak, vocabulary_master
	LanguageID *int      `json:"language_id"`
	Period     string    `json:"period"` // week, month, year, all_time
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Data       string    `json:"data" gorm:"type:text"` // JSON with leaderboard entries
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Request/Response models
type FollowUserRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

type UpdateProfileRequest struct {
	DisplayName       string `json:"display_name" validate:"omitempty,max=100"`
	Bio               string `json:"bio" validate:"omitempty,max=500"`
	CountryCode       string `json:"country_code" validate:"omitempty,len=2"`
	TimeZone          string `json:"timezone" validate:"omitempty"`
	IsPublic          *bool  `json:"is_public"`
	ShowProgress      *bool  `json:"show_progress"`
	ShowStreak        *bool  `json:"show_streak"`
	AllowMessages     *bool  `json:"allow_messages"`
	NativeLanguages   []int  `json:"native_languages" validate:"omitempty"`
	LearningLanguages []int  `json:"learning_languages" validate:"omitempty"`
}

type CreateGroupRequest struct {
	Name             string   `json:"name" validate:"required,min=3,max=100"`
	Description      string   `json:"description" validate:"max=1000"`
	LanguageID       int      `json:"language_id" validate:"required"`
	TargetLevel      string   `json:"target_level" validate:"omitempty,oneof=A1 A2 B1 B2 C1 C2"`
	MaxMembers       int      `json:"max_members" validate:"min=2,max=500"`
	IsPublic         bool     `json:"is_public"`
	RequiresApproval bool     `json:"requires_approval"`
	Tags             []string `json:"tags" validate:"omitempty,max=10"`
	Rules            string   `json:"rules" validate:"omitempty,max=2000"`
}

type JoinGroupRequest struct {
	Message string `json:"message" validate:"omitempty,max=200"`
}

type CreateActivityRequest struct {
	Type        string `json:"type" validate:"required"`
	Title       string `json:"title" validate:"required,max=200"`
	Description string `json:"description" validate:"omitempty,max=500"`
	Data        string `json:"data" validate:"omitempty"`
	LanguageID  *int   `json:"language_id"`
	IsPublic    bool   `json:"is_public"`
}

type LeaderboardEntry struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Score       int    `json:"score"`
	Rank        int    `json:"rank"`
	Language    string `json:"language,omitempty"`
}

type LeaderboardResponse struct {
	Type      string             `json:"type"`
	Period    string             `json:"period"`
	Language  *content.Language  `json:"language,omitempty"`
	StartDate time.Time          `json:"start_date"`
	EndDate   time.Time          `json:"end_date"`
	Entries   []LeaderboardEntry `json:"entries"`
	UserRank  *int               `json:"user_rank,omitempty"`
	UserScore *int               `json:"user_score,omitempty"`
}

type SocialStatsResponse struct {
	FollowersCount   int `json:"followers_count"`
	FollowingCount   int `json:"following_count"`
	GroupsCount      int `json:"groups_count"`
	ActivitiesCount  int `json:"activities_count"`
	MentorshipsCount int `json:"mentorships_count"`
	ExchangesCount   int `json:"exchanges_count"`
	ReputationScore  int `json:"reputation_score"`
}

type GroupFilter struct {
	LanguageID  int      `json:"language_id"`
	TargetLevel string   `json:"target_level"`
	IsPublic    *bool    `json:"is_public"`
	HasSpace    bool     `json:"has_space"`
	Tags        []string `json:"tags"`
	Search      string   `json:"search"`
	Limit       int      `json:"limit"`
	Offset      int      `json:"offset"`
}

type UserFilter struct {
	LanguageID           int    `json:"language_id"`
	CountryCode          string `json:"country_code"`
	IsLookingForMentor   bool   `json:"is_looking_for_mentor"`
	IsLookingForExchange bool   `json:"is_looking_for_exchange"`
	Search               string `json:"search"`
	Limit                int    `json:"limit"`
	Offset               int    `json:"offset"`
}

type DiscoverContent struct {
	PopularGroups     []StudyGroup       `json:"popular_groups"`
	RecommendedUsers  []UserProfile      `json:"recommended_users"`
	RecentActivities  []ActivityFeed     `json:"recent_activities"`
	TrendingLanguages []content.Language `json:"trending_languages"`
	UpcomingEvents    []interface{}      `json:"upcoming_events"` // Future feature
}
