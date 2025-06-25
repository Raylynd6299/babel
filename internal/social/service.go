package social

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/Raylynd6299/babel/internal/content"
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

// Profile management
func (s *Service) GetProfile(ctx context.Context, userID string, viewerID string) (*UserProfile, error) {
	var profile UserProfile
	err := s.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("profile not found")
		}
		return nil, err
	}

	// Get followers/following counts
	var followersCount, followingCount int64
	s.db.Model(&UserFollow{}).Where("following_id = ?", userID).Count(&followersCount)
	s.db.Model(&UserFollow{}).Where("follower_id = ?", userID).Count(&followingCount)

	profile.FollowersCount = int(followersCount)
	profile.FollowingCount = int(followingCount)

	// Check relationship with viewer
	if viewerID != "" && viewerID != userID {
		var followCount int64
		s.db.Model(&UserFollow{}).Where("follower_id = ? AND following_id = ?", viewerID, userID).Count(&followCount)
		profile.IsFollowing = followCount > 0

		s.db.Model(&UserFollow{}).Where("follower_id = ? AND following_id = ?", userID, viewerID).Count(&followCount)
		profile.IsFollower = followCount > 0
	}

	return &profile, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*UserProfile, error) {
	var profile UserProfile
	err := s.db.Where("user_id = ?", userID).First(&profile).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new profile
		profile = UserProfile{
			UserID: userID,
		}
	} else if err != nil {
		return nil, err
	}

	// Update fields
	if req.DisplayName != "" {
		profile.DisplayName = req.DisplayName
	}
	if req.Bio != "" {
		profile.Bio = req.Bio
	}
	if req.CountryCode != "" {
		profile.CountryCode = req.CountryCode
	}
	if req.TimeZone != "" {
		profile.TimeZone = req.TimeZone
	}
	if req.IsPublic != nil {
		profile.IsPublic = *req.IsPublic
	}
	if req.ShowProgress != nil {
		profile.ShowProgress = *req.ShowProgress
	}
	if req.ShowStreak != nil {
		profile.ShowStreak = *req.ShowStreak
	}
	if req.AllowMessages != nil {
		profile.AllowMessages = *req.AllowMessages
	}

	// Handle language arrays
	if len(req.NativeLanguages) > 0 {
		nativeJSON, _ := json.Marshal(req.NativeLanguages)
		profile.NativeLanguages = string(nativeJSON)
	}
	if len(req.LearningLanguages) > 0 {
		learningJSON, _ := json.Marshal(req.LearningLanguages)
		profile.LearningLanguages = string(learningJSON)
	}

	if profile.UserID == userID && err == gorm.ErrRecordNotFound {
		err = s.db.Create(&profile).Error
	} else {
		err = s.db.Save(&profile).Error
	}

	return &profile, err
}

// Follow system
func (s *Service) FollowUser(ctx context.Context, followerID, followingID string) error {
	if followerID == followingID {
		return errors.New("cannot follow yourself")
	}

	// Check if already following
	var existingFollow UserFollow
	err := s.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&existingFollow).Error
	if err == nil {
		return errors.New("already following this user")
	}

	follow := UserFollow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}

	err = s.db.Create(&follow).Error
	if err != nil {
		return err
	}

	// Create activity
	s.createActivity(followerID, "follow", "Started following a user", "", nil, true)

	return nil
}

func (s *Service) UnfollowUser(ctx context.Context, followerID, followingID string) error {
	result := s.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).Delete(&UserFollow{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("not following this user")
	}
	return nil
}

func (s *Service) GetFollowers(ctx context.Context, userID string, limit, offset int) ([]UserProfile, error) {
	var profiles []UserProfile

	query := `
        SELECT p.* FROM user_profiles p
        JOIN user_follows f ON p.user_id = f.follower_id
        WHERE f.following_id = ? AND p.is_public = true
        ORDER BY f.created_at DESC
        LIMIT ? OFFSET ?
    `

	err := s.db.Raw(query, userID, limit, offset).Scan(&profiles).Error
	return profiles, err
}

