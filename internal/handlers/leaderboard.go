package handlers

import (
	"net/http"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
)

// GetWeeklyLeaderboard returns the weekly points leaderboard
func GetWeeklyLeaderboard(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	query := `
		SELECT
			id as user_id,
			username,
			display_name,
			avatar_url,
			color_theme,
			weekly_points as points,
			weekly_points,
			total_points,
			lifetime_points_earned,
			level,
			streak_days,
			0 as chores_completed
		FROM users
		WHERE is_parent = false AND is_active = true
		ORDER BY weekly_points DESC, total_points DESC, username ASC
	`

	rows, err := db.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query leaderboard", "details": err.Error()})
		return
	}
	defer rows.Close()

	leaderboard := []models.LeaderboardEntry{}
	rank := 1

	for rows.Next() {
		var entry models.LeaderboardEntry
		err := rows.Scan(
			&entry.UserID,
			&entry.Username,
			&entry.DisplayName,
			&entry.AvatarURL,
			&entry.ColorTheme,
			&entry.Points,
			&entry.WeeklyPoints,
			&entry.TotalPoints,
			&entry.LifetimePoints,
			&entry.Level,
			&entry.StreakDays,
			&entry.ChoresCompleted,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse leaderboard data", "details": err.Error()})
			return
		}

		entry.Rank = rank
		leaderboard = append(leaderboard, entry)
		rank++
	}

	c.JSON(http.StatusOK, models.LeaderboardResponse{
		Period:      "week",
		Leaderboard: leaderboard,
		TotalUsers:  len(leaderboard),
	})
}

// GetAllTimeLeaderboard returns the all-time points leaderboard
func GetAllTimeLeaderboard(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	query := `
		SELECT
			id as user_id,
			username,
			display_name,
			avatar_url,
			color_theme,
			total_points as points,
			weekly_points,
			total_points,
			lifetime_points_earned,
			level,
			streak_days,
			0 as chores_completed
		FROM users
		WHERE is_parent = false AND is_active = true
		ORDER BY total_points DESC, weekly_points DESC, username ASC
	`

	rows, err := db.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query leaderboard", "details": err.Error()})
		return
	}
	defer rows.Close()

	leaderboard := []models.LeaderboardEntry{}
	rank := 1

	for rows.Next() {
		var entry models.LeaderboardEntry
		err := rows.Scan(
			&entry.UserID,
			&entry.Username,
			&entry.DisplayName,
			&entry.AvatarURL,
			&entry.ColorTheme,
			&entry.Points,
			&entry.WeeklyPoints,
			&entry.TotalPoints,
			&entry.LifetimePoints,
			&entry.Level,
			&entry.StreakDays,
			&entry.ChoresCompleted,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse leaderboard data", "details": err.Error()})
			return
		}

		entry.Rank = rank
		leaderboard = append(leaderboard, entry)
		rank++
	}

	c.JSON(http.StatusOK, models.LeaderboardResponse{
		Period:      "alltime",
		Leaderboard: leaderboard,
		TotalUsers:  len(leaderboard),
	})
}
