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

	// Public routes (phonetic information)
	public := v1.Group("/phonetic")
	{
		public.GET("/languages/:language_id/phonemes", phoneticRouter.GetPhonemes)
		public.GET("/languages/:language_id/phoneme/:phoneme_id", phoneticRouter.GetPhoneme)
		// public.GET("/languages/:language_id/alphabet", phoneticRouter.GetPhoneticAlphabet)
		// public.GET("/languages/:language_id/sounds", phoneticRouter.GetSoundCategories)
	}

	// Protected routes (user progress and practice)
	protected := v1.Group("/phonetic")
	protected.Use(phoneticRouter.jwtService.AuthMiddleware())
	{
		// Progress tracking
		protected.GET("/progress", phoneticRouter.GetPhoneticProgress)
		protected.GET("/progress/:language_id", phoneticRouter.GetLanguagePhoneticProgress)
		// protected.POST("/progress", phoneticRouter.UpdatePhoneticProgress)

		// Practice sessions
		// protected.GET("/practice/session", phoneticRouter.GetPracticeSession)
		// protected.POST("/practice/session", phoneticRouter.StartPracticeSession)
		// protected.PUT("/practice/session/:session_id", phoneticRouter.UpdatePracticeSession)
		protected.POST("/practice/session/:session_id/complete", phoneticRouter.CompletePracticeSession)

		// Exercises
		protected.GET("/exercises", phoneticRouter.GetExercises)
		// protected.GET("/exercises/:exercise_type", phoneticRouter.GetExercisesByType)
		// protected.POST("/exercises/:exercise_id/attempt", phoneticRouter.AttemptExercise)

		// Minimal pairs
		protected.GET("/minimal-pairs", phoneticRouter.GetMinimalPairs)
		// protected.POST("/minimal-pairs/practice", phoneticRouter.PracticeMinimalPairs)

		// Pronunciation assessment
		// protected.POST("/pronunciation/assess", phoneticRouter.AssessPronunciation)
		// protected.GET("/pronunciation/history", phoneticRouter.GetPronunciationHistory)

		// Learning path
		// protected.GET("/learning-path", phoneticRouter.GetLearningPath)
		// protected.POST("/learning-path/update", phoneticRouter.UpdateLearningPath)

		// Statistics and analytics
		protected.GET("/stats", phoneticRouter.GetPhoneticStats)
		// protected.GET("/analytics", phoneticRouter.GetPhoneticAnalytics)

		// Challenges and achievements
		// protected.GET("/challenges", phoneticRouter.GetPhoneticChallenges)
		// protected.POST("/challenges/:challenge_id/join", phoneticRouter.JoinChallenge)
		// protected.GET("/achievements", phoneticRouter.GetPhoneticAchievements)
	}

	return router
}

// Public routes - Phonetic information
func (r *Router) GetPhonemes(c *gin.Context) {
	languageID, err := strconv.Atoi(c.Param("language_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language ID"})
		return
	}

	var req GetPhonemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	phonemes, total, err := r.service.GetPhonemes(c.Request.Context(), languageID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"phonemes":    phonemes,
		"language_id": languageID,
		"count":       total,
	})
}

func (r *Router) GetPhoneme(c *gin.Context) {

	phonemeID, err := strconv.Atoi(c.Param("phoneme_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phoneme ID"})
		return
	}

	phoneme, err := r.service.GetPhonemeByID(c.Request.Context(), phonemeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"phoneme": phoneme})
}

// func (r *Router) GetPhoneticAlphabet(c *gin.Context) {
// 	languageID, err := strconv.Atoi(c.Param("language_id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language ID"})
// 		return
// 	}

// 	alphabet, err := r.service.GetPhoneticAlphabet(c.Request.Context(), languageID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"alphabet": alphabet})
// }

// func (r *Router) GetSoundCategories(c *gin.Context) {
// 	languageID, err := strconv.Atoi(c.Param("language_id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language ID"})
// 		return
// 	}

// 	categories, err := r.service.GetSoundCategories(c.Request.Context(), languageID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"categories": categories})
// }

