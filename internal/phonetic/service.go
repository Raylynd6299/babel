package phonetic

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db        *gorm.DB
	jwtSecret string
}

func NewService(db *gorm.DB, jwtSecret string) *Service {
	return &Service{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

// Phoneme management
func (s *Service) GetPhonemesByLanguage(ctx context.Context, languageID int) ([]Phoneme, error) {
	var phonemes []Phoneme
	err := s.db.Where("language_id = ?", languageID).
		Order("category, symbol").
		Find(&phonemes).Error
	return phonemes, err
}

func (s *Service) GetPhoneme(ctx context.Context, id int) (*Phoneme, error) {
	var phoneme Phoneme
	err := s.db.Preload("Language").Where("id = ?", id).First(&phoneme).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("phoneme not found")
		}
		return nil, err
	}
	return &phoneme, nil
}

// User progress tracking
func (s *Service) GetUserProgress(ctx context.Context, userID string, languageID int) ([]PhoneticProgressResponse, error) {
	var progress []UserPhoneticProgress

	query := `
        SELECT upp.*, p.symbol 
        FROM user_phonetic_progress upp
        JOIN phonemes p ON upp.phoneme_id = p.id
        WHERE upp.user_id = ? AND p.language_id = ?
        ORDER BY upp.mastery_level ASC, upp.last_practiced_at ASC
    `

	err := s.db.Raw(query, userID, languageID).Scan(&progress).Error
	if err != nil {
		return nil, err
	}

	result := make([]PhoneticProgressResponse, len(progress))
	for i, p := range progress {
		result[i] = PhoneticProgressResponse{
			PhonemeID:           p.PhonemeID,
			Symbol:              p.Phoneme.Symbol,
			DiscriminationScore: p.DiscriminationScore,
			ProductionScore:     p.ProductionScore,
			MasteryLevel:        p.MasteryLevel,
			PracticeCount:       p.PracticeCount,
			LastPracticedAt:     p.LastPracticedAt,
			RecommendedNext:     p.MasteryLevel < 3, // Recommend if not mastered
		}
	}

	return result, nil
}

func (s *Service) PracticePhoneme(ctx context.Context, userID string, req PracticePhonemeRequest) error {
	// Get or create user progress
	var progress UserPhoneticProgress
	err := s.db.Where("user_id = ? AND phoneme_id = ?", userID, req.PhonemeID).First(&progress).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new progress
		progress = UserPhoneticProgress{
			UserID:    userID,
			PhonemeID: req.PhonemeID,
		}
		if err := s.db.Create(&progress).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Update progress based on practice type
	now := time.Now()
	progress.LastPracticedAt = &now
	progress.PracticeCount++

	if req.Type == "discrimination" {
		progress.DiscriminationScore = s.calculateNewScore(progress.DiscriminationScore, req.Score, progress.PracticeCount)
	} else if req.Type == "production" {
		progress.ProductionScore = s.calculateNewScore(progress.ProductionScore, req.Score, progress.PracticeCount)
	}

	// Update mastery level
	progress.MasteryLevel = s.calculateMasteryLevel(progress.DiscriminationScore, progress.ProductionScore)

	return s.db.Save(&progress).Error
}

func (s *Service) calculateNewScore(currentScore, newScore, practiceCount int) int {
	if practiceCount == 1 {
		return newScore
	}

	// Weighted average giving more weight to recent performance
	weight := 0.3 // 30% weight to new score
	return int(float64(currentScore)*(1-weight) + float64(newScore)*weight)
}

func (s *Service) calculateMasteryLevel(discriminationScore, productionScore int) int {
	avgScore := (discriminationScore + productionScore) / 2

	switch {
	case avgScore >= 90:
		return 5 // Master
	case avgScore >= 80:
		return 4 // Advanced
	case avgScore >= 70:
		return 3 // Intermediate
	case avgScore >= 60:
		return 2 // Beginner
	case avgScore >= 50:
		return 1 // Novice
	default:
		return 0 // Needs practice
	}
}

