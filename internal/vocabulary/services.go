package vocabulary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db        *gorm.DB
	jwtSecret string
	srsConfig SRSConfig
}

func NewService(db *gorm.DB, jwtSecret string) *Service {
	return &Service{
		db:        db,
		jwtSecret: jwtSecret,
		srsConfig: SRSConfig{
			EasyBonus:        1.3,
			HardPenalty:      0.85,
			FailurePenalty:   0.2,
			MinEaseFactor:    1.3,
			MaxEaseFactor:    2.5,
			GraduationSteps:  []int{1, 6},
			NewWordsPerDay:   20,
			MaxReviewsPerDay: 200,
		},
	}
}

func (s *Service) AddVocabulary(ctx context.Context, userID string, languageID int, req AddVocabularyRequest) (*UserVocabulary, error) {
	// Check if word already exists in global vocabulary
	var existingVocab Vocabulary
	err := s.db.Where("word = ? AND language_id = ?", req.Word, languageID).First(&existingVocab).Error

	var vocabID string
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new vocabulary entry
		newVocab := Vocabulary{
			Word:                  req.Word,
			LanguageID:            languageID,
			Translation:           req.Translation,
			PhoneticTranscription: req.PhoneticTranscription,
			Definition:            req.Definition,
			ExampleSentence:       req.ExampleSentence,
			DifficultyLevel:       req.DifficultyLevel,
			CreatedBy:             userID,
		}

		if err := s.db.Create(&newVocab).Error; err != nil {
			return nil, err
		}
		vocabID = newVocab.ID
	} else if err != nil {
		return nil, err
	} else {
		vocabID = existingVocab.ID
	}

	// Check if user already has this word
	var existingUserVocab UserVocabulary
	err = s.db.Where("user_id = ? AND vocabulary_id = ?", userID, vocabID).First(&existingUserVocab).Error
	if err == nil {
		return nil, errors.New("word already exists in your vocabulary")
	}

	// Create user vocabulary entry
	nextReview := time.Now().Add(time.Hour * 24) // Review tomorrow
	userVocab := UserVocabulary{
		UserID:          userID,
		VocabularyID:    vocabID,
		ContextSentence: req.ContextSentence,
		PersonalNote:    req.PersonalNote,
		SourceContentID: &req.SourceContentID,
		NextReviewAt:    &nextReview,
		EaseFactor:      s.srsConfig.MaxEaseFactor,
		IntervalDays:    1,
	}

	if err := s.db.Create(&userVocab).Error; err != nil {
		return nil, err
	}

	// Load the vocabulary relation
	s.db.Preload("Vocabulary").Where("id = ?", userVocab.ID).First(&userVocab)

	return &userVocab, nil
}

func (s *Service) GetVocabularyForReview(ctx context.Context, userID string, languageID int, limit int) ([]UserVocabulary, error) {
	var vocab []UserVocabulary

	err := s.db.Preload("Vocabulary").
		Joins("JOIN vocabulary ON user_vocabulary.vocabulary_id = vocabulary.id").
		Where("user_vocabulary.user_id = ? AND vocabulary.language_id = ?", userID, languageID).
		Where("user_vocabulary.next_review_at IS NULL OR user_vocabulary.next_review_at <= ?", time.Now()).
		Order("user_vocabulary.next_review_at ASC").
		Limit(limit).
		Find(&vocab).Error

	return vocab, err
}

func (s *Service) ReviewVocabulary(ctx context.Context, userID string, req ReviewRequest) (*UserVocabulary, error) {
	var userVocab UserVocabulary
	err := s.db.Where("user_id = ? AND vocabulary_id = ?", userID, req.VocabularyID).First(&userVocab).Error
	if err != nil {
		return nil, errors.New("vocabulary not found")
	}

	// Update review statistics
	userVocab.ReviewCount++
	if req.Correct {
		userVocab.CorrectCount++
	}
	now := time.Now()
	userVocab.LastReviewedAt = &now

	// Calculate next review using user's SRS config
	s.calculateNextReview(&userVocab, req.Correct, userID) // ← PASS userID

	if err := s.db.Save(&userVocab).Error; err != nil {
		return nil, err
	}

	return &userVocab, nil
}

