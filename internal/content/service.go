package content

import (
	"context"
	"errors"
	"strings"

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

func (s *Service) GetDB() *gorm.DB {
	return s.db
}
func (s *Service) CreateContent(ctx context.Context, req CreateContentRequest, userID string) (*Content, error) {
	content := Content{
		Title:                  req.Title,
		ContentType:            req.ContentType,
		LanguageID:             req.LanguageID,
		TotalEpisodes:          req.TotalEpisodes,
		AverageEpisodeDuration: req.AverageEpisodeDuration,
		YearReleased:           req.YearReleased,
		Country:                req.Country,
		Genre:                  req.Genre,
		Description:            req.Description,
		PosterURL:              req.PosterURL,
		IMDbRating:             req.IMDbRating,
		CreatedBy:              userID,
	}

	if err := s.db.Create(&content).Error; err != nil {
		return nil, err
	}

	return &content, nil
}

func (s *Service) GetContent(ctx context.Context, id string) (*Content, error) {
	var content Content
	err := s.db.Preload("Episodes").Preload("Language").Where("id = ?", id).First(&content).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("content not found")
		}
		return nil, err
	}

	return &content, nil
}

func (s *Service) GetContentList(ctx context.Context, filter ContentFilter) ([]Content, int64, error) {
	query := s.db.Model(&Content{}).Preload("Language")

	// Apply filters
	if filter.LanguageID > 0 {
		query = query.Where("language_id = ?", filter.LanguageID)
	}

	if filter.ContentType != "" {
		query = query.Where("content_type = ?", filter.ContentType)
	}

	if filter.Genre != "" {
		query = query.Where("genre ILIKE ?", "%"+filter.Genre+"%")
	}

	if filter.Country != "" {
		query = query.Where("country ILIKE ?", "%"+filter.Country+"%")
	}

	if filter.MinRating > 0 {
		query = query.Where("imdb_rating >= ?", filter.MinRating)
	}

	if filter.MaxRating > 0 {
		query = query.Where("imdb_rating <= ?", filter.MaxRating)
	}

	if len(filter.Difficulty) > 0 {
		query = query.Where("difficulty_level IN ?", filter.Difficulty)
	}

	if filter.YearFrom > 0 {
		query = query.Where("year_released >= ?", filter.YearFrom)
	}

	if filter.YearTo > 0 {
		query = query.Where("year_released <= ?", filter.YearTo)
	}

	if filter.Search != "" {
		searchTerm := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	// Count total records
	var total int64
	query.Count(&total)

	// Apply sorting
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}

	sortDirection := "desc"
	if filter.SortDirection == "asc" {
		sortDirection = "asc"
	}

	query = query.Order(sortBy + " " + sortDirection)

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(20) // Default limit
	}

	if filter.Offset > 0 {
		query.Offset(filter.Offset)
	}

	var contents []Content
	if err := query.Find(&contents).Error; err != nil {
		return nil, 0, err
	}

	return contents, total, nil
}

func (s *Service) RateContent(ctx context.Context, userID, contentID string, rating ContentRating) error {
	//  Check if content exists
	var content Content
	if err := s.db.Where("id = ?", contentID).First(&content).Error; err != nil {
		return errors.New("content not found")
	}

	// Check if user already rated this content
	var existingRating ContentRating
	err := s.db.Where("user_id = ? AND content_id = ?", userID, contentID).First(&existingRating).Error

	if err == nil {
		// Update existing rating
		existingRating.DifficultyRating = rating.DifficultyRating
		existingRating.UsefulnessRating = rating.UsefulnessRating
		existingRating.EntertainmentRating = rating.EntertainmentRating
		existingRating.ReviewText = rating.ReviewText

		return s.db.Save(&existingRating).Error
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new Rating
		rating.UserID = userID
		rating.ContentID = contentID

		return s.db.Create(&rating).Error
	}

	return err
}

func (s *Service) GetRecommendations(ctx context.Context, userID string, languageID int, limit int) ([]Content, error) {
	// Simple recommendation: popular content in the user's target language
	// that they haven't consumed yet
	var contents []Content

	subquery := s.db.Table("user_progress").
		Select("DISTINCT content_id").
		Where("user_id = ?", userID)

	err := s.db.Where("language_id = ?", languageID).
		Where("id NOT IN (?)", subquery).
		Order("view_count DESC, imdb_rating DESC").
		Limit(limit).
		Find(&contents).Error

	return contents, err
}

