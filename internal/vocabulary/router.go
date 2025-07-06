package vocabulary

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	polyfyjwt "github.com/Raylynd6299/babel/pkg/jwt"

	_ "github.com/Raylynd6299/babel/cmd/vocabulary-service/docs"
)

type Router struct {
	service    *Service
	validator  *validator.Validate
	jwtService *polyfyjwt.Service
}

func NewRouter(service *Service) *gin.Engine {
	router := gin.Default()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	// Create JWT Service
	jwtConfig := polyfyjwt.Config{
		SecretKey:            service.jwtSecret,
		AccessTokenDuration:  time.Hour * 2,
		RefreshTokenDuration: time.Hour * 24 * 7,
		Issuer:               "polyfy-auth",
	}

	jwtService := polyfyjwt.NewService(jwtConfig)

	vocabularyRouter := &Router{
		service:    service,
		validator:  validator.New(),
		jwtService: jwtService,
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("vocabulary")))

	v1 := router.Group("/api/v1")

	// All vocabulary routes require authentication
	protected := v1.Group("/vocabulary")
	protected.Use(vocabularyRouter.jwtService.AuthMiddleware())
	{
		// CRUD operations
		protected.POST("/", vocabularyRouter.AddVocabulary)
		protected.GET("/", vocabularyRouter.GetUserVocabulary)
		protected.PUT("/:id", vocabularyRouter.UpdateVocabulary)
		protected.DELETE("/:id", vocabularyRouter.DeleteVocabulary)

		// Review system
		protected.GET("/reviews", vocabularyRouter.GetVocabularyForReview)
		protected.POST("/reviews", vocabularyRouter.ReviewVocabulary)
		protected.POST("/reviews/batch", vocabularyRouter.BatchReviewVocabulary)

		// Statistics and analytics
		protected.GET("/stats", vocabularyRouter.GetVocabularyStats)
		protected.GET("/progress", vocabularyRouter.GetVocabularyProgress)

		// Search and filter
		protected.GET("/search", vocabularyRouter.SearchVocabulary)
		protected.GET("/filter", vocabularyRouter.FilterVocabulary)

		// Import/Export
		protected.POST("/import", vocabularyRouter.ImportVocabulary)
		protected.GET("/export", vocabularyRouter.ExportVocabulary)

		// SRS Configuration
		protected.GET("/srs-config", vocabularyRouter.GetSRSConfig)
		protected.PUT("/srs-config", vocabularyRouter.UpdateSRSConfig)
		protected.PUT("/srs-config/preset/:preset", vocabularyRouter.ApplySRSPreset)

		// Vocabulary lists and collections
		protected.GET("/lists", vocabularyRouter.GetVocabularyLists)
		protected.POST("/lists", vocabularyRouter.CreateVocabularyList)
		protected.GET("/lists/:list_id", vocabularyRouter.GetVocabularyList)
		protected.PUT("/lists/:list_id", vocabularyRouter.UpdateVocabularyList)
		protected.DELETE("/lists/:list_id", vocabularyRouter.DeleteVocabularyList)

		// Bulk operations
		protected.POST("/bulk-add", vocabularyRouter.BulkAddVocabulary)
		protected.POST("/bulk-delete", vocabularyRouter.BulkDeleteVocabulary)
		protected.POST("/bulk-reset", vocabularyRouter.BulkResetProgress)
	}

	return router
}

// AddVocabulary godoc
// @Summary      Add new vocabulary
// @Description  Add a new vocabulary word to user's collection
// @Tags         vocabulary
// @Accept       json
// @Produce      json
// @Param        language_id query int true "Language ID"
// @Param        request body AddVocabularyRequest true "Vocabulary data"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary [post]
func (r *Router) AddVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req AddVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get language_id from query parameter
	languageID := 0
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = id
		}
	}

	if languageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language_id is required"})
		return
	}

	userVocab, err := r.service.AddVocabulary(c.Request.Context(), userID, languageID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Vocabulary added successfully",
		"vocabulary": userVocab,
	})
}

