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

// RedeemRewardRequest is the request body for redeeming a reward
type RedeemRewardRequest struct {
	Notes *string `json:"notes,omitempty"`
}

// ListRewards returns all active rewards with user-specific redemption counts
func ListRewards(c *gin.Context) {
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
			r.id, r.name, r.description, r.cost_points, r.category, r.icon,
			r.max_per_week, r.requires_parent_approval, r.active, r.availability,
			r.stock_remaining,
			COALESCE(COUNT(CASE WHEN rr.user_id = $1 THEN 1 END), 0)::int as user_redemption_count
		FROM rewards r
		LEFT JOIN reward_redemptions rr ON r.id = rr.reward_id
			AND rr.user_id = $1
			AND rr.status IN ('pending', 'approved', 'completed')
		WHERE r.active = true
		GROUP BY r.id
		ORDER BY r.name ASC
	`

	rows, err := db.Query(c.Request.Context(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query rewards", "details": err.Error()})
		return
	}
	defer rows.Close()

	rewards := []models.RewardListResponse{}
	for rows.Next() {
		var reward models.RewardListResponse

		err := rows.Scan(
			&reward.ID,
			&reward.Name,
			&reward.Description,
			&reward.CostPoints,
			&reward.Category,
			&reward.Icon,
			&reward.MaxPerWeek,
			&reward.RequiresParentApproval,
			&reward.Active,
			&reward.Availability,
			&reward.StockRemaining,
			&reward.UserRedemptionCount,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse reward data", "details": err.Error()})
			return
		}

		rewards = append(rewards, reward)
	}

	c.JSON(http.StatusOK, gin.H{
		"rewards": rewards,
		"count":   len(rewards),
	})
}

// RedeemReward allows a user to redeem a reward for points
func RedeemReward(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	rewardIDParam := c.Param("id")
	rewardID, err := uuid.Parse(rewardIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reward ID format"})
		return
	}

	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req RedeemRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Notes are optional
		req = RedeemRewardRequest{}
	}

	// Start transaction
	tx, err := db.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	// Get reward details
	var (
		rewardName             string
		costPoints             int
		requiresParentApproval bool
		active                 bool
		stockRemaining         *int
	)

	err = tx.QueryRow(c.Request.Context(), `
		SELECT name, cost_points, requires_parent_approval, active, stock_remaining
		FROM rewards
		WHERE id = $1
	`, rewardID).Scan(&rewardName, &costPoints, &requiresParentApproval, &active, &stockRemaining)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Reward not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query reward", "details": err.Error()})
		}
		return
	}

	if !active {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reward is not active"})
		return
	}

	// Check stock
	if stockRemaining != nil && *stockRemaining <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reward is out of stock"})
		return
	}

	// Check user has enough points
	var availablePoints int
	err = tx.QueryRow(c.Request.Context(),
		"SELECT available_points FROM users WHERE id = $1",
		userID,
	).Scan(&availablePoints)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user points", "details": err.Error()})
		return
	}

	if availablePoints < costPoints {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Insufficient points",
			"points_available":  availablePoints,
			"points_required":   costPoints,
			"points_short":      costPoints - availablePoints,
		})
		return
	}

	// Determine status based on approval requirement
	status := "approved"
	if requiresParentApproval {
		status = "pending"
	}

	// Create redemption record
	redemptionID := uuid.New()
	_, err = tx.Exec(c.Request.Context(), `
		INSERT INTO reward_redemptions (
			id, user_id, reward_id, points_spent, status, notes, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, redemptionID, userID, rewardID, costPoints, status, req.Notes)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create redemption", "details": err.Error()})
		return
	}

	// Deduct points from user (available_points only - trigger handles the rest)
	_, err = tx.Exec(c.Request.Context(), `
		UPDATE users
		SET available_points = available_points - $1,
			updated_at = NOW()
		WHERE id = $2
	`, costPoints, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct points", "details": err.Error()})
		return
	}

	// Create negative point transaction
	_, err = tx.Exec(c.Request.Context(), `
		INSERT INTO point_transactions (
			id, user_id, points, transaction_type, description, created_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
	`, uuid.New(), userID, -costPoints, "reward_redemption", fmt.Sprintf("Redeemed: %s", rewardName))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}

	// Update stock if limited
	if stockRemaining != nil {
		_, err = tx.Exec(c.Request.Context(), `
			UPDATE rewards
			SET stock_remaining = stock_remaining - 1,
				updated_at = NOW()
			WHERE id = $1
		`, rewardID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock", "details": err.Error()})
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                  "Reward redeemed successfully",
		"redemption_id":            redemptionID,
		"reward_name":              rewardName,
		"points_spent":             costPoints,
		"status":                   status,
		"requires_parent_approval": requiresParentApproval,
	})
}

