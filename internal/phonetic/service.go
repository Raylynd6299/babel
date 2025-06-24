package phonetic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func (s *Service) GetPhonemes(ctx context.Context, languageID int, filter GetPhonemeRequest) ([]Phoneme, int64, error) {
	var phonemes []Phoneme
	var total int64

	query := s.db.Where("language_id = ?", languageID)

	// Apply filters
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if filter.Difficulty > 0 {
		query = query.Where("difficulty = ?", filter.Difficulty)
	}

	// Count total
	query.Model(&Phoneme{}).Count(&total)

	// Apply pagination
	limit := 20
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}

	offset := 0
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	err := query.Order("difficulty ASC, symbol ASC").
		Limit(limit).
		Offset(offset).
		Find(&phonemes).Error

	return phonemes, total, err
}

func (s *Service) GetPhonemeByID(ctx context.Context, phonemeID int) (*Phoneme, error) {
	var phoneme Phoneme
	err := s.db.Preload("Language").Where("id = ?", phonemeID).First(&phoneme).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("phoneme not found")
		}
		return nil, err
	}
	return &phoneme, nil
}

// User progress tracking

func (s *Service) GetUserPhoneticProgress(ctx context.Context, userID string, languageID int) (*PhoneticProgressResponse, error) {
	// Get all phonemes for the language
	var totalPhonemes int64
	s.db.Model(&Phoneme{}).Where("language_id = ?", languageID).Count(&totalPhonemes)

	// Get user progress
	var userProgress []UserPhoneticProgress
	err := s.db.Preload("Phoneme").
		Joins("JOIN phonemes ON user_phonetic_progress.phoneme_id = phonemes.id").
		Where("user_phonetic_progress.user_id = ? AND phonemes.language_id = ?", userID, languageID).
		Find(&userProgress).Error
	if err != nil {
		return nil, err
	}

	// Calculate statistics
	masteredCount := 0
	inProgressCount := 0
	totalScore := 0.0
	weakPhonemes := make([]Phoneme, 0)

	for _, progress := range userProgress {
		averageScore := float64(progress.DiscriminationScore+progress.ProductionScore) / 2

		if progress.MasteryLevel >= 4 {
			masteredCount++
		} else if progress.MasteryLevel > 0 {
			inProgressCount++
		}

		totalScore += averageScore

		// Identify weak phonemes (score < 70%)
		if averageScore < 70 && progress.PracticeCount > 2 {
			weakPhonemes = append(weakPhonemes, progress.Phoneme)
		}
	}

	overallScore := 0.0
	if len(userProgress) > 0 {
		overallScore = totalScore / float64(len(userProgress))
	}

	// Get recent progress (last 10 sessions)
	var recentProgress []UserPhoneticProgress
	s.db.Preload("Phoneme").
		Joins("JOIN phonemes ON user_phonetic_progress.phoneme_id = phonemes.id").
		Where("user_phonetic_progress.user_id = ? AND phonemes.language_id = ?", userID, languageID).
		Where("user_phonetic_progress.last_practiced_at IS NOT NULL").
		Order("user_phonetic_progress.last_practiced_at DESC").
		Limit(10).
		Find(&recentProgress)

	// Generate recommendations
	recommendations := s.generatePhonemeRecommendations(userID, languageID, userProgress)

	return &PhoneticProgressResponse{
		OverallScore:        overallScore,
		TotalPhonemes:       int(totalPhonemes),
		MasteredPhonemes:    masteredCount,
		InProgressPhonemes:  inProgressCount,
		WeakPhonemes:        weakPhonemes,
		RecentProgress:      recentProgress,
		NextRecommendations: recommendations,
	}, nil
}