// Protected routes - User progress
func (r *Router) GetPhoneticProgress(c *gin.Context) {
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

	progress, err := r.service.GetUserPhoneticProgress(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"progress": progress})
}

func (r *Router) GetLanguagePhoneticProgress(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	languageID, err := strconv.Atoi(c.Param("language_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language ID"})
		return
	}

	progress, err := r.service.GetUserPhoneticProgress(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"progress": progress})
}

// func (r *Router) UpdatePhoneticProgress(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	var req UpdateProgressRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if err := r.validator.Struct(req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	progress, err := r.service.UpdatePhoneticProgress(c.Request.Context(), userID, req)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message":  "Progress updated successfully",
// 		"progress": progress,
// 	})
// }

// Practice sessions
// func (r *Router) GetPracticeSession(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	languageID := 0
// 	if lang := c.Query("language_id"); lang != "" {
// 		if id, err := strconv.Atoi(lang); err == nil {
// 			languageID = id
// 		}
// 	}

// 	sessionType := c.Query("type")      // "discrimination", "production", "mixed"
// 	difficulty := c.Query("difficulty") // "beginner", "intermediate", "advanced"

// 	session, err := r.service.GetActivePracticeSession(c.Request.Context(), userID, languageID)
// 	if err != nil {
// 		// No active session, create a new one
// 		session, err = r.service.CreatePracticeSession(c.Request.Context(), userID, languageID, sessionType, difficulty)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 			return
// 		}
// 	}

// 	c.JSON(http.StatusOK, gin.H{"session": session})
// }

// func (r *Router) StartPracticeSession(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	var req StartPracticeRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if err := r.validator.Struct(req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	session, err := r.service.CreatePracticeSession(c.Request.Context(), userID, req.LanguageID, req.SessionType, req.Difficulty)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusCreated, gin.H{
// 		"message": "Practice session started",
// 		"session": session,
// 	})
// }

// func (r *Router) UpdatePracticeSession(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	sessionID := c.Param("session_id")
// 	if sessionID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
// 		return
// 	}

// 	var req UpdateSessionRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if err := r.validator.Struct(req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	session, err := r.service.UpdatePracticeSession(c.Request.Context(), userID, sessionID, req)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Session updated successfully",
// 		"session": session,
// 	})
// }

func (r *Router) CompletePracticeSession(c *gin.Context) {
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

	var req ExerciseSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := r.service.CompleteExerciseSession(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session completed successfully",
		"result":  result,
	})
}

// Exercises
func (r *Router) GetExercises(c *gin.Context) {

	languageID := 0
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = id
		}
	}

	difficulty := 1
	if diff := c.Query("difficulty"); diff != "" {
		if level, err := strconv.Atoi(diff); err == nil {
			difficulty = level
		}
	}
	category := c.Query("category")

	exercises, err := r.service.GetExercises(c.Request.Context(), languageID, category, difficulty)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exercises": exercises,
		"count":     len(exercises),
	})
}

// func (r *Router) GetExercisesByType(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	exerciseType := c.Param("exercise_type")
// 	if exerciseType == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Exercise type required"})
// 		return
// 	}

// 	languageID := 0
// 	if lang := c.Query("language_id"); lang != "" {
// 		if id, err := strconv.Atoi(lang); err == nil {
// 			languageID = id
// 		}
// 	}

// 	exercises, err := r.service.GetExercisesByType(c.Request.Context(), userID, languageID, exerciseType)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"exercises": exercises,
// 		"type":      exerciseType,
// 		"count":     len(exercises),
// 	})
// }

// func (r *Router) AttemptExercise(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	exerciseID := c.Param("exercise_id")
// 	if exerciseID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Exercise ID required"})
// 		return
// 	}

// 	var req ExerciseAttemptRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if err := r.validator.Struct(req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	result, err := r.service.AttemptExercise(c.Request.Context(), userID, exerciseID, req)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Exercise attempt recorded",
// 		"result":  result,
// 	})
// }

