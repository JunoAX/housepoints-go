package handlers

import (
	"net/http"
	"time"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListUsers returns all users in the family
func ListUsers(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	family, _ := middleware.GetFamily(c)

	query := `
		SELECT
			id, username, display_name, age, color_theme, avatar_url,
			total_points, weekly_points, level, is_parent, is_active
		FROM users
		WHERE is_active = true
		ORDER BY is_parent DESC, display_name ASC
	`

	rows, err := db.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query users"})
		return
	}
	defer rows.Close()

	users := []models.UserListResponse{}
	for rows.Next() {
		var user models.UserListResponse
		err := rows.Scan(
			&user.ID, &user.Username, &user.DisplayName, &user.Age,
			&user.ColorTheme, &user.AvatarURL, &user.TotalPoints,
			&user.WeeklyPoints, &user.Level, &user.IsParent, &user.IsActive,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user data"})
			return
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"family_id":   family.ID,
		"family_name": family.Name,
		"users":       users,
		"count":       len(users),
	})
}

// GetUser returns details for a specific user by ID
func GetUser(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	query := `
		SELECT
			id, username, display_name, age, color_theme, avatar_url,
			total_points, weekly_points, level, xp, streak_days, last_active,
			is_parent, birthdate, availability_notifications, auto_approve_work,
			email, phone_number, school, daily_goal, usually_eats_dinner,
			available_points, lifetime_points_earned, is_active, created_at
		FROM users
		WHERE id = $1 AND is_active = true
	`

	var user models.UserDetailResponse
	err = db.QueryRow(c.Request.Context(), query, userID).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.Age,
		&user.ColorTheme, &user.AvatarURL, &user.TotalPoints, &user.WeeklyPoints,
		&user.Level, &user.XP, &user.StreakDays, &user.LastActive,
		&user.IsParent, &user.Birthdate, &user.AvailabilityNotifications,
		&user.AutoApproveWork, &user.Email, &user.PhoneNumber, &user.School,
		&user.DailyGoal, &user.UsuallyEatsDinner, &user.AvailablePoints,
		&user.LifetimePointsEarned, &user.IsActive, &user.CreatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user"})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetCurrentUser returns details for the currently authenticated user
func GetCurrentUser(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Get user ID from auth context
	userID, exists := middleware.GetAuthUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	query := `
		SELECT
			id, username, display_name, age, color_theme, avatar_url,
			total_points, weekly_points, level, xp, streak_days, last_active,
			is_parent, birthdate, availability_notifications, auto_approve_work,
			email, phone_number, school, daily_goal, usually_eats_dinner,
			available_points, lifetime_points_earned, is_active, created_at
		FROM users
		WHERE id = $1 AND is_active = true
	`

	var user models.UserDetailResponse
	err := db.QueryRow(c.Request.Context(), query, userID).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.Age,
		&user.ColorTheme, &user.AvatarURL, &user.TotalPoints, &user.WeeklyPoints,
		&user.Level, &user.XP, &user.StreakDays, &user.LastActive,
		&user.IsParent, &user.Birthdate, &user.AvailabilityNotifications,
		&user.AutoApproveWork, &user.Email, &user.PhoneNumber, &user.School,
		&user.DailyGoal, &user.UsuallyEatsDinner, &user.AvailablePoints,
		&user.LifetimePointsEarned, &user.IsActive, &user.CreatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user"})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetUserStats returns statistics for a specific user
func GetUserStats(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Get basic user info
	userQuery := `
		SELECT id, username, display_name, total_points, weekly_points,
			level, xp, streak_days, available_points, lifetime_points_earned
		FROM users
		WHERE id = $1 AND is_active = true
	`

	var stats struct {
		ID                    uuid.UUID `json:"id"`
		Username              string    `json:"username"`
		DisplayName           string    `json:"display_name"`
		TotalPoints           int       `json:"total_points"`
		WeeklyPoints          int       `json:"weekly_points"`
		Level                 int       `json:"level"`
		XP                    int       `json:"xp"`
		StreakDays            int       `json:"streak_days"`
		AvailablePoints       int       `json:"available_points"`
		LifetimePointsEarned  int       `json:"lifetime_points_earned"`
		CompletedThisWeek     int       `json:"completed_this_week"`
		PendingAssignments    int       `json:"pending_assignments"`
		TotalCompletedChores  int       `json:"total_completed_chores"`
	}

	err = db.QueryRow(c.Request.Context(), userQuery, userID).Scan(
		&stats.ID, &stats.Username, &stats.DisplayName, &stats.TotalPoints,
		&stats.WeeklyPoints, &stats.Level, &stats.XP, &stats.StreakDays,
		&stats.AvailablePoints, &stats.LifetimePointsEarned,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user", "details": err.Error()})
		}
		return
	}

	// Get completed this week count
	weekQuery := `
		SELECT COUNT(*)
		FROM assignments
		WHERE assigned_to = $1
			AND status IN ('completed', 'verified')
			AND completed_at >= date_trunc('week', CURRENT_DATE)
	`
	db.QueryRow(c.Request.Context(), weekQuery, userID).Scan(&stats.CompletedThisWeek)

	// Get pending assignments count
	pendingQuery := `
		SELECT COUNT(*)
		FROM assignments
		WHERE assigned_to = $1
			AND status IN ('pending', 'in_progress', 'pending_verification')
	`
	db.QueryRow(c.Request.Context(), pendingQuery, userID).Scan(&stats.PendingAssignments)

	// Get total completed chores
	totalQuery := `
		SELECT COUNT(*)
		FROM assignments
		WHERE assigned_to = $1
			AND status IN ('completed', 'verified')
	`
	db.QueryRow(c.Request.Context(), totalQuery, userID).Scan(&stats.TotalCompletedChores)

	c.JSON(http.StatusOK, stats)
}

// GetRedeemedRewards returns redemption history for a user
func GetRedeemedRewards(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	query := `
		SELECT
			rt.id, rt.reward_id, rt.user_id, rt.points_cost, rt.status,
			rt.redeemed_at, rt.fulfilled_at, rt.notes,
			r.name as reward_name, r.description as reward_description,
			r.icon as reward_icon, r.category as reward_category
		FROM reward_transactions rt
		JOIN rewards r ON rt.reward_id = r.id
		WHERE rt.user_id = $1
		ORDER BY rt.redeemed_at DESC
		LIMIT 100
	`

	rows, err := db.Query(c.Request.Context(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query redemptions", "details": err.Error()})
		return
	}
	defer rows.Close()

	type RedemptionResponse struct {
		ID          uuid.UUID  `json:"id"`
		RewardID    uuid.UUID  `json:"reward_id"`
		UserID      uuid.UUID  `json:"user_id"`
		PointsCost  int        `json:"points_cost"`
		Status      string     `json:"status"`
		RedeemedAt  string     `json:"redeemed_at"`
		FulfilledAt *string    `json:"fulfilled_at,omitempty"`
		Notes       *string    `json:"notes,omitempty"`
		RewardName  string     `json:"reward_name"`
		RewardDesc  *string    `json:"reward_description,omitempty"`
		RewardIcon  string     `json:"reward_icon"`
		RewardCat   string     `json:"reward_category"`
	}

	redemptions := []RedemptionResponse{}
	for rows.Next() {
		var r RedemptionResponse
		var redeemedAt, fulfilledAt *time.Time

		err := rows.Scan(
			&r.ID, &r.RewardID, &r.UserID, &r.PointsCost, &r.Status,
			&redeemedAt, &fulfilledAt, &r.Notes,
			&r.RewardName, &r.RewardDesc, &r.RewardIcon, &r.RewardCat,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse redemption", "details": err.Error()})
			return
		}

		// Format timestamps
		if redeemedAt != nil {
			str := redeemedAt.Format(time.RFC3339)
			r.RedeemedAt = str
		}
		if fulfilledAt != nil {
			str := fulfilledAt.Format(time.RFC3339)
			r.FulfilledAt = &str
		}

		redemptions = append(redemptions, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"redemptions": redemptions,
		"count":       len(redemptions),
	})
}