func (s *Service) PracticePhoneme(ctx context.Context, userID string, req PracticePhonemeRequest) (*UserPhoneticProgress, error) {
	// Get or create user progress
	var userProgress UserPhoneticProgress
	err := s.db.Where("user_id = ? AND phoneme_id = ?", userID, req.PhonemeID).First(&userProgress).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new progress entry
		userProgress = UserPhoneticProgress{
			UserID:    userID,
			PhonemeID: req.PhonemeID,
		}
	} else if err != nil {
		return nil, err
	}

	// Update progress based on exercise type and score
	now := time.Now()
	userProgress.LastPracticedAt = &now
	userProgress.PracticeCount++

	switch req.ExerciseType {
	case "discrimination":
		userProgress.DiscriminationScore = s.calculateNewScore(userProgress.DiscriminationScore, req.Score, userProgress.PracticeCount)
	case "production":
		userProgress.ProductionScore = s.calculateNewScore(userProgress.ProductionScore, req.Score, userProgress.PracticeCount)
	case "minimal_pairs":
		// Update both scores for minimal pairs exercises
		userProgress.DiscriminationScore = s.calculateNewScore(userProgress.DiscriminationScore, req.Score, userProgress.PracticeCount)
	}

	// Calculate mastery level
	averageScore := (userProgress.DiscriminationScore + userProgress.ProductionScore) / 2
	userProgress.MasteryLevel = s.calculateMasteryLevel(averageScore, userProgress.PracticeCount)

	// Save progress
	if userProgress.ID == "" {
		err = s.db.Create(&userProgress).Error
	} else {
		err = s.db.Save(&userProgress).Error
	}

	if err != nil {
		return nil, err
	}

	// Load phoneme data
	s.db.Preload("Phoneme").Where("id = ?", userProgress.ID).First(&userProgress)

	return &userProgress, nil
}

// Exercise management

func (s *Service) GetExercises(ctx context.Context, languageID int, exerciseType string, difficulty int) ([]PhoneticExercise, error) {
	query := s.db.Where("language_id = ? AND is_active = ?", languageID, true)

	if exerciseType != "" {
		query = query.Where("exercise_type = ?", exerciseType)
	}

	if difficulty > 0 {
		query = query.Where("difficulty = ?", difficulty)
	}

	var exercises []PhoneticExercise
	err := query.Preload("Language").Preload("Phoneme").
		Order("difficulty ASC, created_at ASC").
		Find(&exercises).Error

	return exercises, err
}

func (s *Service) CreateExercise(ctx context.Context, req CreateExerciseRequest) (*PhoneticExercise, error) {
	exercise := PhoneticExercise{
		LanguageID:   req.LanguageID,
		PhonemeID:    req.PhonemeID,
		ExerciseType: req.ExerciseType,
		Title:        req.Title,
		Description:  req.Description,
		Instructions: req.Instructions,
		Data:         req.Data,
		Difficulty:   req.Difficulty,
		Duration:     req.Duration,
		IsActive:     true,
	}

	err := s.db.Create(&exercise).Error
	if err != nil {
		return nil, err
	}

	return &exercise, nil
}

func (s *Service) CompleteExerciseSession(ctx context.Context, userID string, req ExerciseSessionRequest) (*UserExerciseSession, error) {
	// Validate exercise exists
	var exercise PhoneticExercise
	if err := s.db.Where("id = ?", req.ExerciseID).First(&exercise).Error; err != nil {
		return nil, errors.New("exercise not found")
	}

	// Calculate score based on responses
	score := s.calculateExerciseScore(req.Responses, exercise.Data, exercise.ExerciseType)

	// Create session record
	session := UserExerciseSession{
		UserID:      userID,
		ExerciseID:  req.ExerciseID,
		Score:       score,
		Duration:    req.Duration,
		Responses:   req.Responses,
		Feedback:    s.generateExerciseFeedback(score, exercise.ExerciseType),
		CompletedAt: time.Now(),
	}

	err := s.db.Create(&session).Error
	if err != nil {
		return nil, err
	}

	// Load exercise data
	s.db.Preload("Exercise").Where("id = ?", session.ID).First(&session)

	return &session, nil
}

// Minimal pairs management

