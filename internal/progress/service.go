// internal/progress/service.go
package progress

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
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

func (s *Service) LogInput(ctx context.Context, userID string, languageID int, req LogInputRequest) (*UserProgress, error) {
	// Create progress entry
	progress := UserProgress{
		UserID:                  userID,
		ContentID:               req.ContentID,
		EpisodeID:               &req.EpisodeID,
		DurationMinutes:         req.DurationMinutes,
		ComprehensionPercentage: req.ComprehensionPercentage,
		DifficultyRating:        req.DifficultyRating,
		EnjoymentRating:         req.EnjoymentRating,
		Notes:                   req.Notes,
		Completed:               req.Completed,
		WatchedAt:               time.Now(),
	}

	if err := s.db.Create(&progress).Error; err != nil {
		return nil, err
	}

	// Update user stats
	if err := s.updateUserStats(userID, languageID, req.DurationMinutes); err != nil {
		return nil, err
	}

	return &progress, nil
}

func (s *Service) updateUserStats(userID string, languageID int, durationMinutes int) error {
	var stats UserStats
	err := s.db.Where("user_id = ? AND language_id = ?", userID, languageID).First(&stats).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new stats
		stats = UserStats{
			UserID:            userID,
			LanguageID:        languageID,
			TotalInputMinutes: durationMinutes,
			CurrentStreakDays: 1,
			LongestStreakDays: 1,
			TotalPoints:       durationMinutes, // 1 point per minute
			CurrentLevel:      1,
		}
		return s.db.Create(&stats).Error
	} else if err != nil {
		return err
	}

	// Update existing stats
	stats.TotalInputMinutes += durationMinutes
	stats.TotalPoints += durationMinutes

	// Update streak
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	var lastActivity time.Time
	s.db.Model(&UserProgress{}).
		Where("user_id = ?", userID).
		Select("MAX(watched_at)").
		Scan(&lastActivity)

	lastActivityDate := lastActivity.Format("2006-01-02")

	if lastActivityDate == yesterday {
		stats.CurrentStreakDays++
	} else if lastActivityDate != today {
		stats.CurrentStreakDays = 1
	}

	if stats.CurrentStreakDays > stats.LongestStreakDays {
		stats.LongestStreakDays = stats.CurrentStreakDays
	}

	// Calculate level (every 50 hours = 1 level)
	stats.CurrentLevel = (stats.TotalInputMinutes / (50 * 60)) + 1

	return s.db.Save(&stats).Error
}

func (s *Service) GetUserStats(ctx context.Context, userID string, languageID int) (*UserStats, error) {
	var stats UserStats
	err := s.db.Where("user_id = ? AND language_id = ?", userID, languageID).First(&stats).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &UserStats{
				UserID:     userID,
				LanguageID: languageID,
			}, nil
		}
		return nil, err
	}
	return &stats, nil
}

func (s *Service) GetProgressAnalytics(ctx context.Context, userID string, languageID int, days int) (*ProgressAnalytics, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Get all progress entries for the period
	var progressEntries []UserProgress
	err := s.db.Where("user_id = ? AND watched_at >= ?", userID, startDate).
		Find(&progressEntries).Error
	if err != nil {
		return nil, err
	}

	analytics := &ProgressAnalytics{
		ContentTypeBreakdown: make(map[string]int),
		StudyTimeByHour:      make(map[int]int),
	}

	// Calculate basic metrics
	totalMinutes := 0
	sessionCount := len(progressEntries)
	comprehensionSum := 0
	comprehensionCount := 0

	for _, entry := range progressEntries {
		totalMinutes += entry.DurationMinutes

		if entry.ComprehensionPercentage > 0 {
			comprehensionSum += entry.ComprehensionPercentage
			comprehensionCount++
		}

		// Track study time by hour
		hour := entry.WatchedAt.Hour()
		analytics.StudyTimeByHour[hour] += entry.DurationMinutes
	}

	analytics.TotalInputHours = float64(totalMinutes) / 60
	if sessionCount > 0 {
		analytics.AverageSessionMinutes = float64(totalMinutes) / float64(sessionCount)
	}

	// Get streak info from user stats
	stats, err := s.GetUserStats(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}
	analytics.CurrentStreak = stats.CurrentStreakDays
	analytics.LongestStreak = stats.LongestStreakDays

	// Get weekly progress
	analytics.WeeklyProgress = s.getWeeklyProgress(userID, days)

	// Get content type breakdown
	analytics.ContentTypeBreakdown = s.getContentTypeBreakdown(userID, startDate)

	// Get comprehension trend
	analytics.ComprehensionTrend = s.getComprehensionTrend(userID, days)

	// Get most watched content
	analytics.MostWatchedContent = s.getMostWatchedContent(userID, startDate, 10)

	return analytics, nil
}

