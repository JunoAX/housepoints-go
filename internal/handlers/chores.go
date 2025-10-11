package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListChores returns all active chores for the family
func ListChores(c *gin.Context) {
	// Get family database connection from context
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	family, _ := middleware.GetFamily(c)

	// Query chores from family database
	query := `
		SELECT
			id, name, category, base_points, estimated_minutes,
			difficulty, icon, active, assignment_type
		FROM chores
		WHERE is_active = true
		ORDER BY category, name
	`

	rows, err := db.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query chores",
		})
		return
	}
	defer rows.Close()

	chores := []models.ChoreListResponse{}
	for rows.Next() {
		var chore models.ChoreListResponse
		err := rows.Scan(
			&chore.ID,
			&chore.Name,
			&chore.Category,
			&chore.BasePoints,
			&chore.EstimatedMinutes,
			&chore.Difficulty,
			&chore.Icon,
			&chore.Active,
			&chore.AssignmentType,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to parse chore data",
			})
			return
		}
		chores = append(chores, chore)
	}

	c.JSON(http.StatusOK, gin.H{
		"family_id":   family.ID,
		"family_name": family.Name,
		"chores":      chores,
		"count":       len(chores),
	})
}

// CreateChore creates a new chore (requires parent permissions)
func CreateChore(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can create chores"})
		return
	}

	var req models.ChoreCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Set defaults
	if req.Category == "" {
		req.Category = "other"
	}
	if req.Difficulty == "" {
		req.Difficulty = "easy"
	}
	if req.BasePoints == 0 {
		req.BasePoints = 10
	}

	choreID := uuid.New()

	query := `
		INSERT INTO chores (
			id, name, description, instructions, category, base_points,
			bonus_eligible, penalty_points, estimated_minutes, difficulty,
			frequency, active, tags, rotation_eligible, requires_photo,
			requires_verification, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW()
		)
		RETURNING id, name, description, instructions, category, base_points,
			bonus_eligible, penalty_points, estimated_minutes, difficulty,
			frequency, active, created_at, updated_at, tags, rotation_eligible,
			requires_photo, requires_verification
	`

	var chore models.Chore
	err := db.QueryRow(c.Request.Context(), query,
		choreID, req.Name, req.Description, req.Instructions, req.Category, req.BasePoints,
		req.BonusEligible, req.PenaltyPoints, req.EstimatedMinutes, req.Difficulty,
		req.Frequency, true, req.Tags, req.RotationEligible, req.RequiresPhoto,
		req.RequiresVerification,
	).Scan(
		&chore.ID, &chore.Name, &chore.Description, &chore.Instructions, &chore.Category,
		&chore.BasePoints, &chore.BonusEligible, &chore.PenaltyPoints, &chore.EstimatedMinutes,
		&chore.Difficulty, &chore.Frequency, &chore.Active, &chore.CreatedAt, &chore.UpdatedAt,
		&chore.Tags, &chore.RotationEligible, &chore.RequiresPhoto, &chore.RequiresVerification,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chore", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, chore)
}

