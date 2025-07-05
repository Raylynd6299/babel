package vocabulary

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// VocabularyList represents a collection of vocabulary words
type VocabularyList struct {
	ID          string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      string    `json:"user_id" gorm:"not null"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	LanguageID  int       `json:"language_id" gorm:"not null"`
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Items []VocabularyListItem `json:"items,omitempty" gorm:"foreignKey:ListID"`
}

// VocabularyListItem represents a vocabulary word in a list
type VocabularyListItem struct {
	ID           string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ListID       string    `json:"list_id" gorm:"not null"`
	VocabularyID string    `json:"vocabulary_id" gorm:"not null"`
	Order        int       `json:"order" gorm:"default:0"`
	CreatedAt    time.Time `json:"created_at"`

	// Relations
	Vocabulary Vocabulary `json:"vocabulary,omitempty" gorm:"foreignKey:VocabularyID"`
}

// Request/Response types
type CreateVocabularyListRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=1000"`
	LanguageID  int    `json:"language_id" validate:"required"`
	IsPublic    bool   `json:"is_public"`
}

type UpdateVocabularyListRequest struct {
	Name        string `json:"name" validate:"omitempty,min=1,max=255"`
	Description string `json:"description" validate:"omitempty,max=1000"`
	IsPublic    *bool  `json:"is_public"`
}

type BulkAddVocabularyRequest struct {
	VocabularyIDs []string `json:"vocabulary_ids" validate:"required,min=1,max=100"`
	ListID        string   `json:"list_id"`
}

type BulkDeleteVocabularyRequest struct {
	VocabularyIDs []string `json:"vocabulary_ids" validate:"required,min=1,max=100"`
}

type BulkResetProgressRequest struct {
	VocabularyIDs []string `json:"vocabulary_ids" validate:"required,min=1,max=100"`
	ResetType     string   `json:"reset_type" validate:"required,oneof=all progress reviews"`
}

