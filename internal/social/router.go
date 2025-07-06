package social

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	polyfyjwt "github.com/Raylynd6299/babel/pkg/jwt"

	_ "github.com/Raylynd6299/babel/cmd/social-service/docs"
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

	socialRouter := &Router{
		service:    service,
		validator:  validator.New(),
		jwtService: jwtService,
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("social")))

	v1 := router.Group("/api/v1")

	// Public routes (profile viewing, group browsing)
	public := v1.Group("/social")
	{
		public.GET("/profiles/:user_id", socialRouter.GetPublicProfile)
		public.GET("/groups", socialRouter.GetGroups)
		public.GET("/groups/:id", socialRouter.GetPublicGroup)
		public.GET("/leaderboards/:type", socialRouter.GetLeaderboard)
		public.GET("/search/users", socialRouter.SearchUsers)
		public.GET("/search/groups", socialRouter.SearchGroups)
	}

	// Protected routes (require authentication)
	protected := v1.Group("/social")
	protected.Use(socialRouter.jwtService.AuthMiddleware())
	{
		// Profile management
		protected.GET("/profile", socialRouter.GetMyProfile)
		protected.PUT("/profile", socialRouter.UpdateProfile)
		protected.GET("/stats", socialRouter.GetMyStats)

		// Follow system
		protected.POST("/follow", socialRouter.FollowUser)
		protected.DELETE("/follow/:user_id", socialRouter.UnfollowUser)
		protected.GET("/followers", socialRouter.GetMyFollowers)
		protected.GET("/following", socialRouter.GetMyFollowing)
		protected.GET("/profiles/:user_id/followers", socialRouter.GetUserFollowers)
		protected.GET("/profiles/:user_id/following", socialRouter.GetUserFollowing)

		// Activity feed
		protected.GET("/feed", socialRouter.GetFeed)
		protected.POST("/activities", socialRouter.CreateActivity)
		protected.GET("/users/:user_id/activities", socialRouter.GetUserActivities)
		protected.POST("/activities/:id/like", socialRouter.LikeActivity)
		protected.DELETE("/activities/:id/like", socialRouter.UnlikeActivity)
		protected.GET("/activities/:id/likes", socialRouter.GetActivityLikes)

		// Study groups
		protected.POST("/groups", socialRouter.CreateGroup)
		protected.POST("/groups/:id/join", socialRouter.JoinGroup)
		protected.DELETE("/groups/:id/leave", socialRouter.LeaveGroup)
		protected.GET("/groups/:id/members", socialRouter.GetGroupMembers)
		protected.GET("/my-groups", socialRouter.GetMyGroups)

		// Language exchange
		protected.POST("/language-exchange", socialRouter.CreateLanguageExchange)
		protected.GET("/language-exchange", socialRouter.GetMyLanguageExchanges)
		protected.PUT("/language-exchange/:id/status", socialRouter.UpdateExchangeStatus)

		// Mentorship
		protected.POST("/mentorship", socialRouter.CreateMentorship)
		protected.GET("/mentorship", socialRouter.GetMyMentorships)
		protected.PUT("/mentorship/:id/status", socialRouter.UpdateMentorshipStatus)

		// Discovery and recommendations
		protected.GET("/recommendations/users", socialRouter.GetUserRecommendations)
		protected.GET("/recommendations/groups", socialRouter.GetGroupRecommendations)
		protected.GET("/discover", socialRouter.GetDiscoverContent)
	}

	return router
}

