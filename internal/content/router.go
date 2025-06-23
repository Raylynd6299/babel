package content

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
)

type Router struct {
	service   *Service
	validator *validator.Validate
}

func NewRouter(service *Service) *gin.Engine {
	router := gin.Default()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	contentRouter := &Router{
		service:   service,
		validator: validator.New(),
	}

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
	protected.Use(AuthMiddleware(contentRouter.service.jwtSecret))
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

// Content CRUD handlers
func (r *Router) CreateContent(c *gin.Context) {
	userID, exists := c.Get("user_id")
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

	content, err := r.service.CreateContent(c.Request.Context(), req, userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"content": content})
}

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

func (r *Router) UpdateContent(c *gin.Context) {
	userID, exists := c.Get("user_id")
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

	content, err := r.service.UpdateContent(c.Request.Context(), id, req, userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content})
}

func (r *Router) DeleteContent(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content ID required"})
		return
	}

	err := r.service.DeleteContent(c.Request.Context(), id, userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Content deleted successfully"})
}

func (r *Router) RateContent(c *gin.Context) {
	userID, exists := c.Get("user_id")
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
		UserID:              userID.(string),
		ContentID:           contentID,
		DifficultyRating:    req.DifficultyRating,
		UsefulnessRating:    req.UsefulnessRating,
		EntertainmentRating: req.EntertainmentRating,
		ReviewText:          req.ReviewText,
	}

	err := r.service.RateContent(c.Request.Context(), userID.(string), contentID, rating)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Content rated successfully"})
}

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

// Episode handlers
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

func (r *Router) CreateEpisode(c *gin.Context) {
	userID, exists := c.Get("user_id")
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

	episode, err := r.service.CreateEpisode(c.Request.Context(), contentID, req, userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"episode": episode})
}

func (r *Router) UpdateEpisode(c *gin.Context) {
	userID, exists := c.Get("user_id")
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

	episode, err := r.service.UpdateEpisode(c.Request.Context(), episodeID, req, userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"episode": episode})
}

func (r *Router) DeleteEpisode(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	episodeID := c.Param("episode_id")
	if episodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Episode ID required"})
		return
	}

	err := r.service.DeleteEpisode(c.Request.Context(), episodeID, userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Episode deleted successfully"})
}

// Other handlers
func (r *Router) GetLanguages(c *gin.Context) {
	languages, err := r.service.GetLanguages(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"languages": languages})
}

func (r *Router) GetRecommendations(c *gin.Context) {
	userID, exists := c.Get("user_id")
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

	recommendations, err := r.service.GetRecommendations(c.Request.Context(), userID.(string), languageID, limit)
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

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
			c.Set("user_id", claims.Subject)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Next()
	}
}