type BulkOperationResult struct {
	Total     int      `json:"total"`
	Processed int      `json:"processed"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

// Vocabulary Lists Service Methods
func (s *Service) GetVocabularyLists(ctx context.Context, userID string, languageID int) ([]VocabularyList, error) {
	var lists []VocabularyList

	query := s.db.Where("user_id = ?", userID)
	if languageID > 0 {
		query = query.Where("language_id = ?", languageID)
	}

	err := query.Preload("Items").
		Preload("Items.Vocabulary").
		Order("created_at DESC").
		Find(&lists).Error

	return lists, err
}

func (s *Service) CreateVocabularyList(ctx context.Context, userID string, req CreateVocabularyListRequest) (*VocabularyList, error) {
	list := VocabularyList{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		LanguageID:  req.LanguageID,
		IsPublic:    req.IsPublic,
	}

	if err := s.db.Create(&list).Error; err != nil {
		return nil, err
	}

	return &list, nil
}

func (s *Service) GetVocabularyList(ctx context.Context, userID, listID string) (*VocabularyList, error) {
	var list VocabularyList

	err := s.db.Where("id = ? AND (user_id = ? OR is_public = ?)", listID, userID, true).
		Preload("Items").
		Preload("Items.Vocabulary").
		First(&list).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("vocabulary list not found")
		}
		return nil, err
	}

	return &list, nil
}

func (s *Service) UpdateVocabularyList(ctx context.Context, userID, listID string, req UpdateVocabularyListRequest) (*VocabularyList, error) {
	var list VocabularyList
	err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("vocabulary list not found")
		}
		return nil, err
	}

	// Update fields
	if req.Name != "" {
		list.Name = req.Name
	}
	if req.Description != "" {
		list.Description = req.Description
	}
	if req.IsPublic != nil {
		list.IsPublic = *req.IsPublic
	}

	if err := s.db.Save(&list).Error; err != nil {
		return nil, err
	}

	// Reload with items
	s.db.Preload("Items").Preload("Items.Vocabulary").Where("id = ?", list.ID).First(&list)
	return &list, nil
}

func (s *Service) DeleteVocabularyList(ctx context.Context, userID, listID string) error {
	var list VocabularyList
	err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("vocabulary list not found")
		}
		return err
	}

	// Delete list items first
	if err := s.db.Where("list_id = ?", listID).Delete(&VocabularyListItem{}).Error; err != nil {
		return err
	}

	// Delete the list
	return s.db.Delete(&list).Error
}

func (s *Service) AddVocabularyToList(ctx context.Context, userID, listID, vocabularyID string) error {
	// Verify list ownership
	var list VocabularyList
	err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error
	if err != nil {
		return errors.New("vocabulary list not found")
	}

	// Verify vocabulary exists and user has access
	var userVocab UserVocabulary
	err = s.db.Where("user_id = ? AND vocabulary_id = ?", userID, vocabularyID).First(&userVocab).Error
	if err != nil {
		return errors.New("vocabulary not found in your collection")
	}

	// Check if already in list
	var existing VocabularyListItem
	err = s.db.Where("list_id = ? AND vocabulary_id = ?", listID, vocabularyID).First(&existing).Error
	if err == nil {
		return errors.New("vocabulary already in list")
	}

	// Get next order
	var maxOrder int
	s.db.Model(&VocabularyListItem{}).Where("list_id = ?", listID).Select("COALESCE(MAX(order), 0)").Scan(&maxOrder)

	// Add to list
	item := VocabularyListItem{
		ListID:       listID,
		VocabularyID: vocabularyID,
		Order:        maxOrder + 1,
	}

	return s.db.Create(&item).Error
}

func (s *Service) RemoveVocabularyFromList(ctx context.Context, userID, listID, vocabularyID string) error {
	// Verify list ownership
	var list VocabularyList
	err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error
	if err != nil {
		return errors.New("vocabulary list not found")
	}

	// Remove from list
	result := s.db.Where("list_id = ? AND vocabulary_id = ?", listID, vocabularyID).Delete(&VocabularyListItem{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("vocabulary not found in list")
	}

	return nil
}

// Bulk Operations Service Methods
func (s *Service) BulkAddVocabulary(ctx context.Context, userID string, req BulkAddVocabularyRequest) (*BulkOperationResult, error) {
	result := &BulkOperationResult{
		Total:  len(req.VocabularyIDs),
		Errors: make([]string, 0),
	}

	if req.ListID != "" {
		// Add to specific list
		for _, vocabID := range req.VocabularyIDs {
			err := s.AddVocabularyToList(ctx, userID, req.ListID, vocabID)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to add %s to list: %v", vocabID, err))
			} else {
				result.Processed++
			}
		}
	} else {
		result.Errors = append(result.Errors, "List ID is required for bulk add operation")
		result.Failed = result.Total
	}

	return result, nil
}

func (s *Service) BulkDeleteVocabulary(ctx context.Context, userID string, req BulkDeleteVocabularyRequest) (*BulkOperationResult, error) {
	result := &BulkOperationResult{
		Total:  len(req.VocabularyIDs),
		Errors: make([]string, 0),
	}

	for _, vocabID := range req.VocabularyIDs {
		err := s.DeleteVocabulary(ctx, userID, vocabID)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to delete %s: %v", vocabID, err))
		} else {
			result.Processed++
		}
	}

	return result, nil
}

func (s *Service) BulkResetProgress(ctx context.Context, userID string, req BulkResetProgressRequest) (*BulkOperationResult, error) {
	result := &BulkOperationResult{
		Total:  len(req.VocabularyIDs),
		Errors: make([]string, 0),
	}

	for _, vocabID := range req.VocabularyIDs {
		var userVocab UserVocabulary
		err := s.db.Where("user_id = ? AND vocabulary_id = ?", userID, vocabID).First(&userVocab).Error
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Vocabulary %s not found", vocabID))
			continue
		}

		// Reset based on type
		switch req.ResetType {
		case "all":
			userVocab.MasteryLevel = 0
			userVocab.ReviewCount = 0
			userVocab.CorrectCount = 0
			userVocab.LastReviewedAt = nil
			userVocab.NextReviewAt = nil
			userVocab.EaseFactor = s.srsConfig.MaxEaseFactor
			userVocab.IntervalDays = 1
		case "progress":
			userVocab.MasteryLevel = 0
			userVocab.EaseFactor = s.srsConfig.MaxEaseFactor
			userVocab.IntervalDays = 1
			nextReview := time.Now().Add(time.Hour * 24)
			userVocab.NextReviewAt = &nextReview
		case "reviews":
			userVocab.ReviewCount = 0
			userVocab.CorrectCount = 0
			userVocab.LastReviewedAt = nil
		default:
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid reset type: %s", req.ResetType))
			continue
		}

		if err := s.db.Save(&userVocab).Error; err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to reset %s: %v", vocabID, err))
		} else {
			result.Processed++
		}
	}

	return result, nil
}
