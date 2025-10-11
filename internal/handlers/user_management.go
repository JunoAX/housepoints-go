package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateUser creates a new user (parent only)
func CreateUser(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can create users"})
		return
	}

	userID, _ := middleware.GetAuthUserID(c)

	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Set defaults
	if req.ColorTheme == "" {
		req.ColorTheme = "#3498db"
	}

	// Check if username already exists
	var exists bool
	err := db.QueryRow(c.Request.Context(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1))",
		req.Username,
	).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check username", "details": err.Error()})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Hash password if provided
	var passwordHash *string
	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		hashStr := string(hash)
		passwordHash = &hashStr
	}

	newUserID := uuid.New()
	query := `
		INSERT INTO users (
			id, username, display_name, is_parent, email, color_theme,
			login_enabled, password_hash, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, username, display_name, is_parent, email, color_theme, login_enabled
	`

	var user struct {
		ID           uuid.UUID
		Username     string
		DisplayName  string
		IsParent     bool
		Email        *string
		ColorTheme   string
		LoginEnabled bool
	}

	err = db.QueryRow(c.Request.Context(), query,
		newUserID, req.Username, req.DisplayName, req.IsParent, req.Email,
		req.ColorTheme, req.LoginEnabled, passwordHash, userID,
	).Scan(&user.ID, &user.Username, &user.DisplayName, &user.IsParent,
		&user.Email, &user.ColorTheme, &user.LoginEnabled)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"display_name":  user.DisplayName,
		"is_parent":     user.IsParent,
		"email":         user.Email,
		"color_theme":   user.ColorTheme,
		"login_enabled": user.LoginEnabled,
		"message":       "User created successfully",
	})
}

// UpdateUser updates an existing user (parent only)
func UpdateUser(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can update users"})
		return
	}

	currentUserID, _ := middleware.GetAuthUserID(c)

	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	var req models.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Check if user exists
	var exists bool
	err = db.QueryRow(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Build dynamic UPDATE query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Username != nil {
		updates = append(updates, fmt.Sprintf("username = $%d", argIndex))
		args = append(args, *req.Username)
		argIndex++
	}

	if req.DisplayName != nil {
		updates = append(updates, fmt.Sprintf("display_name = $%d", argIndex))
		args = append(args, *req.DisplayName)
		argIndex++
	}

	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, req.Email)
		argIndex++
	}

	if req.ColorTheme != nil {
		updates = append(updates, fmt.Sprintf("color_theme = $%d", argIndex))
		args = append(args, *req.ColorTheme)
		argIndex++
	}

	if req.IsParent != nil {
		updates = append(updates, fmt.Sprintf("is_parent = $%d", argIndex))
		args = append(args, *req.IsParent)
		argIndex++
	}

	if req.LoginEnabled != nil {
		updates = append(updates, fmt.Sprintf("login_enabled = $%d", argIndex))
		args = append(args, *req.LoginEnabled)
		argIndex++
	}

	if req.AvailabilityNotifications != nil {
		updates = append(updates, fmt.Sprintf("availability_notifications = $%d", argIndex))
		args = append(args, *req.AvailabilityNotifications)
		argIndex++
	}

	if req.AutoApproveWork != nil {
		updates = append(updates, fmt.Sprintf("auto_approve_work = $%d", argIndex))
		args = append(args, *req.AutoApproveWork)
		argIndex++
	}

	if req.PhoneNumber != nil {
		updates = append(updates, fmt.Sprintf("phone_number = $%d", argIndex))
		args = append(args, req.PhoneNumber)
		argIndex++
	}

	if req.School != nil {
		updates = append(updates, fmt.Sprintf("school = $%d", argIndex))
		args = append(args, req.School)
		argIndex++
	}

	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIndex))
		args = append(args, req.Notes)
		argIndex++
	}

	if req.DailyGoal != nil {
		updates = append(updates, fmt.Sprintf("daily_goal = $%d", argIndex))
		args = append(args, *req.DailyGoal)
		argIndex++
	}

	if req.UsuallyEatsDinner != nil {
		updates = append(updates, fmt.Sprintf("usually_eats_dinner = $%d", argIndex))
		args = append(args, *req.UsuallyEatsDinner)
		argIndex++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, "updated_at = NOW()")
	updates = append(updates, fmt.Sprintf("updated_by = $%d", argIndex))
	args = append(args, currentUserID)
	argIndex++

	args = append(args, userID)

	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE id = $%d
		RETURNING id, username, display_name, is_parent, email, color_theme,
			login_enabled, is_active
	`, strings.Join(updates, ", "), argIndex)

	var user struct {
		ID           uuid.UUID
		Username     string
		DisplayName  string
		IsParent     bool
		Email        *string
		ColorTheme   string
		LoginEnabled bool
		IsActive     bool
	}

	err = db.QueryRow(c.Request.Context(), query, args...).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.IsParent,
		&user.Email, &user.ColorTheme, &user.LoginEnabled, &user.IsActive,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"display_name":  user.DisplayName,
			"is_parent":     user.IsParent,
			"email":         user.Email,
			"color_theme":   user.ColorTheme,
			"login_enabled": user.LoginEnabled,
			"is_active":     user.IsActive,
		},
		"message": "User updated successfully",
	})
}

// DeleteUser soft-deletes a user (parent only)
func DeleteUser(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can delete users"})
		return
	}

	currentUserID, _ := middleware.GetAuthUserID(c)

	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Prevent self-deletion
	if userID == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	// Soft delete by setting is_active to false
	result, err := db.Exec(c.Request.Context(), `
		UPDATE users
		SET is_active = false, login_enabled = false, updated_at = NOW(), updated_by = $1
		WHERE id = $2
	`, currentUserID, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user", "details": err.Error()})
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}