// Minimal pairs
func (r *Router) GetMinimalPairs(c *gin.Context) {
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

	phoneme1 := 1
	if pho1 := c.Query("phoneme1"); pho1 != "" {
		if level, err := strconv.Atoi(pho1); err == nil {
			phoneme1 = level
		}
	}
	phoneme2 := 1
	if pho2 := c.Query("phoneme2"); pho2 != "" {
		if level, err := strconv.Atoi(pho2); err == nil {
			phoneme2 = level
		}
	}

	pairs, err := r.service.GetMinimalPairs(c.Request.Context(), languageID, phoneme1, phoneme2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"minimal_pairs": pairs,
		"count":         len(pairs),
	})
}

// func (r *Router) PracticeMinimalPairs(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	var req MinimalPairPracticeRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if err := r.validator.Struct(req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	result, err := r.service.PracticeMinimalPairs(c.Request.Context(), userID, req)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Minimal pair practice completed",
// 		"result":  result,
// 	})
// }

// Pronunciation assessment
// func (r *Router) AssessPronunciation(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	var req PronunciationAssessmentRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if err := r.validator.Struct(req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	assessment, err := r.service.AssessPronunciation(c.Request.Context(), userID, req)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message":    "Pronunciation assessed",
// 		"assessment": assessment,
// 	})
// }

// func (r *Router) GetPronunciationHistory(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	languageID := 0
// 	if lang := c.Query("language_id"); lang != "" {
// 		if id, err := strconv.Atoi(lang); err == nil {
// 			languageID = id
// 		}
// 	}

// 	limit := 20
// 	if l := c.Query("limit"); l != "" {
// 		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
// 			limit = parsed
// 		}
// 	}

// 	history, err := r.service.GetPronunciationHistory(c.Request.Context(), userID, languageID, limit)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"history": history,
// 		"count":   len(history),
// 	})
// }

// Learning path
// func (r *Router) GetLearningPath(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	languageID := 0
// 	if lang := c.Query("language_id"); lang != "" {
// 		if id, err := strconv.Atoi(lang); err == nil {
// 			languageID = id
// 		}
// 	}

// 	if languageID == 0 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "language_id is required"})
// 		return
// 	}

// 	path, err := r.service.GetLearningPath(c.Request.Context(), userID, languageID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"learning_path": path})
// }

// func (r *Router) UpdateLearningPath(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	var req UpdateLearningPathRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if err := r.validator.Struct(req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	path, err := r.service.UpdateLearningPath(c.Request.Context(), userID, req)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message":       "Learning path updated",
// 		"learning_path": path,
// 	})
// }

// Statistics and analytics
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
	days := 30
	if d := c.Query("duage_id"); d != "" {
		if days_v, err := strconv.Atoi(d); err == nil {
			days = days_v
		}
	}

	stats, err := r.service.GetPhoneticStatistics(c.Request.Context(), userID, languageID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// func (r *Router) GetPhoneticAnalytics(c *gin.Context) {
// 	userID, exists := polyfyjwt.GetUserIDFromContext(c)
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
// 		return
// 	}

// 	languageID := 0
// 	if lang := c.Query("language_id"); lang != "" {
// 		if id, err := strconv.Atoi(lang); err == nil {
// 			languageID = id
// 		}
// 	}

// 	days := 30
// 	if d := c.Query("days"); d != "" {
// 		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
// 			days = parsed
// 		}
// 	}

// 	analytics, err := r.service.GetPhoneticAnalytics(c.Request.Context(), userID, languageID, days)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
// }

// // Challenges and achievements (placeholder implementations)
// func (r *Router) GetPhoneticChallenges(c *gin.Context) {
// 	c.JSON(http.StatusNotImplemented, gin.H{"message": "Phonetic challenges not implemented yet"})
// }

// func (r *Router) JoinChallenge(c *gin.Context) {
// 	c.JSON(http.StatusNotImplemented, gin.H{"message": "Challenge participation not implemented yet"})
// }

// func (r *Router) GetPhoneticAchievements(c *gin.Context) {
// 	c.JSON(http.StatusNotImplemented, gin.H{"message": "Phonetic achievements not implemented yet"})
// }

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
