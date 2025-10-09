package models

import (
	"time"

	"github.com/google/uuid"
)

// PointsBalance represents a user's current point totals
type PointsBalance struct {
	UserID               uuid.UUID `json:"user_id"`
	Username             string    `json:"username"`
	DisplayName          string    `json:"display_name"`
	TotalPoints          int       `json:"total_points"`
	AvailablePoints      int       `json:"available_points"`
	WeeklyPoints         int       `json:"weekly_points"`
	LifetimePointsEarned int       `json:"lifetime_points_earned"`
	TotalPointsConverted int       `json:"total_points_converted"`
	Level                int       `json:"level"`
	XP                   int       `json:"xp"`
	StreakDays           int       `json:"streak_days"`
}

// PointTransaction represents a point transaction record
type PointTransaction struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	UserID              uuid.UUID  `json:"user_id" db:"user_id"`
	Points              int        `json:"points" db:"points"` // Positive for earned, negative for spent
	TransactionType     string     `json:"transaction_type" db:"transaction_type"`
	Description         string     `json:"description" db:"description"`
	RelatedAssignmentID *uuid.UUID `json:"related_assignment_id,omitempty" db:"related_assignment_id"`
	RelatedUserID       *uuid.UUID `json:"related_user_id,omitempty" db:"related_user_id"`
	ExtraData           *string    `json:"extra_data,omitempty" db:"extra_data"` // JSONB field
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	Reference           *uuid.UUID `json:"reference,omitempty" db:"reference"`
}

// PointTransactionResponse is the API response format
type PointTransactionResponse struct {
	ID                  uuid.UUID  `json:"id"`
	UserID              uuid.UUID  `json:"user_id"`
	Points              int        `json:"points"`
	TransactionType     string     `json:"transaction_type"`
	Description         string     `json:"description"`
	RelatedAssignmentID *uuid.UUID `json:"related_assignment_id,omitempty"`
	RelatedUserID       *uuid.UUID `json:"related_user_id,omitempty"`
	CreatedAt           string     `json:"created_at"`
}

// ToResponse converts PointTransaction to PointTransactionResponse
func (pt *PointTransaction) ToResponse() PointTransactionResponse {
	return PointTransactionResponse{
		ID:                  pt.ID,
		UserID:              pt.UserID,
		Points:              pt.Points,
		TransactionType:     pt.TransactionType,
		Description:         pt.Description,
		RelatedAssignmentID: pt.RelatedAssignmentID,
		RelatedUserID:       pt.RelatedUserID,
		CreatedAt:           pt.CreatedAt.Format(time.RFC3339),
	}
}