// GetUserVocabulary godoc
// @Summary      Get user vocabulary
// @Description  Get paginated list of user's vocabulary words
// @Tags         vocabulary
// @Accept       json
// @Produce      json
// @Param        language_id query int true "Language ID"
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary [get]
func (r *Router) GetUserVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	languageID := 0
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = id
		}
	}

	if languageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language_id is required"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	vocab, total, err := r.service.GetUserVocabulary(c.Request.Context(), userID, languageID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vocabulary": vocab,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// UpdateVocabulary godoc
// @Summary      Update vocabulary
// @Description  Update an existing vocabulary word
// @Tags         vocabulary
// @Accept       json
// @Produce      json
// @Param        id path string true "Vocabulary ID"
// @Param        request body UpdateVocabularyRequest true "Updated vocabulary data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/{id} [put]
func (r *Router) UpdateVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	vocabularyID := c.Param("id")
	if vocabularyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vocabulary ID required"})
		return
	}

	var req UpdateVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userVocab, err := r.service.UpdateVocabulary(c.Request.Context(), userID, vocabularyID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Vocabulary updated successfully",
		"vocabulary": userVocab,
	})
}

// DeleteVocabulary godoc
// @Summary      Delete vocabulary
// @Description  Delete an existing vocabulary word from user's collection
// @Tags         vocabulary
// @Accept       json
// @Produce      json
// @Param        id path string true "Vocabulary ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/{id} [delete]
func (r *Router) DeleteVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	vocabularyID := c.Param("id")
	if vocabularyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vocabulary ID required"})
		return
	}

	err := r.service.DeleteVocabulary(c.Request.Context(), userID, vocabularyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vocabulary deleted successfully"})
}

// GetVocabularyForReview godoc
// @Summary      Get vocabulary for review
// @Description  Get vocabulary words that are due for SRS review
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        language_id query int true "Language ID"
// @Param        limit query int false "Number of items to return (max 50)" default(10)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/reviews [get]
func (r *Router) GetVocabularyForReview(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	languageID := 0
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = id
		}
	}

	if languageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language_id is required"})
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	vocab, err := r.service.GetVocabularyForReview(c.Request.Context(), userID, languageID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vocabulary": vocab,
		"count":      len(vocab),
	})
}

// ReviewVocabulary godoc
// @Summary      Review vocabulary
// @Description  Submit a review for a vocabulary word using SRS algorithm
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        request body ReviewRequest true "Review data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/reviews [post]
func (r *Router) ReviewVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userVocab, err := r.service.ReviewVocabulary(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Review completed successfully",
		"vocabulary": userVocab,
	})
}

// BatchReviewVocabulary godoc
// @Summary      Batch review vocabulary
// @Description  Submit multiple vocabulary reviews in a single request
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        request body BatchReviewRequest true "Batch review data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/reviews/batch [post]
func (r *Router) BatchReviewVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req BatchReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := r.service.BatchReviewVocabulary(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Batch review completed successfully",
		"results": results,
	})
}

// GetVocabularyStats godoc
// @Summary      Get vocabulary statistics
// @Description  Get comprehensive statistics for user's vocabulary in a specific language
// @Tags         statistics
// @Accept       json
// @Produce      json
// @Param        language_id query int true "Language ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/stats [get]
func (r *Router) GetVocabularyStats(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	languageID := 0
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = id
		}
	}

	if languageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language_id is required"})
		return
	}

	stats, err := r.service.GetVocabularyStats(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// GetVocabularyProgress godoc
// @Summary      Get vocabulary progress
// @Description  Get vocabulary learning progress over time with analytics
// @Tags         statistics
// @Accept       json
// @Produce      json
// @Param        language_id query int false "Language ID (optional for all languages)"
// @Param        days query int false "Number of days to analyze (max 365)" default(30)
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/progress [get]
func (r *Router) GetVocabularyProgress(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	languageID := 0
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = id
		}
	}

	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	progress, err := r.service.GetVocabularyProgress(c.Request.Context(), userID, languageID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"progress": progress})
}

