package models

import (
	"time"

	"github.com/google/uuid"
)

// Chore represents a task that can be assigned to family members
type Chore struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	Name                 string     `json:"name" db:"name"`
	Description          *string    `json:"description,omitempty" db:"description"`
	Instructions         *string    `json:"instructions,omitempty" db:"instructions"`
	Category             string     `json:"category" db:"category"`
	BasePoints           int        `json:"base_points" db:"base_points"`
	EstimatedMinutes     *int       `json:"estimated_minutes,omitempty" db:"estimated_minutes"`
	Difficulty           string     `json:"difficulty" db:"difficulty"`
	RequiresVerification bool       `json:"requires_verification" db:"requires_verification"`
	RequiresPhoto        bool       `json:"requires_photo" db:"requires_photo"`
	Icon                 string     `json:"icon" db:"icon"`
	Tags                 []string   `json:"tags,omitempty" db:"tags"`
	Frequency            *string    `json:"frequency,omitempty" db:"frequency"`
	Active               bool       `json:"active" db:"active"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
	MinAge               int        `json:"min_age" db:"min_age"`
	AssignmentType       string     `json:"assignment_type" db:"assignment_type"`
	RotationEligible     bool       `json:"rotation_eligible" db:"rotation_eligible"`
}

// ChoreListResponse is a simplified version for list endpoints
type ChoreListResponse struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Category         string    `json:"category"`
	BasePoints       int       `json:"base_points"`
	EstimatedMinutes *int      `json:"estimated_minutes,omitempty"`
	Difficulty       string    `json:"difficulty"`
	Icon             string    `json:"icon"`
	Active           bool      `json:"active"`
	AssignmentType   string    `json:"assignment_type"`
}
