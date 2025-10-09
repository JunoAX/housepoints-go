package models

import (
	"time"

	"github.com/google/uuid"
)

// Assignment represents a chore assignment
type Assignment struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	ChoreID            uuid.UUID  `json:"chore_id" db:"chore_id"`
	AssignedTo         *uuid.UUID `json:"assigned_to,omitempty" db:"assigned_to"`
	AssignedBy         *uuid.UUID `json:"assigned_by,omitempty" db:"assigned_by"`
	Status             string     `json:"status" db:"status"`
	PointsOffered      int        `json:"points_offered" db:"points_offered"`
	PointsEarned       *int       `json:"points_earned,omitempty" db:"points_earned"`
	DueDate            *time.Time `json:"due_date,omitempty" db:"due_date"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	CompletionNotes    *string    `json:"completion_notes,omitempty" db:"completion_notes"`
	VerificationNotes  *string    `json:"verification_notes,omitempty" db:"verification_notes"`
}

// AssignmentChoreInfo contains chore details for assignment responses
type AssignmentChoreInfo struct {
	ID                   uuid.UUID `json:"id"`
	Name                 string    `json:"name"`
	Description          *string   `json:"description,omitempty"`
	Category             string    `json:"category"`
	Difficulty           string    `json:"difficulty"`
	EstimatedMinutes     *int      `json:"estimated_minutes,omitempty"`
	RequiresVerification bool      `json:"requires_verification"`
	RequiresPhoto        bool      `json:"requires_photo"`
	Icon                 string    `json:"icon"`
	BasePoints           int       `json:"base_points"`
}

// AssignmentUserInfo contains user details for assignment responses
type AssignmentUserInfo struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	ColorTheme  string    `json:"color_theme"`
}

// AssignmentListResponse is the response for assignment lists
type AssignmentListResponse struct {
	ID                uuid.UUID           `json:"id"`
	ChoreID           uuid.UUID           `json:"chore_id"`
	AssignedTo        *uuid.UUID          `json:"assigned_to,omitempty"`
	AssignedBy        *uuid.UUID          `json:"assigned_by,omitempty"`
	Status            string              `json:"status"`
	PointsOffered     int                 `json:"points_offered"`
	PointsEarned      int                 `json:"points_earned"`
	DueDate           *string             `json:"due_date,omitempty"` // ISO date string
	CreatedAt         string              `json:"created_at"`
	UpdatedAt         *string             `json:"updated_at,omitempty"`
	CompletedAt       *string             `json:"completed_at,omitempty"`
	VerifiedAt        *string             `json:"verified_at,omitempty"`
	CompletionNotes   *string             `json:"completion_notes,omitempty"`
	VerificationNotes *string             `json:"verification_notes,omitempty"`
	IsBonus           bool                `json:"is_bonus"` // true if unassigned or open status
	Chore             AssignmentChoreInfo `json:"chore"`
	AssignedUser      *AssignmentUserInfo `json:"assigned_user,omitempty"`
}

// AssignmentDetailResponse includes full details for a single assignment
type AssignmentDetailResponse struct {
	ID                uuid.UUID            `json:"id"`
	ChoreID           uuid.UUID            `json:"chore_id"`
	AssignedTo        *uuid.UUID           `json:"assigned_to,omitempty"`
	AssignedBy        *uuid.UUID           `json:"assigned_by,omitempty"`
	Status            string               `json:"status"`
	PointsOffered     int                  `json:"points_offered"`
	PointsEarned      int                  `json:"points_earned"`
	DueDate           *string              `json:"due_date,omitempty"`
	CreatedAt         string               `json:"created_at"`
	UpdatedAt         *string              `json:"updated_at,omitempty"`
	CompletedAt       *string              `json:"completed_at,omitempty"`
	VerifiedAt        *string              `json:"verified_at,omitempty"`
	CompletionNotes   *string              `json:"completion_notes,omitempty"`
	VerificationNotes *string              `json:"verification_notes,omitempty"`
	IsBonus           bool                 `json:"is_bonus"`
	Chore             AssignmentChoreInfo  `json:"chore"`
	AssignedUser      *AssignmentUserInfo  `json:"assigned_user,omitempty"`
	AssignedByUser    *AssignmentUserInfo  `json:"assigned_by_user,omitempty"`
}
