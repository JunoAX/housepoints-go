package handlers

import (
	"net/http"

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
