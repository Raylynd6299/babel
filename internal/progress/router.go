package progress

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

	progressRouter := &Router{
		service:    service,
		validator:  validator.New(),
		jwtService: jwtService,
	}

	v1 := router.Group("/api/v1")

	// All progress routes require authentication
	protected := v1.Group("/progress")
	protected.Use(progressRouter.jwtService.AuthMiddleware())
	{
		// Input logging
		protected.POST("/input", progressRouter.LogInput)

		// Statistics and analytics
		protected.GET("/stats", progressRouter.GetUserStats)
		protected.GET("/analytics", progressRouter.GetProgressAnalytics)
		protected.GET("/recent", progressRouter.GetRecentActivity)
		protected.GET("/calendar", progressRouter.GetCalendarData)

		// Progress history
		protected.GET("/history", progressRouter.GetProgressHistory)
		protected.GET("/sessions", progressRouter.GetStudySessions)

		// Goals and streaks
		protected.GET("/streak", progressRouter.GetStreakInfo)
		protected.POST("/goals", progressRouter.SetGoals)
		protected.GET("/goals", progressRouter.GetGoals)

		// Reports
		protected.GET("/weekly-report", progressRouter.GetWeeklyReport)
		protected.GET("/monthly-report", progressRouter.GetMonthlyReport)
	}

	return router
}

// Input logging
func (r *Router) LogInput(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req LogInputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get language_id from query parameter or request body
	languageID := req.LanguageID
	if languageID == 0 {
		if lang := c.Query("language_id"); lang != "" {
			if id, err := strconv.Atoi(lang); err == nil {
				languageID = id
			}
		}
	}

	if languageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language_id is required"})
		return
	}

	progress, err := r.service.LogInput(c.Request.Context(), userID, languageID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Input logged successfully",
		"progress": progress,
	})
}

// Statistics
func (r *Router) GetUserStats(c *gin.Context) {
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

	stats, err := r.service.GetUserStats(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (r *Router) GetProgressAnalytics(c *gin.Context) {
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

	days := 30 // Default to 30 days
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	analytics, err := r.service.GetProgressAnalytics(c.Request.Context(), userID, languageID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

func (r *Router) GetRecentActivity(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	activity, err := r.service.GetRecentActivity(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recent_activity": activity})
}

func (r *Router) GetCalendarData(c *gin.Context) {
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

	// Default to current month
	year := time.Now().Year()
	month := int(time.Now().Month())

	if y := c.Query("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil && parsed >= 2020 && parsed <= 2030 {
			year = parsed
		}
	}

	if m := c.Query("month"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed >= 1 && parsed <= 12 {
			month = parsed
		}
	}

	calendar, err := r.service.GetCalendarData(c.Request.Context(), userID, languageID, year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"calendar": calendar})
}

// Progress history
func (r *Router) GetProgressHistory(c *gin.Context) {
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

	history, total, err := r.service.GetProgressHistory(c.Request.Context(), userID, languageID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (r *Router) GetStudySessions(c *gin.Context) {
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

	days := 7 // Default to last 7 days
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	sessions, err := r.service.GetStudySessions(c.Request.Context(), userID, languageID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// Streaks and goals
func (r *Router) GetStreakInfo(c *gin.Context) {
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

	streak, err := r.service.GetStreakInfo(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"streak": streak})
}

func (r *Router) SetGoals(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req SetGoalsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := r.service.SetGoals(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Goals updated successfully"})
}

func (r *Router) GetGoals(c *gin.Context) {
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

	goals, err := r.service.GetGoals(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"goals": goals})
}

// Reports
func (r *Router) GetWeeklyReport(c *gin.Context) {
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

	report, err := r.service.GetWeeklyReport(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"weekly_report": report})
}

func (r *Router) GetMonthlyReport(c *gin.Context) {
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

	report, err := r.service.GetMonthlyReport(c.Request.Context(), userID, languageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"monthly_report": report})
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