func (s *Service) GetFollowing(ctx context.Context, userID string, limit, offset int) ([]UserProfile, error) {
	var profiles []UserProfile

	query := `
        SELECT p.* FROM user_profiles p
        JOIN user_follows f ON p.user_id = f.following_id
        WHERE f.follower_id = ? AND p.is_public = true
        ORDER BY f.created_at DESC
        LIMIT ? OFFSET ?
    `

	err := s.db.Raw(query, userID, limit, offset).Scan(&profiles).Error
	return profiles, err
}

// Study Groups
func (s *Service) CreateGroup(ctx context.Context, userID string, req CreateGroupRequest) (*StudyGroup, error) {
	tagsJSON, _ := json.Marshal(req.Tags)

	group := StudyGroup{
		Name:             req.Name,
		Description:      req.Description,
		LanguageID:       req.LanguageID,
		TargetLevel:      req.TargetLevel,
		MaxMembers:       req.MaxMembers,
		IsPublic:         req.IsPublic,
		RequiresApproval: req.RequiresApproval,
		Tags:             string(tagsJSON),
		Rules:            req.Rules,
		CreatedBy:        userID,
	}

	if err := s.db.Create(&group).Error; err != nil {
		return nil, err
	}

	// Add creator as admin
	membership := GroupMembership{
		GroupID:  group.ID,
		UserID:   userID,
		Role:     "admin",
		Status:   "active",
		JoinedAt: time.Now(),
	}

	if err := s.db.Create(&membership).Error; err != nil {
		return nil, err
	}

	// Create activity
	s.createActivity(userID, "group_created", fmt.Sprintf("Created study group: %s", group.Name), "", &group.LanguageID, true)

	return &group, nil
}

func (s *Service) GetGroups(ctx context.Context, filter GroupFilter, userID string) ([]StudyGroup, int64, error) {
	query := s.db.Model(&StudyGroup{}).Preload("Language")

	if filter.LanguageID > 0 {
		query = query.Where("language_id = ?", filter.LanguageID)
	}

	if filter.TargetLevel != "" {
		query = query.Where("target_level = ?", filter.TargetLevel)
	}

	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}

	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)
	}

	if filter.HasSpace {
		// Subquery to get groups with available space
		subquery := s.db.Table("group_memberships").
			Select("group_id, COUNT(*) as member_count").
			Where("status = 'active'").
			Group("group_id")

		query = query.Joins("LEFT JOIN (?) as mc ON study_groups.id = mc.group_id", subquery).
			Where("COALESCE(mc.member_count, 0) < study_groups.max_members")
	}

	var total int64
	query.Count(&total)

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(20)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var groups []StudyGroup
	err := query.Order("created_at DESC").Find(&groups).Error
	if err != nil {
		return nil, 0, err
	}

	// Add computed fields
	for i := range groups {
		s.addGroupComputedFields(&groups[i], userID)
	}

	return groups, total, nil
}

func (s *Service) GetGroup(ctx context.Context, groupID string, userID string) (*StudyGroup, error) {
	var group StudyGroup
	err := s.db.Preload("Language").Where("id = ?", groupID).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("group not found")
		}
		return nil, err
	}

	s.addGroupComputedFields(&group, userID)
	return &group, nil
}

func (s *Service) addGroupComputedFields(group *StudyGroup, userID string) {
	// Get member count
	var memberCount int64
	s.db.Model(&GroupMembership{}).Where("group_id = ? AND status = 'active'", group.ID).Count(&memberCount)
	group.MemberCount = int(memberCount)

	if userID != "" {
		// Check membership
		var membership GroupMembership
		err := s.db.Where("group_id = ? AND user_id = ? AND status = 'active'", group.ID, userID).First(&membership).Error
		if err == nil {
			group.IsMember = true
			group.IsAdmin = membership.Role == "admin" || membership.Role == "moderator"
		}
	}
}