func (s *Service) getWeeklyProgress(userID string, days int) []WeeklyData {
	var results []WeeklyData

	query := `
        SELECT 
            DATE_TRUNC('week', watched_at) as week,
            SUM(duration_minutes) as minutes,
            COUNT(*) as sessions
        FROM user_progress 
        WHERE user_id = ? AND watched_at >= ?
        GROUP BY DATE_TRUNC('week', watched_at)
        ORDER BY week
    `

	startDate := time.Now().AddDate(0, 0, -days)
	s.db.Raw(query, userID, startDate).Scan(&results)

	return results
}

func (s *Service) getContentTypeBreakdown(userID string, startDate time.Time) map[string]int {
	var results []struct {
		ContentType string `json:"content_type"`
		Minutes     int    `json:"minutes"`
	}

	query := `
        SELECT c.content_type, SUM(up.duration_minutes) as minutes
        FROM user_progress up
        JOIN content c ON up.content_id = c.id
        WHERE up.user_id = ? AND up.watched_at >= ?
        GROUP BY c.content_type
    `

	s.db.Raw(query, userID, startDate).Scan(&results)

	breakdown := make(map[string]int)
	for _, result := range results {
		breakdown[result.ContentType] = result.Minutes
	}

	return breakdown
}

func (s *Service) getComprehensionTrend(userID string, days int) []ComprehensionData {
	var results []ComprehensionData

	query := `
        SELECT 
            DATE(watched_at) as date,
            AVG(comprehension_percentage) as comprehension
        FROM user_progress 
        WHERE user_id = ? AND watched_at >= ? AND comprehension_percentage > 0
        GROUP BY DATE(watched_at)
        ORDER BY date
    `

	startDate := time.Now().AddDate(0, 0, -days)
	s.db.Raw(query, userID, startDate).Scan(&results)

	return results
}

func (s *Service) getMostWatchedContent(userID string, startDate time.Time, limit int) []ContentSummary {
	var results []ContentSummary

	query := `
        SELECT 
            up.content_id,
            c.title,
            SUM(up.duration_minutes) as total_minutes,
            COUNT(*) as sessions
        FROM user_progress up
        JOIN content c ON up.content_id = c.id
        WHERE up.user_id = ? AND up.watched_at >= ?
        GROUP BY up.content_id, c.title
        ORDER BY total_minutes DESC
        LIMIT ?
    `

	s.db.Raw(query, userID, startDate, limit).Scan(&results)

	return results
}

func (s *Service) GetRecentActivity(ctx context.Context, userID string, limit int) ([]UserProgress, error) {
	var progress []UserProgress
	err := s.db.Where("user_id = ?", userID).
		Order("watched_at DESC").
		Limit(limit).
		Find(&progress).Error

	return progress, err
}

