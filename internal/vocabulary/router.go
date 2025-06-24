package vocabulary

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	polyfyjwt "github.com/Raylynd6299/babel/pkg/jwt"
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

// CRUD Operations
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

// Review System
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

// Statistics
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

// Search and Filter
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

// Import/Export
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

// SRS Configuration
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

// Placeholder methods for vocabulary lists and bulk operations
func (r *Router) GetVocabularyLists(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Feature not implemented yet"})
}

func (r *Router) CreateVocabularyList(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Feature not implemented yet"})
}

func (r *Router) GetVocabularyList(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Feature not implemented yet"})
}

func (r *Router) UpdateVocabularyList(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Feature not implemented yet"})
}

func (r *Router) DeleteVocabularyList(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Feature not implemented yet"})
}

func (r *Router) BulkAddVocabulary(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Feature not implemented yet"})
}

func (r *Router) BulkDeleteVocabulary(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Feature not implemented yet"})
}

func (r *Router) BulkResetProgress(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Feature not implemented yet"})
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