func (s *Service) calculateNextReview(userVocab *UserVocabulary, correct bool, userID string) {
	// Get user's SRS config
	config, err := s.getUserSRSConfig(context.Background(), userID)
	if err != nil {
		config = &s.srsConfig // Fallback to default
	}

	if correct {
		// Increase interval and potentially ease factor
		if userVocab.MasteryLevel < 2 {
			// Still learning - use graduation steps
			if userVocab.MasteryLevel < len(config.GraduationSteps) {
				userVocab.IntervalDays = config.GraduationSteps[userVocab.MasteryLevel]
			} else {
				userVocab.IntervalDays = int(float64(userVocab.IntervalDays) * userVocab.EaseFactor)
			}
			userVocab.MasteryLevel++
		} else {
			// Mature card - use ease factor
			userVocab.IntervalDays = int(float64(userVocab.IntervalDays) * userVocab.EaseFactor)
			userVocab.EaseFactor += 0.1
			if userVocab.EaseFactor > config.MaxEaseFactor {
				userVocab.EaseFactor = config.MaxEaseFactor
			}
		}
	} else {
		// Wrong answer - reset to learning
		userVocab.MasteryLevel = 0
		userVocab.IntervalDays = 1
		userVocab.EaseFactor *= config.FailurePenalty
		if userVocab.EaseFactor < config.MinEaseFactor {
			userVocab.EaseFactor = config.MinEaseFactor
		}
	}

	// Set next review date
	nextReview := time.Now().Add(time.Duration(userVocab.IntervalDays) * time.Hour * 24)
	userVocab.NextReviewAt = &nextReview
}

func (s *Service) GetVocabularyStats(ctx context.Context, userID string, languageID int) (*VocabularyStats, error) {
	stats := &VocabularyStats{}

	baseQuery := s.db.Table("user_vocabulary uv").
		Joins("JOIN vocabulary v ON uv.vocabulary_id = v.id").
		Where("uv.user_id = ? AND v.language_id = ?", userID, languageID)

	// Total words
	baseQuery.Count(&stats.TotalWords)

	// Reviews due
	baseQuery.Where("uv.next_review_at IS NULL OR uv.next_review_at <= ?", time.Now()).
		Count(&stats.ReviewsDue)

	// New words (never reviewed)
	baseQuery.Where("uv.review_count = 0").Count(&stats.NewWords)

	// Learning words (mastery level 0-2)
	baseQuery.Where("uv.mastery_level BETWEEN 0 AND 2").Count(&stats.LearningWords)

	// Mature words (mastery level > 2)
	baseQuery.Where("uv.mastery_level > 2").Count(&stats.MatureWords)

	// Reviews today
	today := time.Now().Format("2006-01-02")
	baseQuery.Where("DATE(uv.last_reviewed_at) = ?", today).Count(&stats.ReviewsToday)

	// Accuracy rate
	var totalReviews, correctReviews int64
	baseQuery.Select("SUM(uv.review_count)").Scan(&totalReviews)
	baseQuery.Select("SUM(uv.correct_count)").Scan(&correctReviews)

	if totalReviews > 0 {
		stats.AccuracyRate = float64(correctReviews) / float64(totalReviews) * 100
	}

	return stats, nil
}