func (s *Service) GetCalendarData(ctx context.Context, userID string, languageID int, year int, month int) ([]CalendarDay, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	var dailyProgress []struct {
		Date    time.Time `json:"date"`
		Minutes int       `json:"minutes"`
	}

	query := `
        SELECT DATE(watched_at) as date, SUM(duration_minutes) as minutes
        FROM user_progress up
        WHERE up.user_id = ? AND DATE(watched_at) BETWEEN ? AND ?
        GROUP BY DATE(watched_at)
    `

	err := s.db.Raw(query, userID, startDate, endDate).Scan(&dailyProgress).Error
	if err != nil {
		return nil, err
	}

	// Get user goals
	var stats UserStats
	s.db.Where("user_id = ? AND language_id = ?", userID, languageID).First(&stats)
	dailyGoal := 60
	if stats.ID != "" {
		dailyGoal = stats.DailyGoalMinutes
	}

	// Build calendar data
	calendar := make([]CalendarDay, 0)
	progressMap := make(map[string]int)

	for _, p := range dailyProgress {
		progressMap[p.Date.Format("2006-01-02")] = p.Minutes
	}

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		minutes := progressMap[dateStr]

		calendar = append(calendar, CalendarDay{
			Date:    dateStr,
			Minutes: minutes,
			HasGoal: true,
			MetGoal: minutes >= dailyGoal,
		})
	}

	return calendar, nil
}

func (s *Service) GetProgressHistory(ctx context.Context, userID string, languageID int, limit int, offset int) ([]UserProgress, int64, error) {
	var progress []UserProgress
	var total int64

	query := s.db.Where("user_id = ?", userID)
	if languageID > 0 {
		// We'll need to join with content table to filter by language
		query = query.Joins("JOIN content ON user_progress.content_id = content.id").
			Where("content.language_id = ?", languageID)
	}

	query.Model(&UserProgress{}).Count(&total)

	err := query.Order("watched_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&progress).Error

	return progress, total, err
}

func (s *Service) GetStudySessions(ctx context.Context, userID string, languageID int, days int) ([]StudySession, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	var sessions []StudySession

	query := `
        SELECT 
            DATE(up.watched_at) as date,
            SUM(up.duration_minutes) as total_minutes,
            COUNT(*) as session_count,
            AVG(up.comprehension_percentage) as avg_comprehension
        FROM user_progress up
    `

	if languageID > 0 {
		query += " JOIN content c ON up.content_id = c.id WHERE up.user_id = ? AND c.language_id = ? AND up.watched_at >= ?"
		err := s.db.Raw(query+" GROUP BY DATE(up.watched_at) ORDER BY date DESC", userID, languageID, startDate).Scan(&sessions).Error
		return sessions, err
	} else {
		query += " WHERE up.user_id = ? AND up.watched_at >= ?"
		err := s.db.Raw(query+" GROUP BY DATE(up.watched_at) ORDER BY date DESC", userID, startDate).Scan(&sessions).Error
		return sessions, err
	}
}

func (s *Service) GetStreakInfo(ctx context.Context, userID string, languageID int) (*StreakInfo, error) {
	stats, err := s.GetUserStats(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}

	// Get last activity date
	var lastActivity time.Time
	query := s.db.Model(&UserProgress{}).Where("user_id = ?", userID)
	if languageID > 0 {
		query = query.Joins("JOIN content ON user_progress.content_id = content.id").
			Where("content.language_id = ?", languageID)
	}
	query.Select("MAX(watched_at)").Scan(&lastActivity)

	// Calculate streak start date (approximate)
	streakStartDate := time.Now().AddDate(0, 0, -stats.CurrentStreakDays)

	// Check if active today
	today := time.Now().Format("2006-01-02")
	lastActivityDate := lastActivity.Format("2006-01-02")
	isActiveToday := today == lastActivityDate

	return &StreakInfo{
		CurrentStreak:    stats.CurrentStreakDays,
		LongestStreak:    stats.LongestStreakDays,
		LastActivityDate: lastActivity,
		StreakStartDate:  streakStartDate,
		IsActiveToday:    isActiveToday,
	}, nil
}

func (s *Service) SetGoals(ctx context.Context, userID string, req SetGoalsRequest) error {
	// For now, we'll store goals in user_stats table
	// In a real implementation, you might want a separate goals table
	var stats UserStats
	err := s.db.Where("user_id = ? AND language_id = ?", userID, req.LanguageID).First(&stats).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new stats record
		stats = UserStats{
			UserID:           userID,
			LanguageID:       req.LanguageID,
			DailyGoalMinutes: req.DailyGoalMinutes,
			WeeklyGoalHours:  req.WeeklyGoalHours,
			MonthlyGoalHours: req.MonthlyGoalHours,
		}
		return s.db.Create(&stats).Error
	} else if err != nil {
		return err
	}

	// Update existing stats with new goals
	updates := map[string]interface{}{
		"daily_goal_minutes": req.DailyGoalMinutes,
		"weekly_goal_hours":  req.WeeklyGoalHours,
		"monthly_goal_hours": req.MonthlyGoalHours,
		"updated_at":         time.Now(),
	}

	return s.db.Model(&stats).Updates(updates).Error
}

