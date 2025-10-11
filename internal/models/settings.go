package models

import (
	"time"

	"github.com/google/uuid"
)

// SystemSetting represents a system-wide configuration setting
type SystemSetting struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	SettingKey  string     `json:"setting_key" db:"setting_key"`
	SettingValue string    `json:"setting_value" db:"setting_value"`
	SettingType string     `json:"setting_type" db:"setting_type"` // int, float, bool, string, dict, list
	Description *string    `json:"description,omitempty" db:"description"`
	Category    *string    `json:"category,omitempty" db:"category"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
}

// SettingUpdateRequest is the request body for PUT /api/settings/:key
type SettingUpdateRequest struct {
	Value interface{} `json:"value" binding:"required"`
}

// SettingsResponse is the response format for GET /api/settings
type SettingsResponse map[string]interface{}