func (s *Service) GetUserVocabulary(ctx context.Context, userID string, languageID int, limit, offset int) ([]UserVocabulary, int64, error) {
	var vocab []UserVocabulary
	var total int64

	query := s.db.Preload("Vocabulary").
		Joins("JOIN vocabulary ON user_vocabulary.vocabulary_id = vocabulary.id").
		Where("user_vocabulary.user_id = ? AND vocabulary.language_id = ?", userID, languageID)

	query.Count(&total)

	err := query.Order("user_vocabulary.added_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&vocab).Error

	return vocab, total, err
}

func (s *Service) DeleteVocabulary(ctx context.Context, userID, vocabularyID string) error {
	result := s.db.Where("user_id = ? AND vocabulary_id = ?", userID, vocabularyID).Delete(&UserVocabulary{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("vocabulary not found")
	}
	return nil
}

func (s *Service) SearchVocabulary(ctx context.Context, userID string, languageID int, query string, limit int) ([]UserVocabulary, error) {
	var vocab []UserVocabulary

	searchQuery := "%" + query + "%"
	err := s.db.Preload("Vocabulary").
		Joins("JOIN vocabulary ON user_vocabulary.vocabulary_id = vocabulary.id").
		Where("user_vocabulary.user_id = ? AND vocabulary.language_id = ?", userID, languageID).
		Where("vocabulary.word ILIKE ? OR vocabulary.translation ILIKE ? OR vocabulary.definition ILIKE ?",
			searchQuery, searchQuery, searchQuery).
		Limit(limit).
		Find(&vocab).Error

	return vocab, err
}

// UpdateVocabulary updates user vocabulary
func (s *Service) UpdateVocabulary(ctx context.Context, userID, vocabularyID string, req UpdateVocabularyRequest) (*UserVocabulary, error) {
	var userVocab UserVocabulary
	err := s.db.Preload("Vocabulary").Where("user_id = ? AND vocabulary_id = ?", userID, vocabularyID).First(&userVocab).Error
	if err != nil {
		return nil, errors.New("vocabulary not found")
	}

	// Update user-specific fields
	if req.ContextSentence != "" {
		userVocab.ContextSentence = req.ContextSentence
	}
	if req.PersonalNote != "" {
		userVocab.PersonalNote = req.PersonalNote
	}

	// Update global vocabulary if user is the creator
	var vocab Vocabulary
	if err := s.db.Where("id = ?", vocabularyID).First(&vocab).Error; err == nil {
		if vocab.CreatedBy == userID {
			if req.Translation != "" {
				vocab.Translation = req.Translation
			}
			if req.PhoneticTranscription != "" {
				vocab.PhoneticTranscription = req.PhoneticTranscription
			}
			if req.Definition != "" {
				vocab.Definition = req.Definition
			}
			if req.ExampleSentence != "" {
				vocab.ExampleSentence = req.ExampleSentence
			}
			s.db.Save(&vocab)
		}
	}

	if err := s.db.Save(&userVocab).Error; err != nil {
		return nil, err
	}

	// Reload with vocabulary data
	s.db.Preload("Vocabulary").Where("id = ?", userVocab.ID).First(&userVocab)
	return &userVocab, nil
}

// BatchReviewVocabulary processes multiple vocabulary reviews at once
func (s *Service) BatchReviewVocabulary(ctx context.Context, userID string, req BatchReviewRequest) (*BatchReviewResult, error) {
	result := &BatchReviewResult{
		Results: make([]ReviewVocabularyResult, 0, len(req.Reviews)),
		Errors:  make([]string, 0),
	}

	for _, review := range req.Reviews {
		userVocab, err := s.ReviewVocabulary(ctx, userID, review)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error reviewing %s: %v", review.VocabularyID, err))
			result.Results = append(result.Results, ReviewVocabularyResult{
				VocabularyID: review.VocabularyID,
				Success:      false,
				Error:        err.Error(),
			})
		} else {
			nextReview := ""
			if userVocab.NextReviewAt != nil {
				nextReview = userVocab.NextReviewAt.Format(time.RFC3339)
			}
			result.Results = append(result.Results, ReviewVocabularyResult{
				VocabularyID: review.VocabularyID,
				Success:      true,
				NextReview:   nextReview,
			})
			result.Processed++
		}
	}

	return result, nil
}

// GetVocabularyProgress returns vocabulary learning progress over time
func (s *Service) GetVocabularyProgress(ctx context.Context, userID string, languageID int, days int) (*VocabularyProgress, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	// Get overall stats
	stats, err := s.GetVocabularyStats(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}

	// Get daily progress
	dailyProgress := make([]DailyVocabularyProgress, 0)

	query := `
        SELECT 
            DATE(uv.added_at) as date,
            COUNT(CASE WHEN uv.added_at >= ? AND uv.added_at <= ? THEN 1 END) as words_added,
            COUNT(CASE WHEN uv.last_reviewed_at >= ? AND uv.last_reviewed_at <= ? THEN 1 END) as words_reviewed,
            AVG(CASE WHEN uv.last_reviewed_at >= ? AND uv.last_reviewed_at <= ? 
                     THEN CAST(uv.correct_count AS FLOAT) / NULLIF(uv.review_count, 0) * 100 
                     ELSE NULL END) as accuracy_rate
        FROM user_vocabulary uv
        JOIN vocabulary v ON uv.vocabulary_id = v.id
        WHERE uv.user_id = ? AND v.language_id = ?
        AND (DATE(uv.added_at) BETWEEN ? AND ? OR DATE(uv.last_reviewed_at) BETWEEN ? AND ?)
        GROUP BY DATE(uv.added_at)
        ORDER BY date DESC
    `

	err = s.db.Raw(query,
		startDate, endDate, // words_added
		startDate, endDate, // words_reviewed
		startDate, endDate, // accuracy_rate
		userID, languageID,
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"),
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"),
	).Scan(&dailyProgress).Error
	if err != nil {
		return nil, err
	}

	// Get mastery breakdown
	masteryBreakdown := make(map[string]int64)
	masteryQuery := `
        SELECT 
            CASE 
                WHEN uv.mastery_level = 0 THEN 'new'
                WHEN uv.mastery_level BETWEEN 1 AND 2 THEN 'learning'
                WHEN uv.mastery_level BETWEEN 3 AND 5 THEN 'young'
                ELSE 'mature'
            END as category,
            COUNT(*) as count
        FROM user_vocabulary uv
        JOIN vocabulary v ON uv.vocabulary_id = v.id
        WHERE uv.user_id = ? AND v.language_id = ?
        GROUP BY category
    `

	var masteryResults []struct {
		Category string `json:"category"`
		Count    int64  `json:"count"`
	}

	err = s.db.Raw(masteryQuery, userID, languageID).Scan(&masteryResults).Error
	if err != nil {
		return nil, err
	}

	for _, result := range masteryResults {
		masteryBreakdown[result.Category] = result.Count
	}

	return &VocabularyProgress{
		Period:           fmt.Sprintf("%d days", days),
		TotalWords:       stats.TotalWords,
		WordsLearned:     stats.LearningWords + stats.MatureWords,
		WordsReviewed:    int64(stats.ReviewsToday), // This could be calculated better
		AccuracyRate:     stats.AccuracyRate,
		DailyProgress:    dailyProgress,
		MasteryBreakdown: masteryBreakdown,
	}, nil
}

// FilterVocabulary filters vocabulary based on criteria
func (s *Service) FilterVocabulary(ctx context.Context, userID string, filter VocabularyFilter) ([]UserVocabulary, int64, error) {
	var vocab []UserVocabulary
	var total int64

	query := s.db.Preload("Vocabulary").
		Joins("JOIN vocabulary ON user_vocabulary.vocabulary_id = vocabulary.id").
		Where("user_vocabulary.user_id = ? AND vocabulary.language_id = ?", userID, filter.LanguageID)

	// Apply filters
	if len(filter.MasteryLevels) > 0 {
		query = query.Where("user_vocabulary.mastery_level IN ?", filter.MasteryLevels)
	}

	if filter.SearchQuery != "" {
		searchTerm := "%" + strings.ToLower(filter.SearchQuery) + "%"
		query = query.Where(
			"LOWER(vocabulary.word) LIKE ? OR LOWER(vocabulary.translation) LIKE ? OR LOWER(vocabulary.definition) LIKE ? OR LOWER(user_vocabulary.personal_note) LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm,
		)
	}

	// Count total
	query.Model(&UserVocabulary{}).Count(&total)

	// Apply sorting
	sortBy := "user_vocabulary.added_at"
	if filter.SortBy != "" {
		switch filter.SortBy {
		case "word":
			sortBy = "vocabulary.word"
		case "mastery_level":
			sortBy = "user_vocabulary.mastery_level"
		case "next_review":
			sortBy = "user_vocabulary.next_review_at"
		case "added_at":
			sortBy = "user_vocabulary.added_at"
		}
	}

	sortDirection := "DESC"
	if filter.SortDirection == "asc" {
		sortDirection = "ASC"
	}

	query = query.Order(sortBy + " " + sortDirection)

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(20) // Default limit
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	err := query.Find(&vocab).Error
	return vocab, total, err
}

// ImportVocabulary imports vocabulary from various formats
func (s *Service) ImportVocabulary(ctx context.Context, userID string, req ImportVocabularyRequest) (*ImportResult, error) {
	result := &ImportResult{
		Errors: make([]string, 0),
	}

	var vocabularyItems []AddVocabularyRequest

	// Parse data based on format
	switch req.Format {
	case "json":
		err := json.Unmarshal([]byte(req.Data), &vocabularyItems)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON format: %v", err)
		}
	case "csv":
		vocabularyItems, err := s.parseCSVData(req.Data)
		if err != nil {
			return nil, fmt.Errorf("invalid CSV format: %v", err)
		}
		_ = vocabularyItems // Use the parsed data
	case "anki":
		return nil, errors.New("anki format not implemented yet")
	default:
		return nil, errors.New("unsupported format")
	}

	result.Total = len(vocabularyItems)

	// Import each vocabulary item
	for _, item := range vocabularyItems {
		// Check if word already exists
		if req.Options.SkipDuplicates {
			var existing UserVocabulary
			err := s.db.Joins("JOIN vocabulary ON user_vocabulary.vocabulary_id = vocabulary.id").
				Where("user_vocabulary.user_id = ? AND vocabulary.word = ? AND vocabulary.language_id = ?",
					userID, item.Word, req.LanguageID).
				First(&existing).Error
			if err == nil {
				result.Skipped++
				continue
			}
		}

		_, err := s.AddVocabulary(ctx, userID, req.LanguageID, item)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing '%s': %v", item.Word, err))
		} else {
			result.Imported++
		}
	}

	return result, nil
}