func (s *Service) GetGoals(ctx context.Context, userID string, languageID int) (*GoalsResponse, error) {
	stats, err := s.GetUserStats(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}

	// Calculate current progress
	now := time.Now()

	// Daily progress (today)
	dailyProgress := s.getDailyProgress(userID, languageID, now)

	// Weekly progress (this week)
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weeklyProgress := s.getWeeklyProgressHours(userID, languageID, weekStart)

	// Monthly progress (this month)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthlyProgress := s.getMonthlyProgressHours(userID, languageID, monthStart)

	// USAR stats PARA OBTENER LAS METAS (en lugar de valores por defecto)
	dailyGoal := 60   // Default
	weeklyGoal := 7   // Default
	monthlyGoal := 30 // Default

	// Si el modelo UserStats tiene los campos de metas, usarlos:
	if stats.DailyGoalMinutes > 0 {
		dailyGoal = stats.DailyGoalMinutes
	}
	if stats.WeeklyGoalHours > 0 {
		weeklyGoal = stats.WeeklyGoalHours
	}
	if stats.MonthlyGoalHours > 0 {
		monthlyGoal = stats.MonthlyGoalHours
	}

	return &GoalsResponse{
		LanguageID:             languageID,
		DailyGoalMinutes:       dailyGoal,
		WeeklyGoalHours:        weeklyGoal,
		MonthlyGoalHours:       monthlyGoal,
		DailyProgress:          dailyProgress,
		WeeklyProgress:         weeklyProgress,
		MonthlyProgress:        monthlyProgress,
		DailyProgressPercent:   float64(dailyProgress) / float64(dailyGoal) * 100,
		WeeklyProgressPercent:  weeklyProgress / float64(weeklyGoal) * 100,
		MonthlyProgressPercent: monthlyProgress / float64(monthlyGoal) * 100,
	}, nil
}

func (s *Service) GetWeeklyReport(ctx context.Context, userID string, languageID int) (*WeeklyReport, error) {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weekEnd := weekStart.AddDate(0, 0, 6)

	// Get weekly stats
	var weeklyStats struct {
		TotalMinutes     int     `json:"total_minutes"`
		SessionCount     int     `json:"session_count"`
		AvgComprehension float64 `json:"avg_comprehension"`
	}

	query := `
       SELECT 
           SUM(duration_minutes) as total_minutes,
           COUNT(*) as session_count,
           AVG(comprehension_percentage) as avg_comprehension
       FROM user_progress up
   `

	if languageID > 0 {
		query += " JOIN content c ON up.content_id = c.id WHERE up.user_id = ? AND c.language_id = ? AND up.watched_at BETWEEN ? AND ?"
		s.db.Raw(query, userID, languageID, weekStart, weekEnd).Scan(&weeklyStats)
	} else {
		query += " WHERE up.user_id = ? AND up.watched_at BETWEEN ? AND ?"
		s.db.Raw(query, userID, weekStart, weekEnd).Scan(&weeklyStats)
	}

	// Get daily breakdown
	dailyBreakdown := s.getDailyBreakdown(userID, languageID, weekStart, weekEnd)

	// Get top content
	topContent := s.getMostWatchedContent(userID, weekStart, 5)

	// Get comprehension trend
	comprehensionTrend := s.getComprehensionTrend(userID, 7)

	totalHours := float64(weeklyStats.TotalMinutes) / 60
	avgSessionLength := float64(weeklyStats.TotalMinutes) / float64(weeklyStats.SessionCount)

	return &WeeklyReport{
		WeekStart:          weekStart,
		WeekEnd:            weekEnd,
		TotalMinutes:       weeklyStats.TotalMinutes,
		TotalHours:         totalHours,
		SessionCount:       weeklyStats.SessionCount,
		AvgSessionLength:   avgSessionLength,
		StreakDays:         7,               // Calculate actual streak
		GoalMet:            totalHours >= 7, // Default weekly goal
		GoalProgress:       totalHours / 7 * 100,
		TopContent:         topContent,
		DailyBreakdown:     dailyBreakdown,
		ComprehensionTrend: comprehensionTrend,
	}, nil
}