func (s *Service) UpdateContent(ctx context.Context, id string, req CreateContentRequest, userID string) (*Content, error) {
	var content Content
	if err := s.db.Where("id = ?", id).First(&content).Error; err != nil {
		return nil, errors.New("content not found")
	}

	// Check if user owns the content or is admin
	if content.CreatedBy != userID {
		return nil, errors.New("unauthorized to update this content")
	}

	// Update fields
	content.Title = req.Title
	content.TotalEpisodes = req.TotalEpisodes
	content.AverageEpisodeDuration = req.AverageEpisodeDuration
	content.YearReleased = req.YearReleased
	content.Country = req.Country
	content.Genre = req.Genre
	content.Description = req.Description
	content.PosterURL = req.PosterURL
	content.IMDbRating = req.IMDbRating

	if err := s.db.Save(&content).Error; err != nil {
		return nil, err
	}

	return &content, nil
}

func (s *Service) DeleteContent(ctx context.Context, id string, userID string) error {
	var content Content
	if err := s.db.Where("id = ?", id).First(&content).Error; err != nil {
		return errors.New("content not found")
	}

	// Check if user owns the content or is admin
	if content.CreatedBy != userID {
		return errors.New("unauthorized to delete this content")
	}

	// Soft delete
	if err := s.db.Delete(&content).Error; err != nil {
		return err
	}

	return nil
}

func (s *Service) GetContentEpisodes(ctx context.Context, contentID string) ([]ContentEpisode, error) {
	var episodes []ContentEpisode
	err := s.db.Where("content_id = ?", contentID).
		Order("season_number ASC, episode_number ASC").
		Find(&episodes).Error
	return episodes, err
}

func (s *Service) CreateEpisode(ctx context.Context, contentID string, req CreateEpisodeRequest, userID string) (*ContentEpisode, error) {
	// Verify content exists and user owns it
	var content Content
	if err := s.db.Where("id = ?", contentID).First(&content).Error; err != nil {
		return nil, errors.New("content not found")
	}

	if content.CreatedBy != userID {
		return nil, errors.New("unauthorized to add episodes to this content")
	}

	// Check if episode number already exists for this content/season
	var existingEpisode ContentEpisode
	err := s.db.Where("content_id = ? AND season_number = ? AND episode_number = ?",
		contentID, req.SeasonNumber, req.EpisodeNumber).First(&existingEpisode).Error
	if err == nil {
		return nil, errors.New("episode number already exists for this season")
	}

	episode := ContentEpisode{
		ContentID:       contentID,
		EpisodeNumber:   req.EpisodeNumber,
		Title:           req.Title,
		DurationMinutes: req.DurationMinutes,
		SeasonNumber:    req.SeasonNumber,
		Description:     req.Description,
	}

	if err := s.db.Create(&episode).Error; err != nil {
		return nil, err
	}

	return &episode, nil
}

func (s *Service) UpdateEpisode(ctx context.Context, episodeID string, req CreateEpisodeRequest, userID string) (*ContentEpisode, error) {
	var episode ContentEpisode
	if err := s.db.Preload("Content").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		return nil, errors.New("episode not found")
	}

	// Check if user owns the content
	var content Content
	if err := s.db.Where("id = ?", episode.ContentID).First(&content).Error; err != nil {
		return nil, errors.New("content not found")
	}

	if content.CreatedBy != userID {
		return nil, errors.New("unauthorized to update this episode")
	}

	// Update episode
	episode.EpisodeNumber = req.EpisodeNumber
	episode.Title = req.Title
	episode.DurationMinutes = req.DurationMinutes
	episode.SeasonNumber = req.SeasonNumber
	episode.Description = req.Description

	if err := s.db.Save(&episode).Error; err != nil {
		return nil, err
	}

	return &episode, nil
}

func (s *Service) DeleteEpisode(ctx context.Context, episodeID string, userID string) error {
	var episode ContentEpisode
	if err := s.db.Where("id = ?", episodeID).First(&episode).Error; err != nil {
		return errors.New("episode not found")
	}

	// Check if user owns the content
	var content Content
	if err := s.db.Where("id = ?", episode.ContentID).First(&content).Error; err != nil {
		return errors.New("content not found")
	}

	if content.CreatedBy != userID {
		return errors.New("unauthorized to delete this episode")
	}

	if err := s.db.Delete(&episode).Error; err != nil {
		return err
	}

	return nil
}

func (s *Service) GetLanguages(ctx context.Context) ([]Language, error) {
	var languages []Language
	err := s.db.Where("is_active = ?", true).Order("name ASC").Find(&languages).Error
	return languages, err
}