func (s *Service) GetMinimalPairs(ctx context.Context, languageID int, phonemeID1, phonemeID2 int) ([]MinimalPair, error) {
	query := s.db.Where("language_id = ?", languageID)

	if phonemeID1 > 0 && phonemeID2 > 0 {
		query = query.Where("(phoneme_id_1 = ? AND phoneme_id_2 = ?) OR (phoneme_id_1 = ? AND phoneme_id_2 = ?)",
			phonemeID1, phonemeID2, phonemeID2, phonemeID1)
	} else if phonemeID1 > 0 {
		query = query.Where("phoneme_id_1 = ? OR phoneme_id_2 = ?", phonemeID1, phonemeID1)
	}

	var pairs []MinimalPair
	err := query.Preload("Language").Preload("Phoneme1").Preload("Phoneme2").
		Order("difficulty ASC").
		Find(&pairs).Error

	return pairs, err
}

func (s *Service) CreateMinimalPair(ctx context.Context, req CreateMinimalPairRequest) (*MinimalPair, error) {
	// Validate phonemes exist
	var phoneme1, phoneme2 Phoneme
	if err := s.db.Where("id = ? AND language_id = ?", req.PhonemeID1, req.LanguageID).First(&phoneme1).Error; err != nil {
		return nil, errors.New("phoneme 1 not found")
	}
	if err := s.db.Where("id = ? AND language_id = ?", req.PhonemeID2, req.LanguageID).First(&phoneme2).Error; err != nil {
		return nil, errors.New("phoneme 2 not found")
	}

	pair := MinimalPair{
		LanguageID: req.LanguageID,
		PhonemeID1: req.PhonemeID1,
		PhonemeID2: req.PhonemeID2,
		Word1:      req.Word1,
		Word2:      req.Word2,
		AudioURL1:  req.AudioURL1,
		AudioURL2:  req.AudioURL2,
		Difficulty: req.Difficulty,
	}

	err := s.db.Create(&pair).Error
	if err != nil {
		return nil, err
	}

	// Load relations
	s.db.Preload("Language").Preload("Phoneme1").Preload("Phoneme2").
		Where("id = ?", pair.ID).First(&pair)

	return &pair, nil
}

// Statistics and analytics

func (s *Service) GetPhoneticStatistics(ctx context.Context, userID string, languageID int, days int) (*PhoneticStatistics, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	stats := &PhoneticStatistics{
		ProgressByCategory: make(map[string]CategoryProgress),
	}

	// Get total practice time and sessions
	var sessions []UserExerciseSession
	s.db.Joins("JOIN phonetic_exercises ON user_exercise_sessions.exercise_id = phonetic_exercises.id").
		Where("user_exercise_sessions.user_id = ? AND phonetic_exercises.language_id = ? AND user_exercise_sessions.completed_at >= ?",
			userID, languageID, startDate).
		Find(&sessions)

	totalTime := 0
	totalScore := 0.0
	for _, session := range sessions {
		totalTime += session.Duration
		totalScore += float64(session.Score)
	}

	stats.SessionsCompleted = len(sessions)
	stats.TotalPracticeTime = totalTime / 60 // Convert to minutes

	if len(sessions) > 0 {
		stats.AverageSessionTime = float64(totalTime) / float64(len(sessions)) / 60
		stats.OverallAccuracy = totalScore / float64(len(sessions))
	}

	// Get weakest and strongest phonemes
	stats.WeakestPhonemes, stats.StrongestPhonemes = s.getPhonemesByPerformance(userID, languageID)

	// Get progress by category
	stats.ProgressByCategory = s.getProgressByCategory(userID, languageID)

	// Get weekly progress
	stats.WeeklyProgress = s.getWeeklyPhoneticProgress(userID, languageID, days)

	return stats, nil
}

// Helper methods

func (s *Service) calculateNewScore(currentScore, newScore, practiceCount int) int {
	if practiceCount == 1 {
		return newScore
	}

	// Weighted average giving more weight to recent performance
	weight := 0.3 // 30% weight to new score
	return int(float64(currentScore)*(1-weight) + float64(newScore)*weight)
}