// Exercise management
func (s *Service) GetExercises(ctx context.Context, filter ExerciseFilter) ([]PhoneticExercise, int64, error) {
	query := s.db.Model(&PhoneticExercise{}).Preload("Phoneme")

	if filter.PhonemeID > 0 {
		query = query.Where("phoneme_id = ?", filter.PhonemeID)
	}

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if len(filter.Difficulty) > 0 {
		query = query.Where("difficulty IN ?", filter.Difficulty)
	}

	if filter.LanguageID > 0 {
		query = query.Joins("JOIN phonemes ON phonetic_exercises.phoneme_id = phonemes.id").
			Where("phonemes.language_id = ?", filter.LanguageID)
	}

	query = query.Where("is_active = ?", true)

	var total int64
	query.Count(&total)

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(20)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var exercises []PhoneticExercise
	err := query.Order("difficulty ASC, title ASC").Find(&exercises).Error

	return exercises, total, err
}

func (s *Service) GetExercise(ctx context.Context, id string) (*PhoneticExercise, error) {
	var exercise PhoneticExercise
	err := s.db.Preload("Phoneme").Where("id = ? AND is_active = ?", id, true).First(&exercise).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("exercise not found")
		}
		return nil, err
	}
	return &exercise, nil
}

func (s *Service) StartExercise(ctx context.Context, userID string, exerciseID string) (*UserExerciseSession, error) {
	// Verify exercise exists
	_, err := s.GetExercise(ctx, exerciseID)
	if err != nil {
		return nil, err
	}

	session := UserExerciseSession{
		UserID:     userID,
		ExerciseID: exerciseID,
		StartedAt:  time.Now(),
	}

	if err := s.db.Create(&session).Error; err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *Service) CompleteExercise(ctx context.Context, userID string, req ExerciseCompleteRequest) (*UserExerciseSession, error) {
	var session UserExerciseSession
	err := s.db.Where("id = ? AND user_id = ?", req.SessionID, userID).First(&session).Error
	if err != nil {
		return nil, errors.New("session not found")
	}

	if session.CompletedAt != nil {
		return nil, errors.New("session already completed")
	}

	now := time.Now()
	session.CompletedAt = &now
	session.Score = req.Score
	session.Accuracy = req.Accuracy
	session.TimeSpent = req.TimeSpent
	session.Responses = req.Responses

	if err := s.db.Save(&session).Error; err != nil {
		return nil, err
	}

	return &session, nil
}

// Minimal pairs
func (s *Service) GetMinimalPairs(ctx context.Context, languageID int, phoneme1ID, phoneme2ID int) ([]MinimalPair, error) {
	query := s.db.Where("language_id = ?", languageID)

	if phoneme1ID > 0 && phoneme2ID > 0 {
		query = query.Where("(phoneme1_id = ? AND phoneme2_id = ?) OR (phoneme1_id = ? AND phoneme2_id = ?)",
			phoneme1ID, phoneme2ID, phoneme2ID, phoneme1ID)
	} else if phoneme1ID > 0 {
		query = query.Where("phoneme1_id = ? OR phoneme2_id = ?", phoneme1ID, phoneme1ID)
	}

	var pairs []MinimalPair
	err := query.Preload("Phoneme1").Preload("Phoneme2").
		Order("difficulty ASC").Find(&pairs).Error

	return pairs, err
}