// SearchVocabulary godoc
// @Summary      Search vocabulary
// @Description  Search vocabulary words by term in word, definition, or notes
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        language_id query int false "Language ID (optional for all languages)"
// @Param        q query string true "Search query term"
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/search [get]
func (r *Router) SearchVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	languageID := 0
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = id
		}
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	vocab, err := r.service.SearchVocabulary(c.Request.Context(), userID, languageID, query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vocabulary": vocab,
		"query":      query,
		"count":      len(vocab),
	})
}

// FilterVocabulary godoc
// @Summary      Filter vocabulary
// @Description  Filter vocabulary words using advanced criteria and filters
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        request body VocabularyFilter true "Filter criteria"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/filter [post]
func (r *Router) FilterVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var filter VocabularyFilter
	if err := c.ShouldBindJSON(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vocab, total, err := r.service.FilterVocabulary(c.Request.Context(), userID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vocabulary": vocab,
		"total":      total,
		"filter":     filter,
	})
}

// ImportVocabulary godoc
// @Summary      Import vocabulary
// @Description  Import vocabulary words from external sources or files
// @Tags         import-export
// @Accept       json
// @Produce      json
// @Param        request body ImportVocabularyRequest true "Import data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/import [post]
func (r *Router) ImportVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req ImportVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := r.service.ImportVocabulary(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Vocabulary imported successfully",
		"result":  result,
	})
}

// ExportVocabulary godoc
// @Summary      Export vocabulary
// @Description  Export user's vocabulary words in various formats (JSON, CSV)
// @Tags         import-export
// @Accept       json
// @Produce      json
// @Param        language_id query int false "Language ID (optional for all languages)"
// @Param        format query string false "Export format (json, csv)" default(json)
// @Success      200 {string} string "File content"
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/export [get]
func (r *Router) ExportVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	languageID := 0
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = id
		}
	}

	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	data, err := r.service.ExportVocabulary(c.Request.Context(), userID, languageID, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set appropriate headers based on format
	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=vocabulary.csv")
	case "json":
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=vocabulary.json")
	}

	c.String(http.StatusOK, data)
}

// GetSRSConfig godoc
// @Summary      Get SRS configuration
// @Description  Get current Spaced Repetition System configuration for the user
// @Tags         srs
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/srs-config [get]
func (r *Router) GetSRSConfig(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	configResponse, err := r.service.GetSRSConfig(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"srs_config": configResponse})
}

// UpdateSRSConfig godoc
// @Summary      Update SRS configuration
// @Description  Update Spaced Repetition System configuration with custom parameters
// @Tags         srs
// @Accept       json
// @Produce      json
// @Param        request body UpdateSRSConfigRequest true "SRS configuration data"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/srs-config [put]
func (r *Router) UpdateSRSConfig(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req UpdateSRSConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Additional validation
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := r.service.UpdateSRSConfig(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SRS configuration updated successfully"})
}

// ApplySRSPreset godoc
// @Summary      Apply SRS preset
// @Description  Apply a predefined SRS configuration preset (beginner, intermediate, advanced)
// @Tags         srs
// @Accept       json
// @Produce      json
// @Param        preset path string true "Preset name (beginner, intermediate, advanced)"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/srs-config/preset/{preset} [put]
func (r *Router) ApplySRSPreset(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	presetName := c.Param("preset")
	if presetName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Preset name required"})
		return
	}

	err := r.service.ApplySRSPreset(c.Request.Context(), userID, presetName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SRS preset applied successfully"})
}

// GetVocabularyLists godoc
// @Summary      Get vocabulary lists
// @Description  Get all vocabulary lists for the user, optionally filtered by language
// @Tags         lists
// @Accept       json
// @Produce      json
// @Param        language_id query int false "Language ID (optional for all languages)"
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/lists [get]
func (r *Router) GetVocabularyLists(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	languageID := 0
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = id
		}
	}

	lists, err := r.service.GetVocabularyLists(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"lists": lists})
}

