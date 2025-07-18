package content

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	polyfyjwt "github.com/Raylynd6299/babel/pkg/jwt"

	_ "github.com/Raylynd6299/babel/cmd/content-service/docs"
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

	contentRouter := &Router{
		service:    service,
		validator:  validator.New(),
		jwtService: jwtService,
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("content")))

	v1 := router.Group("/api/v1")

	// Public route (read-only)
	public := v1.Group("/content")
	{
		public.GET("/", contentRouter.GetContentList)
		public.GET("/:id", contentRouter.GetContent)
		public.GET("/:id/episodes", contentRouter.GetContentEpisodes)
		public.GET("/languages", contentRouter.GetLanguages)
	}

	// Protected routes
	protected := v1.Group("/content")
	protected.Use(contentRouter.jwtService.AuthMiddleware())
	{
		protected.POST("/", contentRouter.CreateContent)
		protected.PUT("/:id", contentRouter.UpdateContent)
		protected.DELETE("/:id", contentRouter.DeleteContent)
		protected.POST("/:id/rate", contentRouter.RateContent)
		protected.POST("/:id/episodes", contentRouter.CreateEpisode)
		protected.PUT("/:id/episodes/:episode_id", contentRouter.UpdateEpisode)
		protected.DELETE("/:id/episodes/:episode_id", contentRouter.DeleteEpisode)
		protected.GET("/recommendations", contentRouter.GetRecommendations)
	}

	return router
}

// CreateContent godoc
// @Summary      Create new content
// @Description  Create a new piece of content (movie, series, etc.)
// @Tags         content
// @Accept       json
// @Produce      json
// @Param        request body CreateContentRequest true "Content data"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /content [post]
func (r *Router) CreateContent(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	content, err := r.service.CreateContent(c.Request.Context(), req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"content": content})
}

// GetContent godoc
// @Summary      Get content by ID
// @Description  Get detailed information about a specific content item
// @Tags         content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /content/{id} [get]
func (r *Router) GetContent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content ID required"})
		return
	}

	content, err := r.service.GetContent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content})
}

// UpdateContent godoc
// @Summary      Update content
// @Description  Update an existing content item
// @Tags         content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID"
// @Param        request body CreateContentRequest true "Content update data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /content/{id} [put]
func (r *Router) UpdateContent(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content ID required"})
		return
	}

	var req CreateContentRequest // Reusing the same struct for updates
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	content, err := r.service.UpdateContent(c.Request.Context(), id, req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content})
}

// DeleteContent godoc
// @Summary      Delete content
// @Description  Delete an existing content item
// @Tags         content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /content/{id} [delete]
func (r *Router) DeleteContent(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content ID required"})
		return
	}

	err := r.service.DeleteContent(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Content deleted successfully"})
}

// RateContent godoc
// @Summary      Rate content
// @Description  Rate a content item with difficulty, usefulness and entertainment ratings
// @Tags         ratings
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID"
// @Param        request body RateContentRequest true "Rating data"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /content/{id}/rate [post]
func (r *Router) RateContent(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	contentID := c.Param("id")
	if contentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content ID required"})
		return
	}

	var req RateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rating := ContentRating{
		UserID:              userID,
		ContentID:           contentID,
		DifficultyRating:    req.DifficultyRating,
		UsefulnessRating:    req.UsefulnessRating,
		EntertainmentRating: req.EntertainmentRating,
		ReviewText:          req.ReviewText,
	}

	err := r.service.RateContent(c.Request.Context(), userID, contentID, rating)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Content rated successfully"})
}