// UpdateChore updates an existing chore (requires parent permissions)
func UpdateChore(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can update chores"})
		return
	}

	choreID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chore ID"})
		return
	}

	var req models.ChoreUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Build dynamic UPDATE query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}
	if req.Instructions != nil {
		updates = append(updates, fmt.Sprintf("instructions = $%d", argIndex))
		args = append(args, *req.Instructions)
		argIndex++
	}
	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *req.Category)
		argIndex++
	}
	if req.BasePoints != nil {
		updates = append(updates, fmt.Sprintf("base_points = $%d", argIndex))
		args = append(args, *req.BasePoints)
		argIndex++
	}
	if req.BonusEligible != nil {
		updates = append(updates, fmt.Sprintf("bonus_eligible = $%d", argIndex))
		args = append(args, *req.BonusEligible)
		argIndex++
	}
	if req.PenaltyPoints != nil {
		updates = append(updates, fmt.Sprintf("penalty_points = $%d", argIndex))
		args = append(args, *req.PenaltyPoints)
		argIndex++
	}
	if req.EstimatedMinutes != nil {
		updates = append(updates, fmt.Sprintf("estimated_minutes = $%d", argIndex))
		args = append(args, *req.EstimatedMinutes)
		argIndex++
	}
	if req.Difficulty != nil {
		updates = append(updates, fmt.Sprintf("difficulty = $%d", argIndex))
		args = append(args, *req.Difficulty)
		argIndex++
	}
	if req.Frequency != nil {
		updates = append(updates, fmt.Sprintf("frequency = $%d", argIndex))
		args = append(args, *req.Frequency)
		argIndex++
	}
	if req.Active != nil {
		updates = append(updates, fmt.Sprintf("active = $%d", argIndex))
		args = append(args, *req.Active)
		argIndex++
	}
	if req.Tags != nil {
		updates = append(updates, fmt.Sprintf("tags = $%d", argIndex))
		args = append(args, req.Tags)
		argIndex++
	}
	if req.RotationEligible != nil {
		updates = append(updates, fmt.Sprintf("rotation_eligible = $%d", argIndex))
		args = append(args, *req.RotationEligible)
		argIndex++
	}
	if req.RequiresPhoto != nil {
		updates = append(updates, fmt.Sprintf("requires_photo = $%d", argIndex))
		args = append(args, *req.RequiresPhoto)
		argIndex++
	}
	if req.RequiresVerification != nil {
		updates = append(updates, fmt.Sprintf("requires_verification = $%d", argIndex))
		args = append(args, *req.RequiresVerification)
		argIndex++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Add updated_at and chore ID
	updates = append(updates, "updated_at = NOW()")
	args = append(args, choreID)

	query := fmt.Sprintf(`
		UPDATE chores
		SET %s
		WHERE id = $%d
		RETURNING id, name, description, instructions, category, base_points,
			bonus_eligible, penalty_points, estimated_minutes, difficulty,
			frequency, active, created_at, updated_at, tags, rotation_eligible,
			requires_photo, requires_verification
	`, strings.Join(updates, ", "), argIndex)

	var chore models.Chore
	err = db.QueryRow(c.Request.Context(), query, args...).Scan(
		&chore.ID, &chore.Name, &chore.Description, &chore.Instructions, &chore.Category,
		&chore.BasePoints, &chore.BonusEligible, &chore.PenaltyPoints, &chore.EstimatedMinutes,
		&chore.Difficulty, &chore.Frequency, &chore.Active, &chore.CreatedAt, &chore.UpdatedAt,
		&chore.Tags, &chore.RotationEligible, &chore.RequiresPhoto, &chore.RequiresVerification,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chore", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chore)
}

// DeleteChore soft-deletes a chore by setting active=false (requires parent permissions)
func DeleteChore(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can delete chores"})
		return
	}

	choreID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chore ID"})
		return
	}

	// Soft delete by setting active = false
	query := `
		UPDATE chores
		SET active = false, updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`

	var deletedID uuid.UUID
	err = db.QueryRow(c.Request.Context(), query, choreID).Scan(&deletedID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chore not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Chore deleted successfully",
		"id":      deletedID,
	})
}

// GetChore returns details for a specific chore by ID
func GetChore(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	choreID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chore ID"})
		return
	}

	query := `
		SELECT
			id, name, description, instructions, category, base_points,
			bonus_eligible, penalty_points, estimated_minutes, difficulty,
			frequency, active, tags, rotation_eligible, requires_photo,
			requires_verification, icon, assignment_type, created_at, updated_at
		FROM chores
		WHERE id = $1 AND is_active = true
	`

	var chore models.Chore
	err = db.QueryRow(c.Request.Context(), query, choreID).Scan(
		&chore.ID, &chore.Name, &chore.Description, &chore.Instructions, &chore.Category,
		&chore.BasePoints, &chore.BonusEligible, &chore.PenaltyPoints, &chore.EstimatedMinutes,
		&chore.Difficulty, &chore.Frequency, &chore.Active, &chore.Tags, &chore.RotationEligible,
		&chore.RequiresPhoto, &chore.RequiresVerification, &chore.Icon, &chore.AssignmentType,
		&chore.CreatedAt, &chore.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chore not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query chore", "details": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, chore)
}