// ExportVocabulary exports user vocabulary in specified format
func (s *Service) ExportVocabulary(ctx context.Context, userID string, languageID int, format string) (string, error) {
	vocab, _, err := s.GetUserVocabulary(ctx, userID, languageID, 10000, 0) // Get all
	if err != nil {
		return "", err
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(vocab, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "csv":
		return s.exportToCSV(vocab), nil
	default:
		return "", errors.New("unsupported export format")
	}
}

// GetSRSConfig returns user's SRS configuration
func (s *Service) GetSRSConfig(ctx context.Context, userID string) (*SRSConfigResponse, error) {
	var userConfig UserSRSConfig
	err := s.db.Where("user_id = ?", userID).First(&userConfig).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create default config for user
		userConfig = UserSRSConfig{
			UserID:           userID,
			EasyBonus:        1.3,
			HardPenalty:      0.85,
			FailurePenalty:   0.2,
			MinEaseFactor:    1.3,
			MaxEaseFactor:    2.5,
			GraduationSteps:  "1,6",
			NewWordsPerDay:   20,
			MaxReviewsPerDay: 200,
		}

		if err := s.db.Create(&userConfig).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Get user statistics
	statistics, err := s.getSRSStatistics(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &SRSConfigResponse{
		CurrentConfig: userConfig.ToSRSConfig(),
		Presets:       s.getSRSPresets(),
		Statistics:    *statistics,
	}, nil
}

// UpdateSRSConfig updates user's SRS configuration
func (s *Service) UpdateSRSConfig(ctx context.Context, userID string, configReq UpdateSRSConfigRequest) error {
	// Validate the request
	if err := configReq.Validate(); err != nil {
		return err
	}

	var userConfig UserSRSConfig
	err := s.db.Where("user_id = ?", userID).First(&userConfig).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new config
		userConfig = UserSRSConfig{
			UserID: userID,
		}
	} else if err != nil {
		return err
	}

	// Update fields
	userConfig.EasyBonus = configReq.EasyBonus
	userConfig.HardPenalty = configReq.HardPenalty
	userConfig.FailurePenalty = configReq.FailurePenalty
	userConfig.MinEaseFactor = configReq.MinEaseFactor
	userConfig.MaxEaseFactor = configReq.MaxEaseFactor
	userConfig.SetGraduationSteps(configReq.GraduationSteps)
	userConfig.NewWordsPerDay = configReq.NewWordsPerDay
	userConfig.MaxReviewsPerDay = configReq.MaxReviewsPerDay

	// Save to database
	if userConfig.ID == "" {
		return s.db.Create(&userConfig).Error
	} else {
		return s.db.Save(&userConfig).Error
	}
}

// Helper methods
// ADD new helper method to get user's SRS config
func (s *Service) getUserSRSConfig(ctx context.Context, userID string) (*SRSConfig, error) {
	var userConfig UserSRSConfig
	err := s.db.Where("user_id = ?", userID).First(&userConfig).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Return default config
		return &s.srsConfig, nil
	} else if err != nil {
		return nil, err
	}

	config := userConfig.ToSRSConfig()
	return &config, nil
}