// GetPublicProfile godoc
// @Summary      Get public user profile
// @Description  Get public profile information for a specific user
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /social/profiles/{user_id} [get]
func (r *Router) GetPublicProfile(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	// Get viewer ID if authenticated (optional)
	viewerID, _ := polyfyjwt.GetUserIDFromContext(c)

	profile, err := r.service.GetProfile(c.Request.Context(), userID, viewerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

// GetGroups godoc
// @Summary      Get study groups
// @Description  Get paginated list of public study groups with filtering options
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        language_id query int false "Language ID for filtering"
// @Param        target_level query string false "Target level (beginner, intermediate, advanced)"
// @Param        is_public query bool false "Filter by public groups"
// @Param        has_space query bool false "Filter groups with available space"
// @Param        search query string false "Search term for group name or description"
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]string
// @Router       /social/groups [get]
func (r *Router) GetGroups(c *gin.Context) {
	filter := GroupFilter{
		Limit:  20,
		Offset: 0,
	}

	// Parse query parameters
	if languageID := c.Query("language_id"); languageID != "" {
		if id, err := strconv.Atoi(languageID); err == nil {
			filter.LanguageID = id
		}
	}

	if targetLevel := c.Query("target_level"); targetLevel != "" {
		filter.TargetLevel = targetLevel
	}

	if isPublic := c.Query("is_public"); isPublic != "" {
		if pub, err := strconv.ParseBool(isPublic); err == nil {
			filter.IsPublic = &pub
		}
	}

	if hasSpace := c.Query("has_space"); hasSpace == "true" {
		filter.HasSpace = true
	}

	if search := c.Query("search"); search != "" {
		filter.Search = search
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

	// Get user ID if authenticated (for computed fields)
	userID, _ := polyfyjwt.GetUserIDFromContext(c)

	groups, total, err := r.service.GetGroups(c.Request.Context(), filter, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": groups,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetPublicGroup godoc
// @Summary      Get public group details
// @Description  Get detailed information about a specific public study group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id path string true "Group ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /social/groups/{id} [get]
func (r *Router) GetPublicGroup(c *gin.Context) {
	groupID := c.Param("id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID required"})
		return
	}

	// Get user ID if authenticated (for computed fields)
	userID, _ := polyfyjwt.GetUserIDFromContext(c)

	group, err := r.service.GetGroup(c.Request.Context(), groupID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"group": group})
}

// GetLeaderboard godoc
// @Summary      Get leaderboard
// @Description  Get leaderboard rankings for different metrics and time periods
// @Tags         leaderboards
// @Accept       json
// @Produce      json
// @Param        type path string true "Leaderboard type (streak, points, vocabulary, phonetic)"
// @Param        language_id query int false "Language ID for filtering"
// @Param        period query string false "Time period (day, week, month, year)" default(week)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Router       /social/leaderboards/{type} [get]
func (r *Router) GetLeaderboard(c *gin.Context) {
	leaderboardType := c.Param("type")
	if leaderboardType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Leaderboard type required"})
		return
	}

	var languageID *int
	if lang := c.Query("language_id"); lang != "" {
		if id, err := strconv.Atoi(lang); err == nil {
			languageID = &id
		}
	}

	period := c.Query("period")
	if period == "" {
		period = "week"
	}

	// Get user ID if authenticated (to show user position)
	userID, _ := polyfyjwt.GetUserIDFromContext(c)

	leaderboard, err := r.service.GetLeaderboard(c.Request.Context(), leaderboardType, languageID, period, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
}

// SearchUsers godoc
// @Summary      Search users
// @Description  Search for users with filtering options by language, country, and search term
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        language_id query int false "Language ID for filtering"
// @Param        country_code query string false "Country code for filtering (ISO 3166-1 alpha-2)"
// @Param        search query string false "Search term for username or display name"
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]string
// @Router       /social/search/users [get]
func (r *Router) SearchUsers(c *gin.Context) {
	filter := UserFilter{
		Limit:  20,
		Offset: 0,
	}

	// Parse query parameters
	if languageID := c.Query("language_id"); languageID != "" {
		if id, err := strconv.Atoi(languageID); err == nil {
			filter.LanguageID = id
		}
	}

	if countryCode := c.Query("country_code"); countryCode != "" {
		filter.CountryCode = countryCode
	}

	if search := c.Query("search"); search != "" {
		filter.Search = search
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

	// Get searcher ID if authenticated
	searcherID, _ := polyfyjwt.GetUserIDFromContext(c)

	users, total, err := r.service.SearchUsers(c.Request.Context(), filter, searcherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// SearchGroups godoc
// @Summary      Search groups
// @Description  Search for study groups with filtering options (same as GetGroups but with search focus)
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        language_id query int false "Language ID for filtering"
// @Param        target_level query string false "Target level (beginner, intermediate, advanced)"
// @Param        is_public query bool false "Filter by public groups"
// @Param        has_space query bool false "Filter groups with available space"
// @Param        search query string false "Search term for group name or description"
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]string
// @Router       /social/search/groups [get]
func (r *Router) SearchGroups(c *gin.Context) {
	// Same as GetGroups but with search focus
	r.GetGroups(c)
}

// GetMyProfile godoc
// @Summary      Get my profile
// @Description  Get current user's complete social profile information
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/profile [get]
func (r *Router) GetMyProfile(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	profile, err := r.service.GetProfile(c.Request.Context(), userID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

// UpdateProfile godoc
// @Summary      Update my profile
// @Description  Update current user's social profile information and settings
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Param        request body UpdateProfileRequest true "Profile update data"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/profile [put]
func (r *Router) UpdateProfile(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := r.service.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

// GetMyStats godoc
// @Summary      Get my social statistics
// @Description  Get current user's comprehensive social statistics and metrics
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/stats [get]
func (r *Router) GetMyStats(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	stats, err := r.service.GetSocialStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// FollowUser godoc
// @Summary      Follow user
// @Description  Start following another user to see their activities in feed
// @Tags         follows
// @Accept       json
// @Produce      json
// @Param        request body object true "Follow user data"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/follow [post]
func (r *Router) FollowUser(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req FollowUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := r.service.FollowUser(c.Request.Context(), userID, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User followed successfully"})
}

// UnfollowUser godoc
// @Summary      Unfollow user
// @Description  Stop following a user and remove their activities from feed
// @Tags         follows
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID to unfollow"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/follow/{user_id} [delete]
func (r *Router) UnfollowUser(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	followingID := c.Param("user_id")
	if followingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	err := r.service.UnfollowUser(c.Request.Context(), userID, followingID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unfollowed successfully"})
}

// GetMyFollowers godoc
// @Summary      Get my followers
// @Description  Get paginated list of users who follow the current user
// @Tags         follows
// @Accept       json
// @Produce      json
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/followers [get]
func (r *Router) GetMyFollowers(c *gin.Context) {
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

	followers, err := r.service.GetFollowers(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"followers": followers})
}

// GetMyFollowing godoc
// @Summary      Get my following
// @Description  Get paginated list of users that the current user follows
// @Tags         follows
// @Accept       json
// @Produce      json
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/following [get]
func (r *Router) GetMyFollowing(c *gin.Context) {
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

	following, err := r.service.GetFollowing(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"following": following})
}

// GetUserFollowers godoc
// @Summary      Get user followers
// @Description  Get paginated list of users who follow a specific user
// @Tags         follows
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/profiles/{user_id}/followers [get]
func (r *Router) GetUserFollowers(c *gin.Context) {
	targetUserID := c.Param("user_id")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
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

	followers, err := r.service.GetFollowers(c.Request.Context(), targetUserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"followers": followers})
}

// GetUserFollowing godoc
// @Summary      Get user following
// @Description  Get paginated list of users that a specific user follows
// @Tags         follows
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/profiles/{user_id}/following [get]
func (r *Router) GetUserFollowing(c *gin.Context) {
	targetUserID := c.Param("user_id")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
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

	following, err := r.service.GetFollowing(c.Request.Context(), targetUserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"following": following})
}

// GetFeed godoc
// @Summary      Get activity feed
// @Description  Get personalized activity feed with posts from followed users and groups
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/feed [get]
func (r *Router) GetFeed(c *gin.Context) {
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

	activities, err := r.service.GetFeed(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

// CreateActivity godoc
// @Summary      Create activity
// @Description  Create a new activity post to share with followers and groups
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        request body object true "Activity creation data"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/activities [post]
func (r *Router) CreateActivity(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	activity, err := r.service.CreateActivity(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"activity": activity})
}

// GetUserActivities godoc
// @Summary      Get user activities
// @Description  Get paginated list of activities from a specific user
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/users/{user_id}/activities [get]
func (r *Router) GetUserActivities(c *gin.Context) {
	targetUserID := c.Param("user_id")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	viewerID, exists := polyfyjwt.GetUserIDFromContext(c)
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

	activities, err := r.service.GetUserActivities(c.Request.Context(), targetUserID, viewerID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

// LikeActivity godoc
// @Summary      Like activity
// @Description  Add a like to a specific activity post
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        id path string true "Activity ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/activities/{id}/like [post]
func (r *Router) LikeActivity(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	activityID := c.Param("id")
	if activityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Activity ID required"})
		return
	}

	err := r.service.LikeActivity(c.Request.Context(), userID, activityID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity liked successfully"})
}

// UnlikeActivity godoc
// @Summary      Unlike activity
// @Description  Remove a like from a specific activity post
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        id path string true "Activity ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/activities/{id}/like [delete]
func (r *Router) UnlikeActivity(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	activityID := c.Param("id")
	if activityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Activity ID required"})
		return
	}

	err := r.service.UnlikeActivity(c.Request.Context(), userID, activityID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity unliked successfully"})
}

// GetActivityLikes godoc
// @Summary      Get activity likes
// @Description  Get list of users who liked a specific activity post
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        id path string true "Activity ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /social/activities/{id}/likes [get]
func (r *Router) GetActivityLikes(c *gin.Context) {
	activityID := c.Param("id")
	if activityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Activity ID required"})
		return
	}

	likes, err := r.service.GetActivityLikes(c.Request.Context(), activityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"likes": likes})
}

// CreateGroup godoc
// @Summary      Create study group
// @Description  Create a new study group for language learning collaboration
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        request body object true "Group creation data"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/groups [post]
func (r *Router) CreateGroup(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := r.service.CreateGroup(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"group": group})
}

// JoinGroup godoc
// @Summary      Join study group
// @Description  Join an existing study group as a member
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id path string true "Group ID"
// @Param        request body object true "Join group data"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/groups/{id}/join [post]
func (r *Router) JoinGroup(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID required"})
		return
	}

	var req JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := r.service.JoinGroup(c.Request.Context(), userID, groupID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Joined group successfully"})
}

// LeaveGroup godoc
// @Summary      Leave study group
// @Description  Leave a study group that the user is currently a member of
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id path string true "Group ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/groups/{id}/leave [delete]
func (r *Router) LeaveGroup(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID required"})
		return
	}

	err := r.service.LeaveGroup(c.Request.Context(), userID, groupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left group successfully"})
}

// GetGroupMembers godoc
// @Summary      Get group members
// @Description  Get paginated list of members in a specific study group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id path string true "Group ID"
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        offset query int false "Number of items to skip" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /social/groups/{id}/members [get]
func (r *Router) GetGroupMembers(c *gin.Context) {
	groupID := c.Param("id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID required"})
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

	members, err := r.service.GetGroupMembers(c.Request.Context(), groupID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// GetMyGroups godoc
// @Summary      Get my groups
// @Description  Get all study groups that the current user is a member of
// @Tags         groups
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/my-groups [get]
func (r *Router) GetMyGroups(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	groups, err := r.service.GetMyGroups(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// CreateLanguageExchange godoc
// @Summary      Create language exchange
// @Description  Create a new language exchange partnership between two users
// @Tags         exchanges
// @Accept       json
// @Produce      json
// @Param        request body object true "Language exchange data"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/language-exchange [post]
func (r *Router) CreateLanguageExchange(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		PartnerID         string `json:"partner_id" validate:"required,uuid"`
		ITeachLanguage    int    `json:"i_teach_language" validate:"required"`
		ILearnLanguage    int    `json:"i_learn_language" validate:"required"`
		TheyTeachLanguage int    `json:"they_teach_language" validate:"required"`
		TheyLearnLanguage int    `json:"they_learn_language" validate:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exchange, err := r.service.CreateLanguageExchange(c.Request.Context(),
		userID, req.PartnerID, req.ITeachLanguage, req.ILearnLanguage,
		req.TheyTeachLanguage, req.TheyLearnLanguage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"exchange": exchange})
}

// GetMyLanguageExchanges godoc
// @Summary      Get my language exchanges
// @Description  Get all language exchange partnerships for the current user
// @Tags         exchanges
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/language-exchange [get]
func (r *Router) GetMyLanguageExchanges(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	exchanges, err := r.service.GetLanguageExchanges(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exchanges": exchanges})
}

// UpdateExchangeStatus godoc
// @Summary      Update exchange status
// @Description  Update the status of a language exchange partnership
// @Tags         exchanges
// @Accept       json
// @Produce      json
// @Param        id path string true "Exchange ID"
// @Param        request body object true "Status update data"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/language-exchange/{id}/status [put]
func (r *Router) UpdateExchangeStatus(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	exchangeID := c.Param("id")
	if exchangeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exchange ID required"})
		return
	}

	var req struct {
		Status string `json:"status" validate:"required,oneof=active paused completed cancelled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := r.service.UpdateExchangeStatus(c.Request.Context(), userID, exchangeID, req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Exchange status updated successfully"})
}

// CreateMentorship godoc
// @Summary      Create mentorship
// @Description  Create a new mentorship relationship between mentor and mentee
// @Tags         mentorship
// @Accept       json
// @Produce      json
// @Param        request body object true "Mentorship data"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/mentorship [post]
func (r *Router) CreateMentorship(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		MenteeID    string   `json:"mentee_id" validate:"required,uuid"`
		LanguageID  int      `json:"language_id" validate:"required"`
		Description string   `json:"description" validate:"max=1000"`
		Goals       []string `json:"goals" validate:"required,min=1,max=10"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mentorship, err := r.service.CreateMentorship(c.Request.Context(),
		userID, req.MenteeID, req.LanguageID, req.Description, req.Goals)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mentorship": mentorship})
}

// GetMyMentorships godoc
// @Summary      Get my mentorships
// @Description  Get all mentorship relationships for the current user (as mentor or mentee)
// @Tags         mentorship
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/mentorship [get]
func (r *Router) GetMyMentorships(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	mentorships, err := r.service.GetMentorships(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mentorships": mentorships})
}

// UpdateMentorshipStatus godoc
// @Summary      Update mentorship status
// @Description  Update the status of a mentorship relationship
// @Tags         mentorship
// @Accept       json
// @Produce      json
// @Param        id path string true "Mentorship ID"
// @Param        request body object true "Status update data"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /social/mentorship/{id}/status [put]
func (r *Router) UpdateMentorshipStatus(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	mentorshipID := c.Param("id")
	if mentorshipID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mentorship ID required"})
		return
	}

	var req struct {
		Status string `json:"status" validate:"required,oneof=active completed cancelled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := r.service.UpdateMentorshipStatus(c.Request.Context(), userID, mentorshipID, req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mentorship status updated successfully"})
}

// Discovery and recommendations
func (r *Router) GetUserRecommendations(c *gin.Context) {
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

	recommendations, err := r.service.GetUserRecommendations(c.Request.Context(), userID, languageID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recommendations": recommendations})
}

func (r *Router) GetGroupRecommendations(c *gin.Context) {
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

	recommendations, err := r.service.GetGroupRecommendations(c.Request.Context(), userID, languageID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recommendations": recommendations})
}

func (r *Router) GetDiscoverContent(c *gin.Context) {
	userID, exists := polyfyjwt.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	discover, err := r.service.GetDiscoverContent(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"discover": discover})
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
