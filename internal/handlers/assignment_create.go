package handlers

import (
	"net/http"
	"time"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateAssignment creates a new assignment (parent only)
func CreateAssignment(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can create assignments"})
		return
	}

	userID, _ := middleware.GetAuthUserID(c)

	var req models.AssignmentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Set default points if not provided
	if req.PointsOffered == 0 {
		req.PointsOffered = 10
	}

	// Check if chore exists
	var choreName string
	err := db.QueryRow(c.Request.Context(), "SELECT name FROM chores WHERE id = $1", req.ChoreID).Scan(&choreName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chore not found"})
		return
	}

	// Check if assigned user exists (if provided)
	var assigneeName *string
	if req.AssignedTo != nil {
		err := db.QueryRow(c.Request.Context(), "SELECT display_name FROM users WHERE id = $1", req.AssignedTo).Scan(&assigneeName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Assigned user not found"})
			return
		}
	}

	// Parse due date or default to end of today
	var dueDate time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		// Try parsing as date-time first (ISO format)
		parsed, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			// Try parsing as date only (YYYY-MM-DD)
			parsed, err = time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due_date format. Use YYYY-MM-DD or RFC3339"})
				return
			}
			// Set to end of day
			dueDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, parsed.Location())
		} else {
			dueDate = parsed
		}
	} else {
		// Default to end of today
		now := time.Now()
		dueDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	}

	// Determine status based on whether it's assigned
	status := "pending"
	if req.AssignedTo == nil {
		status = "open"
	}

	// Create assignment
	assignmentID := uuid.New()
	query := `
		INSERT INTO assignments (
			id, chore_id, assigned_to, assigned_by, status,
			points_offered, due_date, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id
	`

	var returnedID uuid.UUID
	err = db.QueryRow(c.Request.Context(), query,
		assignmentID, req.ChoreID, req.AssignedTo, userID, status,
		req.PointsOffered, dueDate,
	).Scan(&returnedID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             returnedID,
		"chore_id":       req.ChoreID,
		"assigned_to":    req.AssignedTo,
		"assigned_by":    userID,
		"status":         status,
		"points_offered": req.PointsOffered,
		"due_date":       dueDate.Format(time.RFC3339),
		"message":        "Assignment created successfully",
	})
}