func (s *Service) GetMonthlyReport(ctx context.Context, userID string, languageID int) (*MonthlyReport, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, -1)

	// Get monthly stats
	var monthlyStats struct {
		TotalMinutes int `json:"total_minutes"`
		SessionCount int `json:"session_count"`
		ActiveDays   int `json:"active_days"`
	}

	query := `
       SELECT 
           SUM(duration_minutes) as total_minutes,
           COUNT(*) as session_count,
           COUNT(DISTINCT DATE(watched_at)) as active_days
       FROM user_progress up
   `

	if languageID > 0 {
		query += " JOIN content c ON up.content_id = c.id WHERE up.user_id = ? AND c.language_id = ? AND up.watched_at BETWEEN ? AND ?"
		s.db.Raw(query, userID, languageID, monthStart, monthEnd).Scan(&monthlyStats)
	} else {
		query += " WHERE up.user_id = ? AND up.watched_at BETWEEN ? AND ?"
		s.db.Raw(query, userID, monthStart, monthEnd).Scan(&monthlyStats)
	}

	// Get weekly breakdown
	weeklyBreakdown := s.getWeeklyBreakdown(userID, languageID, monthStart, monthEnd)

	// Get top content
	topContent := s.getMostWatchedContent(userID, monthStart, 10)

	// Get comprehension trend
	comprehensionTrend := s.getComprehensionTrend(userID, 30)

	totalHours := float64(monthlyStats.TotalMinutes) / 60
	avgSessionLength := float64(monthlyStats.TotalMinutes) / float64(monthlyStats.SessionCount)

	return &MonthlyReport{
		MonthStart:         monthStart,
		MonthEnd:           monthEnd,
		TotalMinutes:       monthlyStats.TotalMinutes,
		TotalHours:         totalHours,
		SessionCount:       monthlyStats.SessionCount,
		AvgSessionLength:   avgSessionLength,
		ActiveDays:         monthlyStats.ActiveDays,
		LongestStreak:      30,               // Calculate actual longest streak for month
		GoalMet:            totalHours >= 30, // Default monthly goal
		GoalProgress:       totalHours / 30 * 100,
		TopContent:         topContent,
		WeeklyBreakdown:    weeklyBreakdown,
		ComprehensionTrend: comprehensionTrend,
	}, nil
}

// Helper methods
func (s *Service) getDailyProgress(userID string, languageID int, date time.Time) int {
	var minutes int
	dateStr := date.Format("2006-01-02")

	query := `
       SELECT COALESCE(SUM(duration_minutes), 0)
       FROM user_progress up
   `

	if languageID > 0 {
		query += " JOIN content c ON up.content_id = c.id WHERE up.user_id = ? AND c.language_id = ? AND DATE(up.watched_at) = ?"
		s.db.Raw(query, userID, languageID, dateStr).Scan(&minutes)
	} else {
		query += " WHERE up.user_id = ? AND DATE(up.watched_at) = ?"
		s.db.Raw(query, userID, dateStr).Scan(&minutes)
	}

	return minutes
}

