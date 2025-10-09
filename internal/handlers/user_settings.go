package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
)

// GetCurrentUserProfile returns the complete profile for the authenticated user
func GetCurrentUserProfile(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	query := `
		SELECT
			id, username, display_name, email, phone_number, color_theme, avatar_url,
			is_parent, birthdate, age, daily_goal, usually_eats_dinner,
			total_points, available_points, weekly_points, lifetime_points_earned,
			level, xp, streak_days, auto_approve_work, availability_notifications,
			login_enabled, is_active, preferences, last_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var profile models.UserProfile
	var prefsJSON []byte

	err := db.QueryRow(c.Request.Context(), query, userID).Scan(
		&profile.ID,
		&profile.Username,
		&profile.DisplayName,
		&profile.Email,
		&profile.PhoneNumber,
		&profile.ColorTheme,
		&profile.AvatarURL,
		&profile.IsParent,
		&profile.Birthdate,
		&profile.Age,
		&profile.DailyGoal,
		&profile.UsuallyEatsDinner,
		&profile.TotalPoints,
		&profile.AvailablePoints,
		&profile.WeeklyPoints,
		&profile.LifetimePointsEarned,
		&profile.Level,
		&profile.XP,
		&profile.StreakDays,
		&profile.AutoApproveWork,
		&profile.AvailabilityNotifications,
		&profile.LoginEnabled,
		&profile.IsActive,
		&prefsJSON,
		&profile.LastActive,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user", "details": err.Error()})
		}
		return
	}

	// Parse preferences JSON
	if len(prefsJSON) > 0 {
		if err := json.Unmarshal(prefsJSON, &profile.Preferences); err != nil {
			profile.Preferences = make(map[string]interface{})
		}
	} else {
		profile.Preferences = make(map[string]interface{})
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateCurrentUserProfile updates the authenticated user's profile
func UpdateCurrentUserProfile(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Build dynamic update query
	updates := make(map[string]interface{})
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.PhoneNumber != nil {
		updates["phone_number"] = *req.PhoneNumber
	}
	if req.ColorTheme != nil {
		updates["color_theme"] = *req.ColorTheme
	}
	if req.AvatarURL != nil {
		updates["avatar_url"] = *req.AvatarURL
	}
	if req.Birthdate != nil {
		updates["birthdate"] = *req.Birthdate
	}
	if req.DailyGoal != nil {
		updates["daily_goal"] = *req.DailyGoal
	}
	if req.UsuallyEatsDinner != nil {
		updates["usually_eats_dinner"] = *req.UsuallyEatsDinner
	}
	if req.AutoApproveWork != nil {
		updates["auto_approve_work"] = *req.AutoApproveWork
	}
	if req.AvailabilityNotifications != nil {
		updates["availability_notifications"] = *req.AvailabilityNotifications
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Build SQL query with proper parameter indexing
	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		query += fmt.Sprintf(", %s = $%d", field, argIndex)
		args = append(args, value)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, userID)

	_, err := db.Exec(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"updated_fields": len(updates),
	})
}

// UpdateCurrentUserPreferences updates the authenticated user's preferences
func UpdateCurrentUserPreferences(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Convert preferences to JSON
	prefsJSON, err := json.Marshal(req.Preferences)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid preferences format"})
		return
	}

	// Update preferences
	_, err = db.Exec(c.Request.Context(), `
		UPDATE users
		SET preferences = $1,
			updated_at = NOW()
		WHERE id = $2
	`, prefsJSON, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preferences updated successfully",
	})
}
