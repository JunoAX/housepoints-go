package handlers

import (
	"fmt"
	"net/http"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetUserPoints returns a user's current point balance and stats
func GetUserPoints(c *gin.Context) {
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
			id, username, display_name,
			total_points, available_points, weekly_points,
			lifetime_points_earned, total_points_converted,
			level, xp, streak_days
		FROM users
		WHERE id = $1 AND is_active = true
	`

	var balance models.PointsBalance
	err = db.QueryRow(c.Request.Context(), query, userID).Scan(
		&balance.UserID,
		&balance.Username,
		&balance.DisplayName,
		&balance.TotalPoints,
		&balance.AvailablePoints,
		&balance.WeeklyPoints,
		&balance.LifetimePointsEarned,
		&balance.TotalPointsConverted,
		&balance.Level,
		&balance.XP,
		&balance.StreakDays,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user points", "details": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, balance)
}

// GetUserTransactions returns a user's point transaction history
func GetUserTransactions(c *gin.Context) {
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

	// Optional filters
	transactionType := c.Query("type") // e.g., "assignment", "reward", "adjustment"
	limitParam := c.DefaultQuery("limit", "50")
	offsetParam := c.DefaultQuery("offset", "0")

	query := `
		SELECT
			id, user_id, points, transaction_type, description,
			related_assignment_id, related_user_id, created_at
		FROM point_transactions
		WHERE user_id = $1
	`

	params := []interface{}{userID}
	paramCount := 1

	if transactionType != "" {
		paramCount++
		query += fmt.Sprintf(" AND transaction_type = $%d", paramCount)
		params = append(params, transactionType)
	}

	query += " ORDER BY created_at DESC"

	// Add pagination
	paramCount++
	query += fmt.Sprintf(" LIMIT $%d", paramCount)
	params = append(params, limitParam)

	paramCount++
	query += fmt.Sprintf(" OFFSET $%d", paramCount)
	params = append(params, offsetParam)

	rows, err := db.Query(c.Request.Context(), query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query transactions", "details": err.Error()})
		return
	}
	defer rows.Close()

	transactions := []models.PointTransactionResponse{}
	for rows.Next() {
		var pt models.PointTransaction

		err := rows.Scan(
			&pt.ID,
			&pt.UserID,
			&pt.Points,
			&pt.TransactionType,
			&pt.Description,
			&pt.RelatedAssignmentID,
			&pt.RelatedUserID,
			&pt.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse transaction data", "details": err.Error()})
			return
		}

		transactions = append(transactions, pt.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
	})
}
