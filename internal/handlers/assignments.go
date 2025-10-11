package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListAssignments returns assignments with optional filters
func ListAssignments(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Get query parameters for filtering
	userIDParam := c.Query("user_id")           // Filter by assigned user
	status := c.Query("status")                  // Filter by status
	startDate := c.Query("start_date")           // Filter by due date range
	endDate := c.Query("end_date")               // Filter by due date range

	// Build query
	query := `
		SELECT
			a.id, a.chore_id, a.assigned_to, a.assigned_by,
			a.status, a.points_offered, COALESCE(a.points_earned, 0) as points_earned,
			a.due_date, a.created_at, a.updated_at, a.completed_at, a.verified_at,
			a.completion_notes, a.verification_notes,
			c.name as chore_name, c.description as chore_description,
			c.category, c.difficulty, c.estimated_minutes, c.base_points,
			c.requires_verification, c.requires_photo, c.icon,
			u.display_name as assigned_user_name, u.username as assigned_username,
			u.color_theme as assigned_user_color
		FROM assignments a
		JOIN chores c ON a.chore_id = c.id
		LEFT JOIN users u ON a.assigned_to = u.id
		WHERE 1=1
	`

	params := []interface{}{}
	paramCount := 0

	// Apply filters
	if userIDParam != "" {
		userID, err := uuid.Parse(userIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
			return
		}
		paramCount++
		query += fmt.Sprintf(" AND a.assigned_to = $%d", paramCount)
		params = append(params, userID)
	}

	if status != "" {
		paramCount++
		query += fmt.Sprintf(" AND a.status = $%d", paramCount)
		params = append(params, status)
	} else {
		// Default: show active assignments
		query += ` AND a.status IN ('pending', 'in_progress', 'pending_verification', 'open')`
	}

	if startDate != "" {
		paramCount++
		query += fmt.Sprintf(" AND a.due_date >= $%d", paramCount)
		params = append(params, startDate)
	}

	if endDate != "" {
		paramCount++
		query += fmt.Sprintf(" AND a.due_date <= $%d", paramCount)
		params = append(params, endDate)
	}

	query += ` ORDER BY a.due_date ASC NULLS LAST, a.created_at DESC LIMIT 100`

	rows, err := db.Query(c.Request.Context(), query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query assignments", "details": err.Error()})
		return
	}
	defer rows.Close()

	assignments := []models.AssignmentListResponse{}
	for rows.Next() {
		var (
			id, choreID                            uuid.UUID
			assignedTo, assignedBy                 *uuid.UUID
			status                                  string
			pointsOffered, pointsEarned            int
			dueDate, completedAt, verifiedAt       *time.Time
			createdAt                              time.Time
			updatedAt                              *time.Time
			completionNotes, verificationNotes     *string
			choreName                              string
			choreDescription                       *string
			category, difficulty, icon             string
			estimatedMinutes, basePoints           *int
			requiresVerification, requiresPhoto    bool
			assignedUserName, assignedUsername     *string
			assignedUserColor                      *string
		)

		err := rows.Scan(
			&id, &choreID, &assignedTo, &assignedBy,
			&status, &pointsOffered, &pointsEarned,
			&dueDate, &createdAt, &updatedAt, &completedAt, &verifiedAt,
			&completionNotes, &verificationNotes,
			&choreName, &choreDescription, &category, &difficulty,
			&estimatedMinutes, &basePoints, &requiresVerification, &requiresPhoto, &icon,
			&assignedUserName, &assignedUsername, &assignedUserColor,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse assignment data", "details": err.Error()})
			return
		}

		// Build chore info
		chore := models.AssignmentChoreInfo{
			ID:                   choreID,
			Name:                 choreName,
			Description:          choreDescription,
			Category:             category,
			Difficulty:           difficulty,
			EstimatedMinutes:     estimatedMinutes,
			RequiresVerification: requiresVerification,
			RequiresPhoto:        requiresPhoto,
			Icon:                 icon,
			BasePoints:           *basePoints,
		}

		// Build assigned user info (if assigned)
		var assignedUser *models.AssignmentUserInfo
		if assignedTo != nil && assignedUserName != nil {
			assignedUser = &models.AssignmentUserInfo{
				ID:          *assignedTo,
				Username:    *assignedUsername,
				DisplayName: *assignedUserName,
				ColorTheme:  *assignedUserColor,
			}
		}

		// Determine if bonus assignment
		isBonus := assignedTo == nil || status == "open"

		// Format dates as ISO strings
		var dueDateStr, completedAtStr, verifiedAtStr, updatedAtStr *string
		if dueDate != nil {
			str := dueDate.Format("2006-01-02")
			dueDateStr = &str
		}
		if completedAt != nil {
			str := completedAt.Format(time.RFC3339)
			completedAtStr = &str
		}
		if verifiedAt != nil {
			str := verifiedAt.Format(time.RFC3339)
			verifiedAtStr = &str
		}
		if updatedAt != nil {
			str := updatedAt.Format(time.RFC3339)
			updatedAtStr = &str
		}

		assignment := models.AssignmentListResponse{
			ID:                id,
			ChoreID:           choreID,
			AssignedTo:        assignedTo,
			AssignedBy:        assignedBy,
			Status:            status,
			PointsOffered:     pointsOffered,
			PointsEarned:      pointsEarned,
			DueDate:           dueDateStr,
			CreatedAt:         createdAt.Format(time.RFC3339),
			UpdatedAt:         updatedAtStr,
			CompletedAt:       completedAtStr,
			VerifiedAt:        verifiedAtStr,
			CompletionNotes:   completionNotes,
			VerificationNotes: verificationNotes,
			IsBonus:           isBonus,
			Chore:             chore,
			AssignedUser:      assignedUser,
		}

		assignments = append(assignments, assignment)
	}

	c.JSON(http.StatusOK, gin.H{
		"assignments": assignments,
		"count":       len(assignments),
	})
}

