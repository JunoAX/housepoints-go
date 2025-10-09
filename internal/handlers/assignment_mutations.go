package handlers

import (
	"fmt"
	"net/http"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ClaimAssignmentRequest is the request body for claiming
type ClaimAssignmentRequest struct {
	// No body needed - user is from auth context
}

// CompleteAssignmentRequest is the request body for completing
type CompleteAssignmentRequest struct {
	Notes *string `json:"notes,omitempty"`
}

// VerifyAssignmentRequest is the request body for verification
type VerifyAssignmentRequest struct {
	Approved          bool    `json:"approved"`
	PointsAwarded     *int    `json:"points_awarded,omitempty"`
	VerificationNotes *string `json:"verification_notes,omitempty"`
}

// ClaimAssignment allows a user to claim an open bonus assignment
func ClaimAssignment(c *gin.Context) {
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

	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Start transaction
	tx, err := db.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	// Check assignment exists and is open
	var status string
	err = tx.QueryRow(c.Request.Context(),
		"SELECT status FROM assignments WHERE id = $1",
		assignmentID,
	).Scan(&status)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query assignment", "details": err.Error()})
		}
		return
	}

	if status != "open" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Assignment is not available to claim"})
		return
	}

	// Claim the assignment
	_, err = tx.Exec(c.Request.Context(), `
		UPDATE assignments
		SET assigned_to = $1,
			status = 'pending',
			updated_at = NOW()
		WHERE id = $2
	`, userID, assignmentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim assignment", "details": err.Error()})
		return
	}

	// Commit transaction
	if err = tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Assignment claimed successfully",
		"assignment_id": assignmentID,
		"status":        "pending",
	})
}

// CompleteAssignment marks an assignment as completed
func CompleteAssignment(c *gin.Context) {
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

	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	isParent, _ := middleware.GetAuthIsParent(c)

	var req CompleteAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Notes are optional, so empty body is fine
		req = CompleteAssignmentRequest{}
	}

	// Start transaction
	tx, err := db.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	// Get assignment details
	var (
		assignedTo           *uuid.UUID
		status               string
		requiresVerification bool
		pointsOffered        int
	)

	err = tx.QueryRow(c.Request.Context(), `
		SELECT a.assigned_to, a.status, c.requires_verification, a.points_offered
		FROM assignments a
		JOIN chores c ON a.chore_id = c.id
		WHERE a.id = $1
	`, assignmentID).Scan(&assignedTo, &status, &requiresVerification, &pointsOffered)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query assignment", "details": err.Error()})
		}
		return
	}

	// Check permission - only assigned user or parent can complete
	if !isParent {
		if assignedTo == nil || *assignedTo != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to complete this assignment"})
			return
		}
	}

	// Check status
	if status != "pending" && status != "in_progress" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot complete assignment with status: %s", status)})
		return
	}

	// Determine new status
	var newStatus string
	var pointsEarned int

	if isParent && assignedTo != nil && *assignedTo == userID {
		// Parent completing their own chore
		if requiresVerification {
			newStatus = "pending_verification"
			pointsEarned = 0
		} else {
			newStatus = "verified"
			pointsEarned = pointsOffered
		}
	} else {
		// Child completing chore - always needs verification
		newStatus = "pending_verification"
		pointsEarned = 0
	}

	// Update assignment
	_, err = tx.Exec(c.Request.Context(), `
		UPDATE assignments
		SET status = $1,
			completed_at = NOW(),
			completion_notes = $2,
			points_earned = $3,
			updated_at = NOW()
		WHERE id = $4
	`, newStatus, req.Notes, pointsEarned, assignmentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment", "details": err.Error()})
		return
	}

	// If auto-verified, create point transaction
	if newStatus == "verified" && assignedTo != nil {
		_, err = tx.Exec(c.Request.Context(), `
			INSERT INTO point_transactions (
				id, user_id, points, transaction_type, description,
				related_assignment_id, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		`, uuid.New(), *assignedTo, pointsOffered, "chore_completion",
			"Completed chore", assignmentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create point transaction", "details": err.Error()})
			return
		}

		// Update user points
		_, err = tx.Exec(c.Request.Context(), `
			UPDATE users
			SET total_points = total_points + $1,
				available_points = available_points + $1,
				weekly_points = weekly_points + $1,
				lifetime_points_earned = lifetime_points_earned + $1,
				updated_at = NOW()
			WHERE id = $2
		`, pointsOffered, *assignedTo)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user points", "details": err.Error()})
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Assignment completed successfully",
		"assignment_id":  assignmentID,
		"status":         newStatus,
		"points_earned":  pointsEarned,
		"requires_verification": newStatus == "pending_verification",
	})
}

// VerifyAssignment verifies a completed assignment (parent only)
func VerifyAssignment(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can verify assignments"})
		return
	}

	assignmentIDParam := c.Param("id")
	assignmentID, err := uuid.Parse(assignmentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID format"})
		return
	}

	var req VerifyAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Start transaction
	tx, err := db.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	// Get assignment details
	var (
		assignedTo    *uuid.UUID
		status        string
		pointsOffered int
	)

	err = tx.QueryRow(c.Request.Context(), `
		SELECT assigned_to, status, points_offered
		FROM assignments
		WHERE id = $1
	`, assignmentID).Scan(&assignedTo, &status, &pointsOffered)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query assignment", "details": err.Error()})
		}
		return
	}

	// Check status
	if status != "completed" && status != "pending_verification" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot verify assignment with status: %s", status)})
		return
	}

	if assignedTo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Assignment has no assignee"})
		return
	}

	// Handle rejection
	if !req.Approved {
		_, err = tx.Exec(c.Request.Context(), `
			UPDATE assignments
			SET status = 'pending',
				verification_notes = $1,
				updated_at = NOW()
			WHERE id = $2
		`, req.VerificationNotes, assignmentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject assignment", "details": err.Error()})
			return
		}

		if err = tx.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "Assignment rejected - reset to pending",
			"assignment_id": assignmentID,
			"status":        "pending",
		})
		return
	}

	// Approval - determine points
	pointsAwarded := pointsOffered
	if req.PointsAwarded != nil {
		pointsAwarded = *req.PointsAwarded
	}

	// Update assignment
	_, err = tx.Exec(c.Request.Context(), `
		UPDATE assignments
		SET status = 'verified',
			verified_at = NOW(),
			verification_notes = $1,
			points_earned = $2,
			updated_at = NOW()
		WHERE id = $3
	`, req.VerificationNotes, pointsAwarded, assignmentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify assignment", "details": err.Error()})
		return
	}

	// Create point transaction
	_, err = tx.Exec(c.Request.Context(), `
		INSERT INTO point_transactions (
			id, user_id, points, transaction_type, description,
			related_assignment_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, uuid.New(), *assignedTo, pointsAwarded, "chore_completion",
		"Completed chore", assignmentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create point transaction", "details": err.Error()})
		return
	}

	// Update user points
	_, err = tx.Exec(c.Request.Context(), `
		UPDATE users
		SET total_points = total_points + $1,
			available_points = available_points + $1,
			weekly_points = weekly_points + $1,
			lifetime_points_earned = lifetime_points_earned + $1,
			updated_at = NOW()
		WHERE id = $2
	`, pointsAwarded, *assignedTo)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user points", "details": err.Error()})
		return
	}

	// Commit transaction
	if err = tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Assignment verified successfully",
		"assignment_id":  assignmentID,
		"status":         "verified",
		"points_awarded": pointsAwarded,
	})
}