func (s *Service) calculateMasteryLevel(averageScore, practiceCount int) int {
	if practiceCount < 3 {
		return 0 // Need minimum practice
	}

	switch {
	case averageScore >= 95:
		return 5 // Expert
	case averageScore >= 85:
		return 4 // Advanced
	case averageScore >= 75:
		return 3 // Intermediate
	case averageScore >= 60:
		return 2 // Beginner
	case averageScore >= 40:
		return 1 // Novice
	default:
		return 0 // Needs work
	}
}

func (s *Service) generatePhonemeRecommendations(userID string, languageID int, userProgress []UserPhoneticProgress) []PhonemeRecommendation {
	recommendations := make([]PhonemeRecommendation, 0)

	// Get all phonemes for language
	var allPhonemes []Phoneme
	s.db.Where("language_id = ?", languageID).Find(&allPhonemes)

	// Create map of practiced phonemes
	practicedMap := make(map[int]UserPhoneticProgress)
	for _, progress := range userProgress {
		practicedMap[progress.PhonemeID] = progress
	}

	// Recommend unpracticed phonemes (prioritize easy ones first)
	for _, phoneme := range allPhonemes {
		if _, practiced := practicedMap[phoneme.ID]; !practiced {
			recommendations = append(recommendations, PhonemeRecommendation{
				Phoneme:       phoneme,
				Reason:        "New phoneme to learn",
				Priority:      5 - phoneme.Difficulty, // Higher priority for easier phonemes
				EstimatedTime: phoneme.Difficulty * 5, // 5-25 minutes based on difficulty
			})
		}
	}

	// Recommend weak phonemes for review
	for _, progress := range userProgress {
		averageScore := float64(progress.DiscriminationScore+progress.ProductionScore) / 2
		if averageScore < 70 && progress.PracticeCount > 2 {
			recommendations = append(recommendations, PhonemeRecommendation{
				Phoneme:       progress.Phoneme,
				Reason:        fmt.Sprintf("Needs improvement (%.1f%% accuracy)", averageScore),
				Priority:      4, // High priority for weak phonemes
				EstimatedTime: 10,
			})
		}
	}

	// Sort by priority (highest first)
	for i := 0; i < len(recommendations)-1; i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[i].Priority < recommendations[j].Priority {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}

	// Limit to top 5 recommendations
	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}

	return recommendations
}

func (s *Service) calculateExerciseScore(responses, exerciseData, exerciseType string) int {
	// This is a simplified scoring system
	// In a real implementation, you'd parse the exercise data and responses
	// to calculate an accurate score based on the exercise type

	switch exerciseType {
	case "discrimination":
		return s.calculateDiscriminationScore(responses, exerciseData)
	case "production":
		return s.calculateProductionScore(responses, exerciseData)
	case "minimal_pairs":
		return s.calculateMinimalPairsScore(responses, exerciseData)
	default:
		return 50 // Default score
	}
}

func (s *Service) calculateDiscriminationScore(responses, exerciseData string) int {
	// Parse responses and exercise data to calculate accuracy
	// This is a placeholder implementation
	return 75 // Example score
}

func (s *Service) calculateProductionScore(responses, exerciseData string) int {
	// For production exercises, you might analyze audio quality, pronunciation accuracy, etc.
	// This is a placeholder implementation
	return 80 // Example score
}

func (s *Service) calculateMinimalPairsScore(responses, exerciseData string) int {
	// Calculate accuracy for minimal pairs identification
	// This is a placeholder implementation
	return 85 // Example score
}

func (s *Service) generateExerciseFeedback(score int, exerciseType string) string {
	feedback := make(map[string]interface{})

	switch {
	case score >= 90:
		feedback["level"] = "excellent"
		feedback["message"] = "Outstanding performance! You've mastered this exercise."
	case score >= 80:
		feedback["level"] = "good"
		feedback["message"] = "Good work! You're making solid progress."
	case score >= 70:
		feedback["level"] = "fair"
		feedback["message"] = "Fair performance. Keep practicing to improve."
	case score >= 60:
		feedback["level"] = "needs_improvement"
		feedback["message"] = "This needs more practice. Focus on the difficult sounds."
	default:
		feedback["level"] = "poor"
		feedback["message"] = "Don't give up! This is challenging but you'll improve with practice."
	}

	// Add exercise-specific tips
	switch exerciseType {
	case "discrimination":
		feedback["tip"] = "Focus on listening to the subtle differences between sounds."
	case "production":
		feedback["tip"] = "Pay attention to tongue and lip position when making sounds."
	case "minimal_pairs":
		feedback["tip"] = "Listen for the contrasting sounds that change word meaning."
	}

	feedbackJSON, _ := json.Marshal(feedback)
	return string(feedbackJSON)
}