// CreateVocabularyList godoc
// @Summary      Create vocabulary list
// @Description  Create a new vocabulary list for organizing words
// @Tags         lists
// @Accept       json
// @Produce      json
// @Param        request body CreateVocabularyListRequest true "List data"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/lists [post]
func (r *Router) CreateVocabularyList(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateVocabularyListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	list, err := r.service.CreateVocabularyList(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Vocabulary list created successfully",
		"list":    list,
	})
}

// GetVocabularyList godoc
// @Summary      Get vocabulary list
// @Description  Get a specific vocabulary list with its words
// @Tags         lists
// @Accept       json
// @Produce      json
// @Param        list_id path string true "List ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/lists/{list_id} [get]
func (r *Router) GetVocabularyList(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	listID := c.Param("list_id")
	if listID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "List ID required"})
		return
	}

	list, err := r.service.GetVocabularyList(c.Request.Context(), userID, listID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"list": list})
}

// UpdateVocabularyList godoc
// @Summary      Update vocabulary list
// @Description  Update an existing vocabulary list's metadata and settings
// @Tags         lists
// @Accept       json
// @Produce      json
// @Param        list_id path string true "List ID"
// @Param        request body UpdateVocabularyListRequest true "Updated list data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/lists/{list_id} [put]
func (r *Router) UpdateVocabularyList(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	listID := c.Param("list_id")
	if listID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "List ID required"})
		return
	}

	var req UpdateVocabularyListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	list, err := r.service.UpdateVocabularyList(c.Request.Context(), userID, listID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Vocabulary list updated successfully",
		"list":    list,
	})
}

// DeleteVocabularyList godoc
// @Summary      Delete vocabulary list
// @Description  Delete an existing vocabulary list and optionally its associated words
// @Tags         lists
// @Accept       json
// @Produce      json
// @Param        list_id path string true "List ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/lists/{list_id} [delete]
func (r *Router) DeleteVocabularyList(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	listID := c.Param("list_id")
	if listID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "List ID required"})
		return
	}

	err := r.service.DeleteVocabularyList(c.Request.Context(), userID, listID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vocabulary list deleted successfully"})
}

// BulkAddVocabulary godoc
// @Summary      Bulk add vocabulary
// @Description  Add multiple vocabulary words in a single operation
// @Tags         bulk
// @Accept       json
// @Produce      json
// @Param        request body BulkAddVocabularyRequest true "Bulk vocabulary data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/bulk-add [post]
func (r *Router) BulkAddVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req BulkAddVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := r.service.BulkAddVocabulary(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bulk add operation completed",
		"result":  result,
	})
}

// BulkDeleteVocabulary godoc
// @Summary      Bulk delete vocabulary
// @Description  Delete multiple vocabulary words in a single operation
// @Tags         bulk
// @Accept       json
// @Produce      json
// @Param        request body BulkDeleteVocabularyRequest true "Bulk delete data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/bulk-delete [post]
func (r *Router) BulkDeleteVocabulary(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req BulkDeleteVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := r.service.BulkDeleteVocabulary(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bulk delete operation completed",
		"result":  result,
	})
}

// BulkResetProgress godoc
// @Summary      Bulk reset progress
// @Description  Reset SRS progress for multiple vocabulary words in a single operation
// @Tags         bulk
// @Accept       json
// @Produce      json
// @Param        request body BulkResetProgressRequest true "Bulk reset data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /vocabulary/bulk-reset [post]
func (r *Router) BulkResetProgress(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req BulkResetProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := r.service.BulkResetProgress(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bulk reset operation completed",
		"result":  result,
	})
}

// Middleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