// parseCSVData parses CSV format vocabulary data
func (s *Service) parseCSVData(data string) ([]AddVocabularyRequest, error) {
	lines := strings.Split(data, "\n")
	if len(lines) < 2 {
		return nil, errors.New("CSV must have at least a header and one data row")
	}

	// Parse header
	header := strings.Split(lines[0], ",")
	var wordCol, translationCol, definitionCol, exampleCol int = -1, -1, -1, -1

	for i, col := range header {
		col = strings.TrimSpace(strings.ToLower(col))
		switch col {
		case "word", "front":
			wordCol = i
		case "translation", "back", "meaning":
			translationCol = i
		case "definition", "def":
			definitionCol = i
		case "example", "example_sentence":
			exampleCol = i
		}
	}

	if wordCol == -1 || translationCol == -1 {
		return nil, errors.New("CSV must have 'word' and 'translation' columns")
	}

	var vocabulary []AddVocabularyRequest
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) <= wordCol || len(fields) <= translationCol {
			continue
		}

		item := AddVocabularyRequest{
			Word:        strings.TrimSpace(fields[wordCol]),
			Translation: strings.TrimSpace(fields[translationCol]),
		}

		if definitionCol != -1 && len(fields) > definitionCol {
			item.Definition = strings.TrimSpace(fields[definitionCol])
		}

		if exampleCol != -1 && len(fields) > exampleCol {
			item.ExampleSentence = strings.TrimSpace(fields[exampleCol])
		}

		vocabulary = append(vocabulary, item)
	}

	return vocabulary, nil
}