func (s *Service) getPhonemesByPerformance(userID string, languageID int) ([]PhonemeWithScore, []PhonemeWithScore) {
	var results []struct {
		Phoneme             Phoneme
		DiscriminationScore int
		ProductionScore     int
		PracticeCount       int
	}

	s.db.Table("user_phonetic_progress upp").
		Select("phonemes.*, upp.discrimination_score, upp.production_score, upp.practice_count").
		Joins("JOIN phonemes ON upp.phoneme_id = phonemes.id").
		Where("upp.user_id = ? AND phonemes.language_id = ? AND upp.practice_count > 2", userID, languageID).
		Scan(&results)

	weakest := make([]PhonemeWithScore, 0)
	strongest := make([]PhonemeWithScore, 0)

	for _, result := range results {
		averageScore := float64(result.DiscriminationScore+result.ProductionScore) / 2

		phonemeWithScore := PhonemeWithScore{
			Phoneme: result.Phoneme,
			Score:   averageScore,
		}

		if averageScore < 70 {
			weakest = append(weakest, phonemeWithScore)
		} else if averageScore >= 85 {
			strongest = append(strongest, phonemeWithScore)
		}
	}

	// Sort weakest (lowest scores first)
	for i := 0; i < len(weakest)-1; i++ {
		for j := i + 1; j < len(weakest); j++ {
			if weakest[i].Score > weakest[j].Score {
				weakest[i], weakest[j] = weakest[j], weakest[i]
			}
		}
	}

	// Sort strongest (highest scores first)
	for i := 0; i < len(strongest)-1; i++ {
		for j := i + 1; j < len(strongest); j++ {
			if strongest[i].Score < strongest[j].Score {
				strongest[i], strongest[j] = strongest[j], strongest[i]
			}
		}
	}

	// Limit to top 5 each
	if len(weakest) > 5 {
		weakest = weakest[:5]
	}
	if len(strongest) > 5 {
		strongest = strongest[:5]
	}

	return weakest, strongest
}

func (s *Service) getProgressByCategory(userID string, languageID int) map[string]CategoryProgress {
	var results []struct {
		Category          string
		TotalPhonemes     int
		MasteredCount     int
		AvgDiscrimination float64
		AvgProduction     float64
	}

	s.db.Raw(`
       SELECT 
           p.category,
           COUNT(p.id) as total_phonemes,
           COUNT(CASE WHEN upp.mastery_level >= 4 THEN 1 END) as mastered_count,
           AVG(upp.discrimination_score) as avg_discrimination,
           AVG(upp.production_score) as avg_production
       FROM phonemes p
       LEFT JOIN user_phonetic_progress upp ON p.id = upp.phoneme_id AND upp.user_id = ?
       WHERE p.language_id = ?
       GROUP BY p.category
   `, userID, languageID).Scan(&results)

	progress := make(map[string]CategoryProgress)
	for _, result := range results {
		averageScore := (result.AvgDiscrimination + result.AvgProduction) / 2

		progress[result.Category] = CategoryProgress{
			Category:      result.Category,
			TotalPhonemes: result.TotalPhonemes,
			MasteredCount: result.MasteredCount,
			AverageScore:  averageScore,
			TimeSpent:     0, // Would need to calculate from exercise sessions
		}
	}

	return progress
}