func (s *Service) JoinGroup(ctx context.Context, userID, groupID string, req JoinGroupRequest) error {
	// Check if group exists and has space
	var group StudyGroup
	err := s.db.Where("id = ?", groupID).First(&group).Error
	if err != nil {
		return errors.New("group not found")
	}

	// Check if already a member
	var existingMembership GroupMembership
	err = s.db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&existingMembership).Error
	if err == nil {
		return errors.New("already a member of this group")
	}

	// Check space
	var memberCount int64
	s.db.Model(&GroupMembership{}).Where("group_id = ? AND status = 'active'", groupID).Count(&memberCount)
	if int(memberCount) >= group.MaxMembers {
		return errors.New("group is full")
	}

	status := "active"
	if group.RequiresApproval {
		status = "pending"
	}

	membership := GroupMembership{
		GroupID:  groupID,
		UserID:   userID,
		Role:     "member",
		Status:   status,
		JoinedAt: time.Now(),
	}

	err = s.db.Create(&membership).Error
	if err != nil {
		return err
	}

	// Create activity
	if status == "active" {
		s.createActivity(userID, "group_joined", fmt.Sprintf("Joined study group: %s", group.Name), "", &group.LanguageID, true)
	}

	return nil
}

func (s *Service) LeaveGroup(ctx context.Context, userID, groupID string) error {
	result := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&GroupMembership{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("not a member of this group")
	}
	return nil
}

// Activity Feed
func (s *Service) CreateActivity(ctx context.Context, userID string, req CreateActivityRequest) (*ActivityFeed, error) {
	activity := ActivityFeed{
		UserID:      userID,
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		Data:        req.Data,
		LanguageID:  req.LanguageID,
		IsPublic:    req.IsPublic,
	}

	if err := s.db.Create(&activity).Error; err != nil {
		return nil, err
	}

	return &activity, nil
}

func (s *Service) createActivity(userID, activityType, title, description string, languageID *int, isPublic bool) {
	activity := ActivityFeed{
		UserID:      userID,
		Type:        activityType,
		Title:       title,
		Description: description,
		LanguageID:  languageID,
		IsPublic:    isPublic,
	}
	s.db.Create(&activity)
}

func (s *Service) GetFeed(ctx context.Context, userID string, limit, offset int) ([]ActivityFeed, error) {
	// Get activities from followed users and own activities
	var activities []ActivityFeed

	query := `
        SELECT a.*, p.username, p.avatar_url
        FROM activity_feeds a
        LEFT JOIN user_profiles p ON a.user_id = p.user_id
        WHERE a.is_public = true 
        AND (
            a.user_id = ? 
            OR a.user_id IN (
                SELECT following_id FROM user_follows WHERE follower_id = ?
            )
        )
        ORDER BY a.created_at DESC
        LIMIT ? OFFSET ?
    `

	err := s.db.Raw(query, userID, userID, limit, offset).Scan(&activities).Error
	return activities, err
}

func (s *Service) GetUserActivities(ctx context.Context, userID string, viewerID string, limit, offset int) ([]ActivityFeed, error) {
	var activities []ActivityFeed

	query := s.db.Model(&ActivityFeed{}).Where("user_id = ?", userID)

	// Only show public activities unless viewing own profile
	if viewerID != userID {
		query = query.Where("is_public = true")
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&activities).Error

	return activities, err
}

// Search and Discovery
func (s *Service) SearchUsers(ctx context.Context, filter UserFilter, searcherID string) ([]UserProfile, int64, error) {
	query := s.db.Model(&UserProfile{}).Where("is_public = true")

	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("username ILIKE ? OR display_name ILIKE ?", searchTerm, searchTerm)
	}

	if filter.CountryCode != "" {
		query = query.Where("country_code = ?", filter.CountryCode)
	}

	if filter.LanguageID > 0 {
		// Search in learning or native languages JSON
		query = query.Where("learning_languages::text LIKE ? OR native_languages::text LIKE ?",
			fmt.Sprintf("%%\"%d\"%%", filter.LanguageID),
			fmt.Sprintf("%%\"%d\"%%", filter.LanguageID))
	}

	var total int64
	query.Count(&total)

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(20)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var profiles []UserProfile
	err := query.Order("created_at DESC").Find(&profiles).Error

	return profiles, total, err
}

