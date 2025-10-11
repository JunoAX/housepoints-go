package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetWeeklySummary returns weekly summary statistics
func GetWeeklySummary(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Get query parameters
	weekStart := c.Query("week_start")
	weekEnd := c.Query("week_end")

	// Default to current week if not specified
	if weekStart == "" {
		today := time.Now()
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday is 0, convert to 7
		}
		startOfWeek := today.AddDate(0, 0, -(weekday - 1))
		weekStart = startOfWeek.Format("2006-01-02")
	}
	if weekEnd == "" {
		start, _ := time.Parse("2006-01-02", weekStart)
		end := start.AddDate(0, 0, 6)
		weekEnd = end.Format("2006-01-02")
	}

	// Get summary statistics
	summaryQuery := `
		SELECT
			COUNT(DISTINCT a.id) as total_assignments,
			COUNT(DISTINCT CASE WHEN a.status = 'completed' THEN a.id END) as completed,
			COUNT(DISTINCT CASE WHEN a.status = 'verified' THEN a.id END) as verified,
			COUNT(DISTINCT CASE WHEN a.status = 'pending' AND a.due_date < CURRENT_DATE THEN a.id END) as overdue,
			COALESCE(SUM(CASE WHEN a.status = 'verified' THEN c.base_points ELSE 0 END), 0) as total_points,
			COUNT(DISTINCT a.assigned_to) as active_children
		FROM assignments a
		LEFT JOIN chores c ON a.chore_id = c.id
		WHERE DATE(a.due_date) BETWEEN $1 AND $2
	`

	var summary models.WeeklySummary
	err := db.QueryRow(c.Request.Context(), summaryQuery, weekStart, weekEnd).Scan(
		&summary.TotalAssignments,
		&summary.Completed,
		&summary.Verified,
		&summary.Overdue,
		&summary.TotalPoints,
		&summary.ActiveChildren,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query summary", "details": err.Error()})
		return
	}

	// Calculate completion rate
	if summary.TotalAssignments > 0 {
		summary.CompletionRate = float64(summary.Completed) / float64(summary.TotalAssignments) * 100
		summary.CompletionRate = float64(int(summary.CompletionRate*10)) / 10 // Round to 1 decimal
	}

	// Get child performance
	childQuery := `
		SELECT
			u.id,
			u.username,
			u.display_name,
			u.color_theme,
			COUNT(a.id) as total_assignments,
			COUNT(CASE WHEN a.status = 'completed' OR a.status = 'verified' THEN 1 END) as completed,
			COUNT(CASE WHEN a.status = 'verified' THEN 1 END) as verified,
			COALESCE(SUM(CASE WHEN a.status = 'verified' THEN c.base_points ELSE 0 END), 0) as points_earned
		FROM users u
		LEFT JOIN assignments a ON u.id = a.assigned_to
			AND DATE(a.due_date) BETWEEN $1 AND $2
		LEFT JOIN chores c ON a.chore_id = c.id
		WHERE u.is_parent = false AND u.is_active = true
		GROUP BY u.id, u.username, u.display_name, u.color_theme
		ORDER BY points_earned DESC
	`

	rows, err := db.Query(c.Request.Context(), childQuery, weekStart, weekEnd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query children", "details": err.Error()})
		return
	}
	defer rows.Close()

	children := []models.ChildWeeklyPerformance{}
	for rows.Next() {
		var child models.ChildWeeklyPerformance
		err := rows.Scan(
			&child.ID,
			&child.Username,
			&child.DisplayName,
			&child.ColorTheme,
			&child.TotalAssignments,
			&child.Completed,
			&child.Verified,
			&child.PointsEarned,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse child", "details": err.Error()})
			return
		}

		// Calculate completion rate
		if child.TotalAssignments > 0 {
			child.CompletionRate = float64(child.Completed) / float64(child.TotalAssignments) * 100
			child.CompletionRate = float64(int(child.CompletionRate*10)) / 10
		}

		children = append(children, child)
	}

	c.JSON(http.StatusOK, models.WeeklySummaryResponse{
		WeekStart: weekStart,
		WeekEnd:   weekEnd,
		Summary:   summary,
		Children:  children,
	})
}