// exportToCSV converts vocabulary to CSV format
func (s *Service) exportToCSV(vocab []UserVocabulary) string {
	var lines []string
	lines = append(lines, "word,translation,phonetic,definition,example_sentence,context_sentence,personal_note,mastery_level,review_count,correct_count,added_at")

	for _, v := range vocab {
		line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%d,%d,%d,%s",
			escapeCsvField(v.Vocabulary.Word),
			escapeCsvField(v.Vocabulary.Translation),
			escapeCsvField(v.Vocabulary.PhoneticTranscription),
			escapeCsvField(v.Vocabulary.Definition),
			escapeCsvField(v.Vocabulary.ExampleSentence),
			escapeCsvField(v.ContextSentence),
			escapeCsvField(v.PersonalNote),
			v.MasteryLevel,
			v.ReviewCount,
			v.CorrectCount,
			v.AddedAt.Format("2006-01-02 15:04:05"),
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// escapeCsvField escapes CSV field if it contains commas or quotes
func escapeCsvField(field string) string {
	if strings.Contains(field, ",") || strings.Contains(field, "\"") || strings.Contains(field, "\n") {
		field = strings.ReplaceAll(field, "\"", "\"\"")
		return "\"" + field + "\""
	}
	return field
}

// getSRSStatistics calculates statistics about user's SRS performance
func (s *Service) getSRSStatistics(ctx context.Context, userID string) (*SRSStatistics, error) {
	stats := &SRSStatistics{}

	// Calculate average retention (accuracy rate)
	var totalReviews, correctReviews int64
	s.db.Model(&UserVocabulary{}).
		Where("user_id = ? AND review_count > 0", userID).
		Select("SUM(review_count)").Scan(&totalReviews)

	s.db.Model(&UserVocabulary{}).
		Where("user_id = ? AND review_count > 0", userID).
		Select("SUM(correct_count)").Scan(&correctReviews)

	if totalReviews > 0 {
		stats.AverageRetention = float64(correctReviews) / float64(totalReviews) * 100
	}

	stats.TotalReviews = totalReviews

	// Calculate average interval for mature cards
	var avgInterval float64
	s.db.Model(&UserVocabulary{}).
		Where("user_id = ? AND mastery_level > 2", userID).
		Select("AVG(interval_days)").Scan(&avgInterval)
	stats.AverageInterval = avgInterval

	// Calculate reviews per day (last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var reviewsLast30Days int64
	s.db.Model(&UserVocabulary{}).
		Where("user_id = ? AND last_reviewed_at >= ?", userID, thirtyDaysAgo).
		Select("SUM(review_count)").Scan(&reviewsLast30Days)
	stats.ReviewsPerDay = float64(reviewsLast30Days) / 30

	// Calculate streak days (simplified)
	var lastReviewDate time.Time
	s.db.Model(&UserVocabulary{}).
		Where("user_id = ?", userID).
		Select("MAX(last_reviewed_at)").Scan(&lastReviewDate)

	if !lastReviewDate.IsZero() {
		daysSinceLastReview := int(time.Since(lastReviewDate).Hours() / 24)
		if daysSinceLastReview <= 1 {
			stats.StreakDays = s.calculateStreakDays(userID)
		}
	}

	// Estimate time to maturity (average for new words to reach mastery level 6)
	stats.TimeToMaturity = 45 // Simplified estimate in days

	return stats, nil
}

// calculateStreakDays calculates consecutive days of vocabulary reviews
func (s *Service) calculateStreakDays(userID string) int {
	// Simplified implementation - in reality you'd check daily activity
	var activeDays int64
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	s.db.Model(&UserVocabulary{}).
		Where("user_id = ? AND last_reviewed_at >= ?", userID, sevenDaysAgo).
		Select("COUNT(DISTINCT DATE(last_reviewed_at))").Scan(&activeDays)

	return int(activeDays)
}

// getSRSPresets returns predefined SRS configurations
func (s *Service) getSRSPresets() []SRSConfigPreset {
	return []SRSConfigPreset{
		{
			Name:             "Conservative",
			Description:      "Lower penalties, longer intervals. Good for beginners.",
			EasyBonus:        1.2,
			HardPenalty:      0.9,
			FailurePenalty:   0.5,
			MinEaseFactor:    1.3,
			MaxEaseFactor:    2.2,
			GraduationSteps:  []int{1, 3, 6},
			NewWordsPerDay:   15,
			MaxReviewsPerDay: 150,
		},
		{
			Name:             "Default",
			Description:      "Balanced settings based on Anki defaults.",
			EasyBonus:        1.3,
			HardPenalty:      0.85,
			FailurePenalty:   0.2,
			MinEaseFactor:    1.3,
			MaxEaseFactor:    2.5,
			GraduationSteps:  []int{1, 6},
			NewWordsPerDay:   20,
			MaxReviewsPerDay: 200,
		},
		{
			Name:             "Aggressive",
			Description:      "Higher penalties, faster intervals. For experienced learners.",
			EasyBonus:        1.5,
			HardPenalty:      0.8,
			FailurePenalty:   0.1,
			MinEaseFactor:    1.2,
			MaxEaseFactor:    3.0,
			GraduationSteps:  []int{1, 4},
			NewWordsPerDay:   30,
			MaxReviewsPerDay: 300,
		},
		{
			Name:             "Exam Prep",
			Description:      "Intensive settings for exam preparation.",
			EasyBonus:        1.2,
			HardPenalty:      0.85,
			FailurePenalty:   0.3,
			MinEaseFactor:    1.4,
			MaxEaseFactor:    2.0,
			GraduationSteps:  []int{1, 2, 4},
			NewWordsPerDay:   40,
			MaxReviewsPerDay: 500,
		},
	}
}

func (s *Service) ApplySRSPreset(ctx context.Context, userID, presetName string) error {
	presets := s.getSRSPresets()

	var selectedPreset *SRSConfigPreset
	for _, preset := range presets {
		if strings.ToLower(preset.Name) == strings.ToLower(presetName) {
			selectedPreset = &preset
			break
		}
	}

	if selectedPreset == nil {
		return errors.New("preset not found")
	}

	req := UpdateSRSConfigRequest{
		EasyBonus:        selectedPreset.EasyBonus,
		HardPenalty:      selectedPreset.HardPenalty,
		FailurePenalty:   selectedPreset.FailurePenalty,
		MinEaseFactor:    selectedPreset.MinEaseFactor,
		MaxEaseFactor:    selectedPreset.MaxEaseFactor,
		GraduationSteps:  selectedPreset.GraduationSteps,
		NewWordsPerDay:   selectedPreset.NewWordsPerDay,
		MaxReviewsPerDay: selectedPreset.MaxReviewsPerDay,
	}

	return s.UpdateSRSConfig(ctx, userID, req)
}