// Leaderboards
func (s *Service) GetLeaderboard(ctx context.Context, leaderboardType string, languageID *int, period string, userID string) (*LeaderboardResponse, error) {
	// Calculate date range based on period
	now := time.Now()
	var startDate, endDate time.Time

	switch period {
	case "week":
		startDate = now.AddDate(0, 0, -int(now.Weekday()))
		endDate = startDate.AddDate(0, 0, 7)
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0)
	case "year":
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(1, 0, 0)
	default: // all_time
		startDate = time.Date(2020, 1, 1, 0, 0, 0, 0, now.Location())
		endDate = now
	}

	var entries []LeaderboardEntry
	var err error

	switch leaderboardType {
	case "input_time":
		entries, err = s.getInputTimeLeaderboard(languageID, startDate, endDate)
	case "streak":
		entries, err = s.getStreakLeaderboard(languageID)
	case "vocabulary":
		entries, err = s.getVocabularyLeaderboard(languageID, startDate, endDate)
	case "social":
		entries, err = s.getSocialLeaderboard()
	default:
		return nil, errors.New("invalid leaderboard type")
	}

	if err != nil {
		return nil, err
	}

	// Add ranks
	for i := range entries {
		entries[i].Rank = i + 1
	}

	// Find user's position
	var userRank, userScore *int
	for i, entry := range entries {
		if entry.UserID == userID {
			rank := i + 1
			userRank = &rank
			userScore = &entry.Score
			break
		}
	}

	var language *content.Language
	if languageID != nil {
		language = &content.Language{}
		s.db.Where("id = ?", *languageID).First(language)
	}

	return &LeaderboardResponse{
		Type:      leaderboardType,
		Period:    period,
		Language:  language,
		StartDate: startDate,
		EndDate:   endDate,
		Entries:   entries,
		UserRank:  userRank,
		UserScore: userScore,
	}, nil
}

func (s *Service) getInputTimeLeaderboard(languageID *int, startDate, endDate time.Time) ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry

	query := `
       SELECT 
           up.user_id,
           p.username,
           p.display_name,
           p.avatar_url,
           SUM(up.duration_minutes) as score
       FROM user_progress up
       JOIN user_profiles p ON up.user_id = p.user_id
       WHERE up.watched_at BETWEEN ? AND ? AND p.is_public = true
   `

	args := []interface{}{startDate, endDate}

	if languageID != nil {
		query += " AND EXISTS (SELECT 1 FROM content c WHERE c.id = up.content_id AND c.language_id = ?)"
		args = append(args, *languageID)
	}

	query += `
       GROUP BY up.user_id, p.username, p.display_name, p.avatar_url
       HAVING SUM(up.duration_minutes) > 0
       ORDER BY score DESC
       LIMIT 100
   `

	err := s.db.Raw(query, args...).Scan(&entries).Error
	return entries, err
}

func (s *Service) getStreakLeaderboard(languageID *int) ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry

	query := `
       SELECT 
           us.user_id,
           p.username,
           p.display_name,
           p.avatar_url,
           us.current_streak_days as score
       FROM user_stats us
       JOIN user_profiles p ON us.user_id = p.user_id
       WHERE us.current_streak_days > 0 AND p.is_public = true
   `

	args := []interface{}{}

	if languageID != nil {
		query += " AND us.language_id = ?"
		args = append(args, *languageID)
	}

	query += `
       ORDER BY score DESC
       LIMIT 100
   `

	err := s.db.Raw(query, args...).Scan(&entries).Error
	return entries, err
}

func (s *Service) getVocabularyLeaderboard(languageID *int, startDate, endDate time.Time) ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry

	query := `
       SELECT 
           uv.user_id,
           p.username,
           p.display_name,
           p.avatar_url,
           COUNT(DISTINCT uv.vocabulary_id) as score
       FROM user_vocabulary uv
       JOIN vocabulary v ON uv.vocabulary_id = v.id
       JOIN user_profiles p ON uv.user_id = p.user_id
       WHERE uv.added_at BETWEEN ? AND ? AND p.is_public = true
   `

	args := []interface{}{startDate, endDate}

	if languageID != nil {
		query += " AND v.language_id = ?"
		args = append(args, *languageID)
	}

	query += `
       GROUP BY uv.user_id, p.username, p.display_name, p.avatar_url
       HAVING COUNT(DISTINCT uv.vocabulary_id) > 0
       ORDER BY score DESC
       LIMIT 100
   `

	err := s.db.Raw(query, args...).Scan(&entries).Error
	return entries, err
}