// GetChildPerformance returns detailed performance metrics for a specific child
func GetChildPerformance(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	childID := c.Param("child_id")
	if _, err := uuid.Parse(childID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid child ID format"})
		return
	}

	// Get days parameter (default 30, max 365)
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if parsedDays, err := strconv.Atoi(daysParam); err == nil {
			if parsedDays >= 1 && parsedDays <= 365 {
				days = parsedDays
			}
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Overall stats
	statsQuery := `
		SELECT
			COUNT(a.id) as total_assignments,
			COUNT(CASE WHEN a.status = 'completed' OR a.status = 'verified' THEN 1 END) as completed,
			COUNT(CASE WHEN a.status = 'verified' THEN 1 END) as verified,
			COUNT(CASE WHEN a.status = 'pending' AND a.due_date < CURRENT_DATE THEN 1 END) as overdue,
			COALESCE(SUM(CASE WHEN a.status = 'verified' THEN c.base_points ELSE 0 END), 0) as total_points
		FROM assignments a
		LEFT JOIN chores c ON a.chore_id = c.id
		WHERE a.assigned_to = $1
		  AND DATE(a.due_date) BETWEEN $2 AND $3
	`

	var stats models.PerformanceStats
	err := db.QueryRow(c.Request.Context(), statsQuery, childID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).Scan(
		&stats.TotalAssignments,
		&stats.Completed,
		&stats.Verified,
		&stats.Overdue,
		&stats.TotalPoints,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query stats", "details": err.Error()})
		return
	}

	// Daily breakdown
	dailyQuery := `
		SELECT
			DATE(a.due_date) as date,
			COUNT(a.id) as assignments,
			COUNT(CASE WHEN a.status = 'completed' OR a.status = 'verified' THEN 1 END) as completed,
			COALESCE(SUM(CASE WHEN a.status = 'verified' THEN c.base_points ELSE 0 END), 0) as points
		FROM assignments a
		LEFT JOIN chores c ON a.chore_id = c.id
		WHERE a.assigned_to = $1
		  AND DATE(a.due_date) BETWEEN $2 AND $3
		GROUP BY DATE(a.due_date)
		ORDER BY DATE(a.due_date)
	`

	rows, err := db.Query(c.Request.Context(), dailyQuery, childID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query daily breakdown", "details": err.Error()})
		return
	}
	defer rows.Close()

	dailyBreakdown := []models.DailyStats{}
	for rows.Next() {
		var day models.DailyStats
		var date time.Time
		err := rows.Scan(&date, &day.Assignments, &day.Completed, &day.Points)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse daily data", "details": err.Error()})
			return
		}
		day.Date = date.Format("2006-01-02")
		dailyBreakdown = append(dailyBreakdown, day)
	}

	c.JSON(http.StatusOK, models.ChildPerformanceResponse{
		ChildID: childID,
		Period: models.DateRange{
			Start: startDate.Format("2006-01-02"),
			End:   endDate.Format("2006-01-02"),
		},
		Stats:          stats,
		DailyBreakdown: dailyBreakdown,
	})
}

// GetCategoryBreakdown returns breakdown by chore category
func GetCategoryBreakdown(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Get days parameter (default 30, max 365)
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if parsedDays, err := strconv.Atoi(daysParam); err == nil {
			if parsedDays >= 1 && parsedDays <= 365 {
				days = parsedDays
			}
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	query := `
		SELECT
			c.category,
			COUNT(a.id) as total_assignments,
			COUNT(CASE WHEN a.status = 'completed' OR a.status = 'verified' THEN 1 END) as completed,
			COALESCE(SUM(CASE WHEN a.status = 'verified' THEN c.base_points ELSE 0 END), 0) as points_earned
		FROM chores c
		LEFT JOIN assignments a ON c.id = a.chore_id
			AND DATE(a.due_date) BETWEEN $1 AND $2
		WHERE c.active = true
		GROUP BY c.category
		ORDER BY points_earned DESC
	`

	rows, err := db.Query(c.Request.Context(), query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query categories", "details": err.Error()})
		return
	}
	defer rows.Close()

	categories := []models.CategoryStats{}
	for rows.Next() {
		var cat models.CategoryStats
		err := rows.Scan(&cat.Category, &cat.TotalAssignments, &cat.Completed, &cat.PointsEarned)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse category", "details": err.Error()})
			return
		}

		// Calculate completion rate
		if cat.TotalAssignments > 0 {
			cat.CompletionRate = float64(cat.Completed) / float64(cat.TotalAssignments) * 100
			cat.CompletionRate = float64(int(cat.CompletionRate*10)) / 10
		}

		categories = append(categories, cat)
	}

	c.JSON(http.StatusOK, models.CategoryBreakdownResponse{
		Period: models.DateRange{
			Start: startDate.Format("2006-01-02"),
			End:   endDate.Format("2006-01-02"),
		},
		Categories: categories,
	})
}