func (s *Service) getWeeklyProgressHours(userID string, languageID int, weekStart time.Time) float64 {
	var minutes int
	weekEnd := weekStart.AddDate(0, 0, 6)

	query := `
       SELECT COALESCE(SUM(duration_minutes), 0)
       FROM user_progress up
   `

	if languageID > 0 {
		query += " JOIN content c ON up.content_id = c.id WHERE up.user_id = ? AND c.language_id = ? AND up.watched_at BETWEEN ? AND ?"
		s.db.Raw(query, userID, languageID, weekStart, weekEnd).Scan(&minutes)
	} else {
		query += " WHERE up.user_id = ? AND up.watched_at BETWEEN ? AND ?"
		s.db.Raw(query, userID, weekStart, weekEnd).Scan(&minutes)
	}

	return float64(minutes) / 60
}

func (s *Service) getMonthlyProgressHours(userID string, languageID int, monthStart time.Time) float64 {
	var minutes int
	monthEnd := monthStart.AddDate(0, 1, -1)

	query := `
       SELECT COALESCE(SUM(duration_minutes), 0)
       FROM user_progress up
   `

	if languageID > 0 {
		query += " JOIN content c ON up.content_id = c.id WHERE up.user_id = ? AND c.language_id = ? AND up.watched_at BETWEEN ? AND ?"
		s.db.Raw(query, userID, languageID, monthStart, monthEnd).Scan(&minutes)
	} else {
		query += " WHERE up.user_id = ? AND up.watched_at BETWEEN ? AND ?"
		s.db.Raw(query, userID, monthStart, monthEnd).Scan(&minutes)
	}

	return float64(minutes) / 60
}

func (s *Service) getDailyBreakdown(userID string, languageID int, startDate, endDate time.Time) []DailyProgressSummary {
	var breakdown []DailyProgressSummary

	query := `
       SELECT 
           DATE(watched_at) as date,
           SUM(duration_minutes) as minutes,
           COUNT(*) as sessions,
           AVG(comprehension_percentage) as comprehension
       FROM user_progress up
   `

	if languageID > 0 {
		query += " JOIN content c ON up.content_id = c.id WHERE up.user_id = ? AND c.language_id = ? AND up.watched_at BETWEEN ? AND ?"
		query += " GROUP BY DATE(watched_at) ORDER BY date"
		s.db.Raw(query, userID, languageID, startDate, endDate).Scan(&breakdown)
	} else {
		query += " WHERE up.user_id = ? AND up.watched_at BETWEEN ? AND ?"
		query += " GROUP BY DATE(watched_at) ORDER BY date"
		s.db.Raw(query, userID, startDate, endDate).Scan(&breakdown)
	}

	// Add MetGoal calculation
	for i := range breakdown {
		breakdown[i].MetGoal = breakdown[i].Minutes >= 60 // Default daily goal
	}

	return breakdown
}

func (s *Service) getWeeklyBreakdown(userID string, languageID int, startDate, endDate time.Time) []WeeklyProgressSummary {
	var breakdown []WeeklyProgressSummary

	query := `
       SELECT 
           DATE_TRUNC('week', watched_at) as week_start,
           SUM(duration_minutes) as minutes,
           COUNT(*) as sessions,
           AVG(comprehension_percentage) as comprehension
       FROM user_progress up
   `

	if languageID > 0 {
		query += " JOIN content c ON up.content_id = c.id WHERE up.user_id = ? AND c.language_id = ? AND up.watched_at BETWEEN ? AND ?"
		query += " GROUP BY DATE_TRUNC('week', watched_at) ORDER BY week_start"
		s.db.Raw(query, userID, languageID, startDate, endDate).Scan(&breakdown)
	} else {
		query += " WHERE up.user_id = ? AND up.watched_at BETWEEN ? AND ?"
		query += " GROUP BY DATE_TRUNC('week', watched_at) ORDER BY week_start"
		s.db.Raw(query, userID, startDate, endDate).Scan(&breakdown)
	}

	// Add calculated fields
	for i := range breakdown {
		breakdown[i].Hours = float64(breakdown[i].Minutes) / 60
		breakdown[i].MetGoal = breakdown[i].Hours >= 7 // Default weekly goal
	}

	return breakdown
}