func (s *Service) getSocialLeaderboard() ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry

	query := `
       SELECT 
           p.user_id,
           p.username,
           p.display_name,
           p.avatar_url,
           (
               (SELECT COUNT(*) FROM user_follows WHERE following_id = p.user_id) * 2 +
               (SELECT COUNT(*) FROM activity_feeds WHERE user_id = p.user_id AND is_public = true) +
               (SELECT COUNT(*) FROM group_memberships gm 
                JOIN study_groups sg ON gm.group_id = sg.id 
                WHERE gm.user_id = p.user_id AND gm.status = 'active') * 3
           ) as score
       FROM user_profiles p
       WHERE p.is_public = true
       HAVING score > 0
       ORDER BY score DESC
       LIMIT 100
   `

	err := s.db.Raw(query).Scan(&entries).Error
	return entries, err
}

// Language Exchange
func (s *Service) CreateLanguageExchange(ctx context.Context, user1ID, user2ID string, user1Teach, user1Learn, user2Teach, user2Learn int) (*LanguageExchange, error) {
	if user1ID == user2ID {
		return nil, errors.New("cannot create exchange with yourself")
	}

	exchange := LanguageExchange{
		User1ID:            user1ID,
		User2ID:            user2ID,
		User1TeachLanguage: user1Teach,
		User1LearnLanguage: user1Learn,
		User2TeachLanguage: user2Teach,
		User2LearnLanguage: user2Learn,
		Status:             "pending",
	}

	if err := s.db.Create(&exchange).Error; err != nil {
		return nil, err
	}

	return &exchange, nil
}

func (s *Service) GetLanguageExchanges(ctx context.Context, userID string) ([]LanguageExchange, error) {
	var exchanges []LanguageExchange
	err := s.db.Where("user1_id = ? OR user2_id = ?", userID, userID).
		Order("created_at DESC").
		Find(&exchanges).Error

	return exchanges, err
}

// Mentorship
func (s *Service) CreateMentorship(ctx context.Context, mentorID, menteeID string, languageID int, description string, goals []string) (*Mentorship, error) {
	if mentorID == menteeID {
		return nil, errors.New("cannot mentor yourself")
	}

	goalsJSON, _ := json.Marshal(goals)

	mentorship := Mentorship{
		MentorID:    mentorID,
		MenteeID:    menteeID,
		LanguageID:  languageID,
		Description: description,
		Goals:       string(goalsJSON),
		Status:      "pending",
	}

	if err := s.db.Create(&mentorship).Error; err != nil {
		return nil, err
	}

	return &mentorship, nil
}

func (s *Service) GetMentorships(ctx context.Context, userID string) ([]Mentorship, error) {
	var mentorships []Mentorship
	err := s.db.Preload("Language").
		Where("mentor_id = ? OR mentee_id = ?", userID, userID).
		Order("created_at DESC").
		Find(&mentorships).Error

	return mentorships, err
}