// GetPerformanceTrends returns performance trend data for charts
func GetPerformanceTrends(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Get range parameter (default 30, options: 7, 30, 90)
	rangeParam := c.DefaultQuery("range", "30")
	days := 30
	switch rangeParam {
	case "7":
		days = 7
	case "90":
		days = 90
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get daily trends
	trendsQuery := `
		SELECT
			DATE(a.due_date) as date,
			COUNT(CASE WHEN a.status IN ('completed', 'verified') THEN 1 END) as completed,
			COALESCE(SUM(CASE WHEN a.status = 'verified' THEN c.base_points ELSE 0 END), 0) as points,
			COUNT(CASE WHEN a.status = 'verified' THEN 1 END) as verified
		FROM assignments a
		LEFT JOIN chores c ON a.chore_id = c.id
		WHERE DATE(a.due_date) BETWEEN $1 AND $2
		GROUP BY DATE(a.due_date)
		ORDER BY DATE(a.due_date)
	`

	rows, err := db.Query(c.Request.Context(), trendsQuery, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query trends", "details": err.Error()})
		return
	}
	defer rows.Close()

	trends := []models.TrendDataPoint{}
	for rows.Next() {
		var trend models.TrendDataPoint
		var date time.Time
		err := rows.Scan(&date, &trend.Completed, &trend.Points, &trend.Verified)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse trend", "details": err.Error()})
			return
		}
		trend.Date = date.Format("2006-01-02")
		trends = append(trends, trend)
	}

	// Get child performance summary
	childQuery := `
		SELECT
			u.id,
			u.display_name,
			COUNT(a.id) as total_assignments,
			COUNT(CASE WHEN a.status IN ('completed', 'verified') THEN 1 END) as completed,
			COALESCE(SUM(CASE WHEN a.status = 'verified' THEN c.base_points ELSE 0 END), 0) as points_earned,
			(
				SELECT COUNT(*)
				FROM assignments a2
				WHERE a2.assigned_to = u.id
				  AND a2.status IN ('completed', 'verified')
				  AND DATE(a2.due_date) BETWEEN $1 AND $2
				  AND DATE(a2.due_date) >= DATE(a2.created_at)
			) as streak
		FROM users u
		LEFT JOIN assignments a ON u.id = a.assigned_to
			AND DATE(a.due_date) BETWEEN $1 AND $2
		LEFT JOIN chores c ON a.chore_id = c.id
		WHERE u.is_parent = false AND u.is_active = true
		GROUP BY u.id, u.display_name
		HAVING COUNT(a.id) > 0
		ORDER BY completed DESC
	`

	rows, err = db.Query(c.Request.Context(), childQuery, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query child performance", "details": err.Error()})
		return
	}
	defer rows.Close()

	childPerformance := []models.ChildTrendPerformance{}
	for rows.Next() {
		var child models.ChildTrendPerformance
		var totalAssignments, completed, pointsEarned, streak int
		var id string
		err := rows.Scan(&id, &child.Name, &totalAssignments, &completed, &pointsEarned, &streak)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse child performance", "details": err.Error()})
			return
		}

		child.ID = id

		// Calculate completion rate
		completionRate := 0
		if totalAssignments > 0 {
			completionRate = int(float64(completed) / float64(totalAssignments) * 100)
		}
		child.Completion = fmt.Sprintf("%d%%", completionRate)

		// Calculate average points per day
		if days > 0 {
			child.AvgPointsPerDay = pointsEarned / days
		}

		child.Streak = fmt.Sprintf("%d days", streak)

		// Determine trend
		if completionRate >= 90 {
			child.Trend = "up"
		} else if completionRate < 70 {
			child.Trend = "down"
		} else {
			child.Trend = "neutral"
		}

		childPerformance = append(childPerformance, child)
	}

	c.JSON(http.StatusOK, models.PerformanceTrendsResponse{
		Trends:           trends,
		ChildPerformance: childPerformance,
	})
}