// GetAssignment returns details for a specific assignment by ID
func GetAssignment(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	assignmentIDParam := c.Param("id")
	assignmentID, err := uuid.Parse(assignmentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID format"})
		return
	}

	query := `
		SELECT
			a.id, a.chore_id, a.assigned_to, a.assigned_by,
			a.status, a.points_offered, COALESCE(a.points_earned, 0) as points_earned,
			a.due_date, a.created_at, a.updated_at, a.completed_at, a.verified_at,
			a.completion_notes, a.verification_notes,
			c.name as chore_name, c.description as chore_description,
			c.category, c.difficulty, c.estimated_minutes, c.base_points,
			c.requires_verification, c.requires_photo, c.icon,
			u.display_name as assigned_user_name, u.username as assigned_username,
			u.color_theme as assigned_user_color,
			ub.display_name as assigned_by_name, ub.username as assigned_by_username,
			ub.color_theme as assigned_by_color
		FROM assignments a
		JOIN chores c ON a.chore_id = c.id
		LEFT JOIN users u ON a.assigned_to = u.id
		LEFT JOIN users ub ON a.assigned_by = ub.id
		WHERE a.id = $1
	`

	var (
		id, choreID                                uuid.UUID
		assignedTo, assignedBy                     *uuid.UUID
		status                                      string
		pointsOffered, pointsEarned                int
		dueDate, completedAt, verifiedAt           *time.Time
		createdAt                                  time.Time
		updatedAt                                  *time.Time
		completionNotes, verificationNotes         *string
		choreName                                  string
		choreDescription                           *string
		category, difficulty, icon                 string
		estimatedMinutes, basePoints               *int
		requiresVerification, requiresPhoto        bool
		assignedUserName, assignedUsername         *string
		assignedUserColor                          *string
		assignedByName, assignedByUsername         *string
		assignedByColor                            *string
	)

	err = db.QueryRow(c.Request.Context(), query, assignmentID).Scan(
		&id, &choreID, &assignedTo, &assignedBy,
		&status, &pointsOffered, &pointsEarned,
		&dueDate, &createdAt, &updatedAt, &completedAt, &verifiedAt,
		&completionNotes, &verificationNotes,
		&choreName, &choreDescription, &category, &difficulty,
		&estimatedMinutes, &basePoints, &requiresVerification, &requiresPhoto, &icon,
		&assignedUserName, &assignedUsername, &assignedUserColor,
		&assignedByName, &assignedByUsername, &assignedByColor,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query assignment", "details": err.Error()})
		}
		return
	}

	// Build chore info
	chore := models.AssignmentChoreInfo{
		ID:                   choreID,
		Name:                 choreName,
		Description:          choreDescription,
		Category:             category,
		Difficulty:           difficulty,
		EstimatedMinutes:     estimatedMinutes,
		RequiresVerification: requiresVerification,
		RequiresPhoto:        requiresPhoto,
		Icon:                 icon,
		BasePoints:           *basePoints,
	}

	// Build assigned user info
	var assignedUser *models.AssignmentUserInfo
	if assignedTo != nil && assignedUserName != nil {
		assignedUser = &models.AssignmentUserInfo{
			ID:          *assignedTo,
			Username:    *assignedUsername,
			DisplayName: *assignedUserName,
			ColorTheme:  *assignedUserColor,
		}
	}

	// Build assigned by user info
	var assignedByUser *models.AssignmentUserInfo
	if assignedBy != nil && assignedByName != nil {
		assignedByUser = &models.AssignmentUserInfo{
			ID:          *assignedBy,
			Username:    *assignedByUsername,
			DisplayName: *assignedByName,
			ColorTheme:  *assignedByColor,
		}
	}

	// Determine if bonus
	isBonus := assignedTo == nil || status == "open"

	// Format dates
	var dueDateStr, completedAtStr, verifiedAtStr, updatedAtStr *string
	if dueDate != nil {
		str := dueDate.Format("2006-01-02")
		dueDateStr = &str
	}
	if completedAt != nil {
		str := completedAt.Format(time.RFC3339)
		completedAtStr = &str
	}
	if verifiedAt != nil {
		str := verifiedAt.Format(time.RFC3339)
		verifiedAtStr = &str
	}
	if updatedAt != nil {
		str := updatedAt.Format(time.RFC3339)
		updatedAtStr = &str
	}

	assignment := models.AssignmentDetailResponse{
		ID:                id,
		ChoreID:           choreID,
		AssignedTo:        assignedTo,
		AssignedBy:        assignedBy,
		Status:            status,
		PointsOffered:     pointsOffered,
		PointsEarned:      pointsEarned,
		DueDate:           dueDateStr,
		CreatedAt:         createdAt.Format(time.RFC3339),
		UpdatedAt:         updatedAtStr,
		CompletedAt:       completedAtStr,
		VerifiedAt:        verifiedAtStr,
		CompletionNotes:   completionNotes,
		VerificationNotes: verificationNotes,
		IsBonus:           isBonus,
		Chore:             chore,
		AssignedUser:      assignedUser,
		AssignedByUser:    assignedByUser,
	}

	c.JSON(http.StatusOK, assignment)
}