// Social Stats
func (s *Service) GetSocialStats(ctx context.Context, userID string) (*SocialStatsResponse, error) {
	stats := &SocialStatsResponse{}

	// Followers/Following
	var count int64
	s.db.Model(&UserFollow{}).Where("following_id = ?", userID).Count(&count)
	stats.FollowersCount = int(count)

	s.db.Model(&UserFollow{}).Where("follower_id = ?", userID).Count(&count)
	stats.FollowingCount = int(count)

	// Groups
	s.db.Model(&GroupMembership{}).Where("user_id = ? AND status = 'active'", userID).Count(&count)
	stats.GroupsCount = int(count)

	// Activities
	s.db.Model(&ActivityFeed{}).Where("user_id = ?", userID).Count(&count)
	stats.ActivitiesCount = int(count)

	// Mentorships
	s.db.Model(&Mentorship{}).Where("mentor_id = ? OR mentee_id = ?", userID, userID).Count(&count)
	stats.MentorshipsCount = int(count)

	// Exchanges
	s.db.Model(&LanguageExchange{}).Where("user1_id = ? OR user2_id = ?", userID, userID).Count(&count)
	stats.ExchangesCount = int(count)

	// Calculate reputation score
	stats.ReputationScore = stats.FollowersCount*2 + stats.ActivitiesCount + stats.GroupsCount*3

	return stats, nil
}

// Group Members
func (s *Service) GetGroupMembers(ctx context.Context, groupID string, limit, offset int) ([]UserProfile, error) {
	var profiles []UserProfile

	query := `
       SELECT p.*, gm.role, gm.joined_at
       FROM user_profiles p
       JOIN group_memberships gm ON p.user_id = gm.user_id
       WHERE gm.group_id = ? AND gm.status = 'active'
       ORDER BY 
           CASE gm.role 
               WHEN 'admin' THEN 1 
               WHEN 'moderator' THEN 2 
               ELSE 3 
           END,
           gm.joined_at ASC
       LIMIT ? OFFSET ?
   `

	err := s.db.Raw(query, groupID, limit, offset).Scan(&profiles).Error
	return profiles, err
}

// User interactions (likes, comments)
func (s *Service) LikeActivity(ctx context.Context, userID, activityID string) error {
	interaction := UserInteraction{
		UserID:     userID,
		TargetType: "activity",
		TargetID:   activityID,
		Type:       "like",
	}

	return s.db.Create(&interaction).Error
}

func (s *Service) UnlikeActivity(ctx context.Context, userID, activityID string) error {
	result := s.db.Where("user_id = ? AND target_type = 'activity' AND target_id = ? AND type = 'like'",
		userID, activityID).Delete(&UserInteraction{})

	if result.RowsAffected == 0 {
		return errors.New("not liked")
	}

	return result.Error
}

func (s *Service) GetActivityLikes(ctx context.Context, activityID string) ([]UserProfile, error) {
	var profiles []UserProfile

	query := `
       SELECT p.* FROM user_profiles p
       JOIN user_interactions ui ON p.user_id = ui.user_id
       WHERE ui.target_type = 'activity' AND ui.target_id = ? AND ui.type = 'like'
       ORDER BY ui.created_at DESC
   `

	err := s.db.Raw(query, activityID).Scan(&profiles).Error
	return profiles, err
}

func (s *Service) GetMyGroups(ctx context.Context, userID string) ([]StudyGroup, error) {
	var groups []StudyGroup

	query := `
        SELECT sg.*, l.name as language_name, l.code as language_code
        FROM study_groups sg
        JOIN group_memberships gm ON sg.id = gm.group_id
        JOIN languages l ON sg.language_id = l.id
        WHERE gm.user_id = ? AND gm.status = 'active'
        ORDER BY gm.joined_at DESC
    `

	err := s.db.Raw(query, userID).Scan(&groups).Error
	if err != nil {
		return nil, err
	}

	// Add computed fields
	for i := range groups {
		s.addGroupComputedFields(&groups[i], userID)
	}

	return groups, nil
}

func (s *Service) UpdateExchangeStatus(ctx context.Context, userID, exchangeID, status string) error {
	var exchange LanguageExchange
	err := s.db.Where("id = ? AND (user1_id = ? OR user2_id = ?)", exchangeID, userID, userID).First(&exchange).Error
	if err != nil {
		return errors.New("exchange not found or access denied")
	}

	exchange.Status = status
	if status == "active" && exchange.StartedAt == nil {
		now := time.Now()
		exchange.StartedAt = &now
	} else if status == "completed" || status == "cancelled" {
		now := time.Now()
		exchange.EndedAt = &now
	}

	return s.db.Save(&exchange).Error
}