// Statistics
func (s *Service) GetPhoneticStats(ctx context.Context, userID string, languageID int) (*PhoneticStatsResponse, error) {
	// Get total phonemes for language
	var totalPhonemes int64
	s.db.Model(&Phoneme{}).Where("language_id = ?", languageID).Count(&totalPhonemes)

	// Get user progress stats
	var practiceStats struct {
		PracticedCount int     `json:"practiced_count"`
		MasteredCount  int     `json:"mastered_count"`
		AvgScore       float64 `json:"avg_score"`
		TotalTime      int     `json:"total_time"`
	}

	query := `
        SELECT 
            COUNT(DISTINCT upp.phoneme_id) as practiced_count,
            COUNT(CASE WHEN upp.mastery_level >= 4 THEN 1 END) as mastered_count,
            AVG((upp.discrimination_score + upp.production_score) / 2.0) as avg_score,
            SUM(COALESCE(sess.total_time, 0)) as total_time
        FROM user_phonetic_progress upp
        JOIN phonemes p ON upp.phoneme_id = p.id
        LEFT JOIN (
            SELECT exercise_id, SUM(time_spent) as total_time
            FROM user_exercise_sessions 
            WHERE user_id = ? AND completed_at IS NOT NULL
            GROUP BY exercise_id
        ) sess ON sess.exercise_id = upp.phoneme_id
        WHERE upp.user_id = ? AND p.language_id = ?
    `

	s.db.Raw(query, userID, userID, languageID).Scan(&practiceStats)

	// Get weakest phonemes
	weakestPhonemes := s.getWeakestPhonemes(userID, languageID, 5)

	// Get recommended practice
	recommendedPractice := s.getRecommendedPractice(userID, languageID, 5)

	return &PhoneticStatsResponse{
		LanguageID:          languageID,
		TotalPhonemes:       int(totalPhonemes),
		PracticedPhonemes:   practiceStats.PracticedCount,
		MasteredPhonemes:    practiceStats.MasteredCount,
		AverageScore:        practiceStats.AvgScore,
		TotalPracticeTime:   practiceStats.TotalTime / 60, // Convert to minutes
		WeakestPhonemes:     weakestPhonemes,
		RecommendedPractice: recommendedPractice,
	}, nil
}

func (s *Service) getWeakestPhonemes(userID string, languageID int, limit int) []PhoneticProgressResponse {
	var progress []UserPhoneticProgress

	query := `
        SELECT upp.*, p.symbol
        FROM user_phonetic_progress upp
        JOIN phonemes p ON upp.phoneme_id = p.id
        WHERE upp.user_id = ? AND p.language_id = ? AND upp.practice_count > 0
        ORDER BY (upp.discrimination_score + upp.production_score) / 2.0 ASC
        LIMIT ?
    `

	s.db.Raw(query, userID, languageID, limit).Scan(&progress)

	result := make([]PhoneticProgressResponse, len(progress))
	for i, p := range progress {
		result[i] = PhoneticProgressResponse{
			PhonemeID:           p.PhonemeID,
			Symbol:              p.Phoneme.Symbol,
			DiscriminationScore: p.DiscriminationScore,
			ProductionScore:     p.ProductionScore,
			MasteryLevel:        p.MasteryLevel,
			PracticeCount:       p.PracticeCount,
			LastPracticedAt:     p.LastPracticedAt,
			RecommendedNext:     true,
		}
	}

	return result
}

func (s *Service) getRecommendedPractice(userID string, languageID int, limit int) []PhoneticProgressResponse {
	var progress []UserPhoneticProgress

	// Get phonemes that haven't been practiced recently or need more practice
	query := `
        SELECT upp.*, p.symbol
        FROM user_phonetic_progress upp
        JOIN phonemes p ON upp.phoneme_id = p.id
        WHERE upp.user_id = ? AND p.language_id = ? 
        AND (upp.last_practiced_at IS NULL OR upp.last_practiced_at < ? OR upp.mastery_level < 3)
        ORDER BY 
            CASE WHEN upp.last_practiced_at IS NULL THEN 0 ELSE 1 END,
            upp.last_practiced_at ASC,
            upp.mastery_level ASC
        LIMIT ?
    `

	threeDaysAgo := time.Now().AddDate(0, 0, -3)
	s.db.Raw(query, userID, languageID, threeDaysAgo, limit).Scan(&progress)

	result := make([]PhoneticProgressResponse, len(progress))
	for i, p := range progress {
		result[i] = PhoneticProgressResponse{
			PhonemeID:           p.PhonemeID,
			Symbol:              p.Phoneme.Symbol,
			DiscriminationScore: p.DiscriminationScore,
			ProductionScore:     p.ProductionScore,
			MasteryLevel:        p.MasteryLevel,
			PracticeCount:       p.PracticeCount,
			LastPracticedAt:     p.LastPracticedAt,
			RecommendedNext:     true,
		}
	}

	return result
}