// GetMyAssignments returns assignments for the current user (both assigned and open/bonus tasks)
func GetMyAssignments(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Get current user ID from auth
	userID, exists := middleware.GetAuthUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get optional status filter
	status := c.Query("status")

	// Build query - returns user's assignments + open assignments (bonus tasks)
	query := `
		SELECT
			a.id, a.chore_id, a.assigned_to, a.assigned_by,
			a.status, a.points_offered, COALESCE(a.points_earned, 0) as points_earned,
			a.due_date, a.created_at, a.updated_at, a.completed_at, a.verified_at,
			a.completion_notes, a.verification_notes,
			c.name as chore_name, c.description as chore_description,
			c.category, c.difficulty, c.estimated_minutes, c.base_points,
			c.requires_verification, c.requires_photo, c.icon,
			u.display_name as assigned_user_name, u.username as assigned_username,
			u.color_theme as assigned_user_color
		FROM assignments a
		JOIN chores c ON a.chore_id = c.id
		LEFT JOIN users u ON a.assigned_to = u.id
		WHERE (a.assigned_to = $1 OR a.assigned_to IS NULL OR a.status = 'open')
	`

	params := []interface{}{userID}
	paramCount := 1

	// Apply status filter if provided
	if status != "" {
		paramCount++
		query += fmt.Sprintf(" AND a.status = $%d", paramCount)
		params = append(params, status)
	} else {
		// Default: show active assignments
		query += ` AND a.status IN ('pending', 'in_progress', 'pending_verification', 'open')`
	}

	query += ` ORDER BY a.due_date ASC NULLS LAST, a.created_at DESC LIMIT 100`

	rows, err := db.Query(c.Request.Context(), query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query assignments", "details": err.Error()})
		return
	}
	defer rows.Close()

	assignments := []models.AssignmentListResponse{}
	for rows.Next() {
		var (
			id, choreID                            uuid.UUID
			assignedTo, assignedBy                 *uuid.UUID
			status                                  string
			pointsOffered, pointsEarned            int
			dueDate, completedAt, verifiedAt       *time.Time
			createdAt                              time.Time
			updatedAt                              *time.Time
			completionNotes, verificationNotes     *string
			choreName                              string
			choreDescription                       *string
			category, difficulty, icon             string
			estimatedMinutes, basePoints           *int
			requiresVerification, requiresPhoto    bool
			assignedUserName, assignedUsername     *string
			assignedUserColor                      *string
		)

		err := rows.Scan(
			&id, &choreID, &assignedTo, &assignedBy,
			&status, &pointsOffered, &pointsEarned,
			&dueDate, &createdAt, &updatedAt, &completedAt, &verifiedAt,
			&completionNotes, &verificationNotes,
			&choreName, &choreDescription, &category, &difficulty,
			&estimatedMinutes, &basePoints, &requiresVerification, &requiresPhoto, &icon,
			&assignedUserName, &assignedUsername, &assignedUserColor,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse assignment data", "details": err.Error()})
			return
		}

		// Build chore info
		chore := models.AssignmentChoreInfo{
			ID:                   choreID,
			Name:                 choreName,
			Description:          choreDescription,
			Category:             category,
			Difficulty:           difficulty,
			EstimatedMinutes:     estimatedMinutes,
			RequiresVerification: requiresVerification,
			RequiresPhoto:        requiresPhoto,
			Icon:                 icon,
			BasePoints:           *basePoints,
		}

		// Build assigned user info (if assigned)
		var assignedUser *models.AssignmentUserInfo
		if assignedTo != nil && assignedUserName != nil {
			assignedUser = &models.AssignmentUserInfo{
				ID:          *assignedTo,
				Username:    *assignedUsername,
				DisplayName: *assignedUserName,
				ColorTheme:  *assignedUserColor,
			}
		}

		// Determine if bonus assignment
		isBonus := assignedTo == nil || status == "open"

		// Format dates as ISO strings
		var dueDateStr, completedAtStr, verifiedAtStr, updatedAtStr *string
		if dueDate != nil {
			str := dueDate.Format("2006-01-02")
			dueDateStr = &str
		}
		if completedAt != nil {
			str := completedAt.Format(time.RFC3339)
			completedAtStr = &str
		}
		if verifiedAt != nil {
			str := verifiedAt.Format(time.RFC3339)
			verifiedAtStr = &str
		}
		if updatedAt != nil {
			str := updatedAt.Format(time.RFC3339)
			updatedAtStr = &str
		}

		assignment := models.AssignmentListResponse{
			ID:                id,
			ChoreID:           choreID,
			AssignedTo:        assignedTo,
			AssignedBy:        assignedBy,
			Status:            status,
			PointsOffered:     pointsOffered,
			PointsEarned:      pointsEarned,
			DueDate:           dueDateStr,
			CreatedAt:         createdAt.Format(time.RFC3339),
			UpdatedAt:         updatedAtStr,
			CompletedAt:       completedAtStr,
			VerifiedAt:        verifiedAtStr,
			CompletionNotes:   completionNotes,
			VerificationNotes: verificationNotes,
			IsBonus:           isBonus,
			Chore:             chore,
			AssignedUser:      assignedUser,
		}

		assignments = append(assignments, assignment)
	}

	c.JSON(http.StatusOK, gin.H{
		"assignments": assignments,
		"count":       len(assignments),
	})
}