func (s *Service) UpdateMentorshipStatus(ctx context.Context, userID, mentorshipID, status string) error {
	var mentorship Mentorship
	err := s.db.Where("id = ? AND (mentor_id = ? OR mentee_id = ?)", mentorshipID, userID, userID).First(&mentorship).Error
	if err != nil {
		return errors.New("mentorship not found or access denied")
	}

	mentorship.Status = status
	if status == "active" && mentorship.StartedAt == nil {
		now := time.Now()
		mentorship.StartedAt = &now
	} else if status == "completed" || status == "cancelled" {
		now := time.Now()
		mentorship.EndedAt = &now
	}

	return s.db.Save(&mentorship).Error
}

func (s *Service) GetUserRecommendations(ctx context.Context, userID string, languageID int, limit int) ([]UserProfile, error) {
	var profiles []UserProfile

	query := `
        SELECT DISTINCT p.*
        FROM user_profiles p
        WHERE p.user_id != ? 
        AND p.is_public = true
        AND p.user_id NOT IN (
            SELECT following_id FROM user_follows WHERE follower_id = ?
        )
    `

	args := []interface{}{userID, userID}

	if languageID > 0 {
		query += ` AND (
            p.learning_languages::text LIKE ? 
            OR p.native_languages::text LIKE ?
        )`
		languagePattern := fmt.Sprintf("%%\"%d\"%%", languageID)
		args = append(args, languagePattern, languagePattern)
	}

	query += ` ORDER BY RANDOM() LIMIT ?`
	args = append(args, limit)

	err := s.db.Raw(query, args...).Scan(&profiles).Error
	return profiles, err
}

func (s *Service) GetGroupRecommendations(ctx context.Context, userID string, languageID int, limit int) ([]StudyGroup, error) {
	var groups []StudyGroup

	query := `
        SELECT sg.*, l.name as language_name, l.code as language_code
        FROM study_groups sg
        JOIN languages l ON sg.language_id = l.id
        WHERE sg.is_public = true
        AND sg.id NOT IN (
            SELECT group_id FROM group_memberships WHERE user_id = ?
        )
        AND (
            SELECT COUNT(*) FROM group_memberships 
            WHERE group_id = sg.id AND status = 'active'
        ) < sg.max_members
    `

	args := []interface{}{userID}

	if languageID > 0 {
		query += ` AND sg.language_id = ?`
		args = append(args, languageID)
	}

	query += ` ORDER BY sg.created_at DESC LIMIT ?`
	args = append(args, limit)

	err := s.db.Raw(query, args...).Scan(&groups).Error
	if err != nil {
		return nil, err
	}

	// Add computed fields
	for i := range groups {
		s.addGroupComputedFields(&groups[i], userID)
	}

	return groups, nil
}

func (s *Service) GetDiscoverContent(ctx context.Context, userID string) (*DiscoverContent, error) {
	discover := &DiscoverContent{}

	// Get popular groups (not member of)
	groups, err := s.GetGroupRecommendations(ctx, userID, 0, 5)
	if err != nil {
		return nil, err
	}
	discover.PopularGroups = groups

	// Get recommended users
	users, err := s.GetUserRecommendations(ctx, userID, 0, 5)
	if err != nil {
		return nil, err
	}
	discover.RecommendedUsers = users

	// Get recent public activities from network
	activities, err := s.GetFeed(ctx, userID, 10, 0)
	if err != nil {
		return nil, err
	}
	discover.RecentActivities = activities

	// Get trending languages (most active)
	var languages []content.Language
	query := `
        SELECT l.*, COUNT(af.id) as activity_count
        FROM languages l
        LEFT JOIN activity_feeds af ON l.id = af.language_id 
        WHERE af.created_at > ? AND af.is_public = true
        GROUP BY l.id, l.code, l.name
        ORDER BY activity_count DESC
        LIMIT 5
    `

	weekAgo := time.Now().AddDate(0, 0, -7)
	if err := s.db.Raw(query, weekAgo).Scan(&languages).Error; err != nil {
		return nil, err
	}
	discover.TrendingLanguages = languages

	return discover, nil
}
