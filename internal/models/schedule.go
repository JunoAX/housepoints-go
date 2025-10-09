package models

import (
	"time"

	"github.com/google/uuid"
)

// FamilyScheduleEntry represents a single day in the family schedule
type FamilyScheduleEntry struct {
	ID                uuid.UUID              `json:"id"`
	Date              string                 `json:"date"`
	Weekday           *string                `json:"weekday,omitempty"`
	DayType           *string                `json:"day_type,omitempty"`
	PresenceData      map[string]interface{} `json:"presence_data,omitempty"`
	KidsPresent       []string               `json:"kids_present"`
	TotalKidsPresent  int                    `json:"total_kids_present"`
	IsJohnWeekend     bool                   `json:"is_john_weekend"`
	Notes             *string                `json:"notes,omitempty"`
	TransitionTime    *string                `json:"transition_time,omitempty"`
	TransitionType    *string                `json:"transition_type,omitempty"`
	CustodyPriority   *int                   `json:"custody_priority,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// FamilyScheduleResponse is the API response for schedule queries
type FamilyScheduleResponse struct {
	StartDate string                 `json:"start_date"`
	EndDate   string                 `json:"end_date"`
	Schedule  []FamilyScheduleEntry  `json:"schedule"`
	TotalDays int                    `json:"total_days"`
}