// Data seeding helpers
func (s *Service) SeedPhonemes(ctx context.Context, languageID int, phonemes []Phoneme) error {
	for _, phoneme := range phonemes {
		phoneme.LanguageID = languageID

		var existing Phoneme
		err := s.db.Where("language_id = ? AND symbol = ?", languageID, phoneme.Symbol).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := s.db.Create(&phoneme).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]UserExerciseSession, error) {
	var sessions []UserExerciseSession
	err := s.db.Preload("Exercise").
		Where("user_id = ?", userID).
		Order("started_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&sessions).Error

	return sessions, err
}

func (s *Service) GetRecommendations(ctx context.Context, userID string, languageID int, limit int) ([]PhoneticProgressResponse, error) {
	// Get phonemes that need practice based on user progress
	var progress []UserPhoneticProgress

	query := `
        SELECT DISTINCT ON (p.id) p.id as phoneme_id, p.symbol, 
               COALESCE(upp.discrimination_score, 0) as discrimination_score,
               COALESCE(upp.production_score, 0) as production_score,
               COALESCE(upp.mastery_level, 0) as mastery_level,
               COALESCE(upp.practice_count, 0) as practice_count,
               upp.last_practiced_at
        FROM phonemes p
        LEFT JOIN user_phonetic_progress upp ON p.id = upp.phoneme_id AND upp.user_id = ?
        WHERE p.language_id = ? 
        AND (upp.mastery_level IS NULL OR upp.mastery_level < 3 OR upp.last_practiced_at < ?)
        ORDER BY p.id, COALESCE(upp.mastery_level, -1) ASC, COALESCE(upp.last_practiced_at, '1970-01-01') ASC
        LIMIT ?
    `

	threeDaysAgo := time.Now().AddDate(0, 0, -3)
	err := s.db.Raw(query, userID, languageID, threeDaysAgo, limit).Scan(&progress).Error
	if err != nil {
		return nil, err
	}

	result := make([]PhoneticProgressResponse, len(progress))
	for i, p := range progress {
		result[i] = PhoneticProgressResponse{
			PhonemeID:           p.PhonemeID,
			Symbol:              p.Phoneme.Symbol,
			DiscriminationScore: p.DiscriminationScore,
			ProductionScore:     p.ProductionScore,
			MasteryLevel:        p.MasteryLevel,
			PracticeCount:       p.PracticeCount,
			LastPracticedAt:     p.LastPracticedAt,
			RecommendedNext:     true,
		}
	}

	return result, nil
}

func (s *Service) GetWeakPhonemes(ctx context.Context, userID string, languageID int, limit int) ([]PhoneticProgressResponse, error) {
	return s.getWeakestPhonemes(userID, languageID, limit), nil
}

func (s *Service) GetPracticePlan(ctx context.Context, userID string, languageID int) (*PracticePlan, error) {
	var plan PracticePlan
	err := s.db.Where("user_id = ? AND language_id = ? AND is_active = ?", userID, languageID, true).
		First(&plan).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("no active practice plan found")
	}

	return &plan, err
}

func (s *Service) CreatePracticePlan(ctx context.Context, userID string, req CreatePracticePlanRequest) (*PracticePlan, error) {
	// Deactivate any existing plans
	s.db.Model(&PracticePlan{}).
		Where("user_id = ? AND language_id = ?", userID, req.LanguageID).
		Update("is_active", false)

	// Convert focus areas to JSON
	focusAreasJSON, err := json.Marshal(req.FocusAreas)
	if err != nil {
		return nil, err
	}

	plan := PracticePlan{
		UserID:            userID,
		LanguageID:        req.LanguageID,
		Name:              req.Name,
		Description:       req.Description,
		DurationWeeks:     req.DurationWeeks,
		SessionsPerWeek:   req.SessionsPerWeek,
		MinutesPerSession: req.MinutesPerSession,
		FocusAreas:        string(focusAreasJSON),
		IsActive:          true,
	}

	if err := s.db.Create(&plan).Error; err != nil {
		return nil, err
	}

	return &plan, nil
}