// CreateReward creates a new reward (parent only)
func CreateReward(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can create rewards"})
		return
	}

	var req models.RewardCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Set defaults
	if req.Icon == "" {
		req.Icon = "🎁"
	}
	if req.Availability == "" {
		req.Availability = "always"
	}

	rewardID := uuid.New()
	query := `
		INSERT INTO rewards (
			id, name, description, cost_points, category, availability,
			stock_remaining, icon, value_in_cents, requires_parent_approval,
			active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id
	`

	var returnedID uuid.UUID
	err := db.QueryRow(c.Request.Context(), query,
		rewardID, req.Name, req.Description, req.CostPoints, req.Category,
		req.Availability, req.StockRemaining, req.Icon, req.ValueInCents,
		req.RequiresParentApproval, req.Active,
	).Scan(&returnedID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reward", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      returnedID,
		"message": "Reward created successfully",
	})
}

// UpdateReward updates an existing reward (parent only)
func UpdateReward(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can update rewards"})
		return
	}

	rewardIDParam := c.Param("id")
	rewardID, err := uuid.Parse(rewardIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reward ID format"})
		return
	}

	var req models.RewardUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Check if reward exists
	var exists bool
	err = db.QueryRow(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM rewards WHERE id = $1)", rewardID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reward not found"})
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

	if req.CostPoints != nil {
		updates = append(updates, fmt.Sprintf("cost_points = $%d", argIndex))
		args = append(args, *req.CostPoints)
		argIndex++
	}

	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *req.Category)
		argIndex++
	}

	if req.Availability != nil {
		updates = append(updates, fmt.Sprintf("availability = $%d", argIndex))
		args = append(args, *req.Availability)
		argIndex++
	}

	if req.StockRemaining != nil {
		updates = append(updates, fmt.Sprintf("stock_remaining = $%d", argIndex))
		args = append(args, req.StockRemaining)
		argIndex++
	}

	if req.Icon != nil {
		updates = append(updates, fmt.Sprintf("icon = $%d", argIndex))
		args = append(args, *req.Icon)
		argIndex++
	}

	if req.ValueInCents != nil {
		updates = append(updates, fmt.Sprintf("value_in_cents = $%d", argIndex))
		args = append(args, req.ValueInCents)
		argIndex++
	}

	if req.RequiresParentApproval != nil {
		updates = append(updates, fmt.Sprintf("requires_parent_approval = $%d", argIndex))
		args = append(args, *req.RequiresParentApproval)
		argIndex++
	}

	if req.Active != nil {
		updates = append(updates, fmt.Sprintf("active = $%d", argIndex))
		args = append(args, *req.Active)
		argIndex++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, rewardID)

	query := fmt.Sprintf(`
		UPDATE rewards
		SET %s
		WHERE id = $%d
		RETURNING id, name, description, cost_points, category, availability,
			stock_remaining, icon, value_in_cents, requires_parent_approval, active
	`, strings.Join(updates, ", "), argIndex)

	var reward models.Reward
	err = db.QueryRow(c.Request.Context(), query, args...).Scan(
		&reward.ID, &reward.Name, &reward.Description, &reward.CostPoints,
		&reward.Category, &reward.Availability, &reward.StockRemaining,
		&reward.Icon, &reward.ValueInCents, &reward.RequiresParentApproval,
		&reward.Active,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reward", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reward":  reward,
		"message": "Reward updated successfully",
	})
}

// DeleteReward soft-deletes a reward (parent only)
func DeleteReward(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can delete rewards"})
		return
	}

	rewardIDParam := c.Param("id")
	rewardID, err := uuid.Parse(rewardIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reward ID format"})
		return
	}

	// Soft delete by setting active to false
	result, err := db.Exec(c.Request.Context(), `
		UPDATE rewards
		SET active = false, updated_at = NOW()
		WHERE id = $1
	`, rewardID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reward", "details": err.Error()})
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reward not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reward deleted successfully",
	})
}