// GetContentList godoc
// @Summary      Get content list
// @Description  Get a paginated list of content with filtering options
// @Tags         content
// @Accept       json
// @Produce      json
// @Param        language_id query int false "Language ID"
// @Param        content_type query string false "Content type (movie, series, etc.)"
// @Param        genre query string false "Genre"
// @Param        country query string false "Country"
// @Param        min_rating query number false "Minimum rating"
// @Param        max_rating query number false "Maximum rating"
// @Param        difficulty query string false "Difficulty levels (comma-separated)"
// @Param        year_from query int false "Year from"
// @Param        year_to query int false "Year to"
// @Param        search query string false "Search term"
// @Param        limit query int false "Limit (max 100)" default(20)
// @Param        offset query int false "Offset" default(0)
// @Param        sort_by query string false "Sort by field"
// @Param        sort_direction query string false "Sort direction (asc/desc)"
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]string
// @Router       /content [get]
func (r *Router) GetContentList(c *gin.Context) {
	filter := ContentFilter{
		Limit:  20,
		Offset: 0,
	}

	// Parse query parameters
	if languageID := c.Query("language_id"); languageID != "" {
		if id, err := strconv.Atoi(languageID); err == nil {
			filter.LanguageID = id
		}
	}

	if contentType := c.Query("content_type"); contentType != "" {
		filter.ContentType = contentType
	}

	if genre := c.Query("genre"); genre != "" {
		filter.Genre = genre
	}

	if country := c.Query("country"); country != "" {
		filter.Country = country
	}

	if minRating := c.Query("min_rating"); minRating != "" {
		if rating, err := strconv.ParseFloat(minRating, 32); err == nil {
			filter.MinRating = float32(rating)
		}
	}

	if maxRating := c.Query("max_rating"); maxRating != "" {
		if rating, err := strconv.ParseFloat(maxRating, 32); err == nil {
			filter.MaxRating = float32(rating)
		}
	}

	if difficulty := c.Query("difficulty"); difficulty != "" {
		filter.Difficulty = strings.Split(difficulty, ",")
	}

	if yearFrom := c.Query("year_from"); yearFrom != "" {
		if year, err := strconv.Atoi(yearFrom); err == nil {
			filter.YearFrom = year
		}
	}

	if yearTo := c.Query("year_to"); yearTo != "" {
		if year, err := strconv.Atoi(yearTo); err == nil {
			filter.YearTo = year
		}
	}

	if search := c.Query("search"); search != "" {
		filter.Search = search
	}

	if limit := c.Query("limit"); limit != "" {
		if limit_v, err := strconv.Atoi(limit); err == nil && limit_v > 0 && limit_v <= 100 {
			filter.Limit = limit_v
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if offset_v, err := strconv.Atoi(offset); err == nil && offset_v >= 0 {
			filter.Offset = offset_v
		}
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		validSorts := []string{"title", "year", "rating", "difficulty", "created_at", "view_count"}
		for _, valid := range validSorts {
			if sortBy == valid {
				filter.SortBy = sortBy
				break
			}
		}
	}

	if sortDirection := c.Query("sort_direction"); sortDirection == "asc" || sortDirection == "desc" {
		filter.SortDirection = sortDirection
	}

	contents, total, err := r.service.GetContentList(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contents": contents,
		"total":    total,
		"limit":    filter.Limit,
		"offset":   filter.Offset,
	})
}

// GetContentEpisodes godoc
// @Summary      Get content episodes
// @Description  Get all episodes for a specific content item
// @Tags         episodes
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Router       /content/{id}/episodes [get]
func (r *Router) GetContentEpisodes(c *gin.Context) {
	contentID := c.Param("id")
	if contentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content ID required"})
		return
	}

	episodes, err := r.service.GetContentEpisodes(c.Request.Context(), contentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"episodes": episodes})
}

// CreateEpisode godoc
// @Summary      Create episode
// @Description  Create a new episode for a content item
// @Tags         episodes
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID"
// @Param        request body CreateEpisodeRequest true "Episode data"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /content/{id}/episodes [post]
func (r *Router) CreateEpisode(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	contentID := c.Param("id")
	if contentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content ID required"})
		return
	}

	var req CreateEpisodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	episode, err := r.service.CreateEpisode(c.Request.Context(), contentID, req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"episode": episode})
}

// UpdateEpisode godoc
// @Summary      Update episode
// @Description  Update an existing episode
// @Tags         episodes
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID"
// @Param        episode_id path string true "Episode ID"
// @Param        request body CreateEpisodeRequest true "Episode update data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /content/{id}/episodes/{episode_id} [put]
func (r *Router) UpdateEpisode(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	episodeID := c.Param("episode_id")
	if episodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Episode ID required"})
		return
	}

	var req CreateEpisodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	episode, err := r.service.UpdateEpisode(c.Request.Context(), episodeID, req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"episode": episode})
}

// DeleteEpisode godoc
// @Summary      Delete episode
// @Description  Delete an existing episode
// @Tags         episodes
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID"
// @Param        episode_id path string true "Episode ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /content/{id}/episodes/{episode_id} [delete]
func (r *Router) DeleteEpisode(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	episodeID := c.Param("episode_id")
	if episodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Episode ID required"})
		return
	}

	err := r.service.DeleteEpisode(c.Request.Context(), episodeID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Episode deleted successfully"})
}

// GetLanguages godoc
// @Summary      Get available languages
// @Description  Get list of all available languages
// @Tags         languages
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]string
// @Router       /content/languages [get]
func (r *Router) GetLanguages(c *gin.Context) {
	languages, err := r.service.GetLanguages(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"languages": languages})
}

// GetRecommendations godoc
// @Summary      Get content recommendations
// @Description  Get personalized content recommendations for the user
// @Tags         recommendations
// @Accept       json
// @Produce      json
// @Param        language_id query int false "Language ID for filtering"
// @Param        limit query int false "Number of recommendations (max 50)" default(10)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /content/recommendations [get]
func (r *Router) GetRecommendations(c *gin.Context) {
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

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	recommendations, err := r.service.GetRecommendations(c.Request.Context(), userID, languageID, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recommendations": recommendations})
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