func (s *Service) getWeeklyPhoneticProgress(userID string, languageID int, days int) []WeeklyPhoneticProgress {
	var results []WeeklyPhoneticProgress

	query := `
       SELECT 
           DATE_TRUNC('week', ues.completed_at) as week_start,
           COUNT(ues.id) as sessions_count,
           SUM(ues.duration) / 60 as total_time,
           AVG(ues.score) as average_score
       FROM user_exercise_sessions ues
       JOIN phonetic_exercises pe ON ues.exercise_id = pe.id
       WHERE ues.user_id = ? AND pe.language_id = ? 
       AND ues.completed_at >= ?
       GROUP BY DATE_TRUNC('week', ues.completed_at)
       ORDER BY week_start DESC
   `

	startDate := time.Now().AddDate(0, 0, -days)
	s.db.Raw(query, userID, languageID, startDate).Scan(&results)

	// Format dates and add additional calculated fields
	for i := range results {
		// Calculate week end date (6 days after start)
		weekStart, _ := time.Parse("2006-01-02", results[i].WeekStart)
		weekEnd := weekStart.AddDate(0, 0, 6)
		results[i].WeekEnd = weekEnd.Format("2006-01-02")

		// These would need more complex queries to calculate accurately
		results[i].NewPhonemes = 0
		results[i].PerfectedPhonemes = 0
	}

	return results
}

// GetExerciseProgress returns user's overall exercise progress
func (s *Service) GetExerciseProgress(ctx context.Context, userID string, languageID int) (*ExerciseProgressResponse, error) {
	// Get total exercises for language
	var totalExercises int64
	s.db.Model(&PhoneticExercise{}).
		Where("language_id = ? AND is_active = ?", languageID, true).
		Count(&totalExercises)

	// Get completed exercises
	var completedExercises int64
	s.db.Model(&UserExerciseSession{}).
		Joins("JOIN phonetic_exercises ON user_exercise_sessions.exercise_id = phonetic_exercises.id").
		Where("user_exercise_sessions.user_id = ? AND phonetic_exercises.language_id = ?", userID, languageID).
		Count(&completedExercises)

	// Get average score and total time
	var stats struct {
		AverageScore float64
		TotalTime    int64
	}

	s.db.Model(&UserExerciseSession{}).
		Select("AVG(score) as average_score, SUM(duration) as total_time").
		Joins("JOIN phonetic_exercises ON user_exercise_sessions.exercise_id = phonetic_exercises.id").
		Where("user_exercise_sessions.user_id = ? AND phonetic_exercises.language_id = ?", userID, languageID).
		Scan(&stats)

	// Get last practice date
	var lastPracticeDate *time.Time
	s.db.Model(&UserExerciseSession{}).
		Select("MAX(completed_at)").
		Joins("JOIN phonetic_exercises ON user_exercise_sessions.exercise_id = phonetic_exercises.id").
		Where("user_exercise_sessions.user_id = ? AND phonetic_exercises.language_id = ?", userID, languageID).
		Scan(&lastPracticeDate)

	// Calculate streak days (simplified)
	streakDays := 0
	if lastPracticeDate != nil && time.Since(*lastPracticeDate).Hours() < 48 {
		streakDays = s.calculatePhoneticStreakDays(userID, languageID)
	}

	return &ExerciseProgressResponse{
		TotalExercises:     int(totalExercises),
		CompletedExercises: int(completedExercises),
		AverageScore:       stats.AverageScore,
		TotalTime:          int(stats.TotalTime),
		StreakDays:         streakDays,
		LastPracticeDate:   lastPracticeDate,
	}, nil
}

func (s *Service) calculatePhoneticStreakDays(userID string, languageID int) int {
	// Simplified streak calculation
	// In reality, you'd check for consecutive days of activity
	var activeDays int64
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	s.db.Model(&UserExerciseSession{}).
		Joins("JOIN phonetic_exercises ON user_exercise_sessions.exercise_id = phonetic_exercises.id").
		Where("user_exercise_sessions.user_id = ? AND phonetic_exercises.language_id = ? AND user_exercise_sessions.completed_at >= ?",
			userID, languageID, sevenDaysAgo).
		Select("COUNT(DISTINCT DATE(completed_at))").
		Scan(&activeDays)

	return int(activeDays)
}
