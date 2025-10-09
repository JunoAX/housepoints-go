package models

import (
	"time"

	"github.com/google/uuid"
)

// Reward represents a rewardthat can be redeemed for points
type Reward struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	Name                   string     `json:"name" db:"name"`
	Description            *string    `json:"description,omitempty" db:"description"`
	CostPoints             int        `json:"cost_points" db:"cost_points"`
	Category               *string    `json:"category,omitempty" db:"category"`
	Icon                   string     `json:"icon" db:"icon"`
	MaxPerWeek             *int       `json:"max_per_week,omitempty" db:"max_per_week"`
	RequiresParentApproval bool       `json:"requires_parent_approval" db:"requires_parent_approval"`
	Active                 bool       `json:"active" db:"active"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	Availability           string     `json:"availability" db:"availability"`
	StockRemaining         *int       `json:"stock_remaining,omitempty" db:"stock_remaining"`
	ValueInCents           *int       `json:"value_in_cents,omitempty" db:"value_in_cents"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy              *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy              *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

// RewardListResponse is the simplified response for reward lists
type RewardListResponse struct {
	ID                     uuid.UUID `json:"id"`
	Name                   string    `json:"name"`
	Description            *string   `json:"description,omitempty"`
	CostPoints             int       `json:"cost_points"`
	Category               *string   `json:"category,omitempty"`
	Icon                   string    `json:"icon"`
	MaxPerWeek             *int      `json:"max_per_week,omitempty"`
	RequiresParentApproval bool      `json:"requires_parent_approval"`
	Active                 bool      `json:"active"`
	Availability           string    `json:"availability"`
	StockRemaining         *int      `json:"stock_remaining,omitempty"`
	UserRedemptionCount    int       `json:"user_redemption_count"` // How many times current user has redeemed
}

// RewardRedemption represents a reward redemption/purchase record
type RewardRedemption struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	RewardID    uuid.UUID  `json:"reward_id" db:"reward_id"`
	PointsSpent int        `json:"points_spent" db:"points_spent"`
	Status      string     `json:"status" db:"status"` // pending, approved, rejected, completed
	Notes       *string    `json:"notes,omitempty" db:"notes"`
	ParentNotes *string    `json:"parent_notes,omitempty" db:"parent_notes"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	ProcessedBy *uuid.UUID `json:"processed_by,omitempty" db:"processed_by"`
}

// RewardRedemptionResponse is the API response format
type RewardRedemptionResponse struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	RewardID    uuid.UUID `json:"reward_id"`
	RewardName  string    `json:"reward_name"`
	PointsSpent int       `json:"points_spent"`
	Status      string    `json:"status"`
	Notes       *string   `json:"notes,omitempty"`
	ParentNotes *string   `json:"parent_notes,omitempty"`
	CreatedAt   string    `json:"created_at"`
	ProcessedAt *string   `json:"processed_at,omitempty"`
}

// ToResponse converts RewardRedemption to RewardRedemptionResponse
func (rr *RewardRedemption) ToResponse(rewardName string) RewardRedemptionResponse {
	response := RewardRedemptionResponse{
		ID:          rr.ID,
		UserID:      rr.UserID,
		RewardID:    rr.RewardID,
		RewardName:  rewardName,
		PointsSpent: rr.PointsSpent,
		Status:      rr.Status,
		Notes:       rr.Notes,
		ParentNotes: rr.ParentNotes,
		CreatedAt:   rr.CreatedAt.Format(time.RFC3339),
	}

	if rr.ProcessedAt != nil {
		processedStr := rr.ProcessedAt.Format(time.RFC3339)
		response.ProcessedAt = &processedStr
	}

	return response
}
