package phonetic

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

	phoneticRouter := &Router{
		service:    service,
		validator:  validator.New(),
		jwtService: jwtService,
	}

	v1 := router.Group("/api/v1")

	// Public routes (phoneme information)
	public := v1.Group("/phonetic")
	{
		public.GET("/languages/:language_id/phonemes", phoneticRouter.GetPhonemes)
		public.GET("/phonemes/:id", phoneticRouter.GetPhoneme)
		public.GET("/languages/:language_id/minimal-pairs", phoneticRouter.GetMinimalPairs)
		public.GET("/exercises", phoneticRouter.GetExercises)
		public.GET("/exercises/:id", phoneticRouter.GetExercise)
	}

	// Protected routes (user progress and practice)
	protected := v1.Group("/phonetic")
	protected.Use(phoneticRouter.jwtService.AuthMiddleware())
	{
		// Progress tracking
		protected.GET("/progress", phoneticRouter.GetUserProgress)
		protected.POST("/practice", phoneticRouter.PracticePhoneme)
		protected.GET("/stats", phoneticRouter.GetPhoneticStats)

		// Exercise sessions
		protected.POST("/exercises/:id/start", phoneticRouter.StartExercise)
		protected.POST("/sessions/:session_id/complete", phoneticRouter.CompleteExercise)
		protected.GET("/sessions", phoneticRouter.GetUserSessions)

		// Recommendations
		protected.GET("/recommendations", phoneticRouter.GetRecommendations)
		protected.GET("/weak-phonemes", phoneticRouter.GetWeakPhonemes)

		// Practice plans
		protected.GET("/practice-plan", phoneticRouter.GetPracticePlan)
		protected.POST("/practice-plan", phoneticRouter.CreatePracticePlan)
	}

	return router
}

// Public handlers
func (r *Router) GetPhonemes(c *gin.Context) {
	languageIDStr := c.Param("language_id")
	languageID, err := strconv.Atoi(languageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language ID"})
		return
	}

	phonemes, err := r.service.GetPhonemesByLanguage(c.Request.Context(), languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"phonemes": phonemes})
}

func (r *Router) GetPhoneme(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phoneme ID"})
		return
	}

	phoneme, err := r.service.GetPhoneme(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"phoneme": phoneme})
}

func (r *Router) GetMinimalPairs(c *gin.Context) {
	languageIDStr := c.Param("language_id")
	languageID, err := strconv.Atoi(languageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language ID"})
		return
	}

	phoneme1ID := 0
	if p1 := c.Query("phoneme1_id"); p1 != "" {
		if id, err := strconv.Atoi(p1); err == nil {
			phoneme1ID = id
		}
	}

	phoneme2ID := 0
	if p2 := c.Query("phoneme2_id"); p2 != "" {
		if id, err := strconv.Atoi(p2); err == nil {
			phoneme2ID = id
		}
	}

	pairs, err := r.service.GetMinimalPairs(c.Request.Context(), languageID, phoneme1ID, phoneme2ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"minimal_pairs": pairs})
}

func (r *Router) GetExercises(c *gin.Context) {
	filter := ExerciseFilter{
		Limit:  20,
		Offset: 0,
	}

	// Parse query parameters
	if languageID := c.Query("language_id"); languageID != "" {
		if id, err := strconv.Atoi(languageID); err == nil {
			filter.LanguageID = id
		}
	}

	if phonemeID := c.Query("phoneme_id"); phonemeID != "" {
		if id, err := strconv.Atoi(phonemeID); err == nil {
			filter.PhonemeID = id
		}
	}

	if exerciseType := c.Query("type"); exerciseType != "" {
		filter.Type = exerciseType
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	exercises, total, err := r.service.GetExercises(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exercises": exercises,
		"total":     total,
		"limit":     filter.Limit,
		"offset":    filter.Offset,
	})
}

func (r *Router) GetExercise(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exercise ID required"})
		return
	}

	exercise, err := r.service.GetExercise(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exercise": exercise})
}

// Protected handlers
func (r *Router) GetUserProgress(c *gin.Context) {
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

	progress, err := r.service.GetUserProgress(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"progress": progress})
}

func (r *Router) PracticePhoneme(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req PracticePhonemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := r.service.PracticePhoneme(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Practice recorded successfully"})
}

func (r *Router) GetPhoneticStats(c *gin.Context) {
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

	stats, err := r.service.GetPhoneticStats(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (r *Router) StartExercise(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	exerciseID := c.Param("id")
	if exerciseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exercise ID required"})
		return
	}

	session, err := r.service.StartExercise(c.Request.Context(), userID, exerciseID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"session": session})
}

func (r *Router) CompleteExercise(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	var req ExerciseCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Override session ID from URL
	req.SessionID = sessionID

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := r.service.CompleteExercise(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exercise completed successfully",
		"session": session,
	})
}

func (r *Router) GetUserSessions(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
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

	sessions, err := r.service.GetUserSessions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

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

	if languageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language_id is required"})
		return
	}

	limit := 5
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	recommendations, err := r.service.GetRecommendations(c.Request.Context(), userID, languageID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recommendations": recommendations})
}

func (r *Router) GetWeakPhonemes(c *gin.Context) {
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

	limit := 5
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	weakPhonemes, err := r.service.GetWeakPhonemes(c.Request.Context(), userID, languageID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"weak_phonemes": weakPhonemes})
}

func (r *Router) GetPracticePlan(c *gin.Context) {
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

	plan, err := r.service.GetPracticePlan(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"practice_plan": plan})
}

func (r *Router) CreatePracticePlan(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreatePracticePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plan, err := r.service.CreatePracticePlan(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Practice plan created successfully",
		"practice_plan": plan,
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
