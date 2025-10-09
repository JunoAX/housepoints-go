package models

import (
	"time"

	"github.com/google/uuid"
)

// Family represents a family account in the multi-tenant system
type Family struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Slug      string     `json:"slug" db:"slug"`                 // Unique identifier for subdomain (e.g., "gamull", "smith-nyc")
	Name      string     `json:"name" db:"name"`                 // Display name (e.g., "The Gamull Family")

	// Database connection info
	DBHost               string `json:"-" db:"db_host"`
	DBPort               int    `json:"-" db:"db_port"`
	DBName               string `json:"-" db:"db_name"`
	DBUser               string `json:"-" db:"db_user"`
	DBPasswordEncrypted  string `json:"-" db:"db_password_encrypted"`

	// Subscription
	Plan      string     `json:"plan" db:"plan"`                 // Subscription plan: "free", "premium", "enterprise"
	Status    string     `json:"status" db:"status"`             // "trial", "active", "suspended", "cancelled"

	// Metadata
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"` // Soft delete
}

// FamilySettings represents family-specific configuration
type FamilySettings struct {
	ID                uuid.UUID `json:"id" db:"id"`
	FamilyID          uuid.UUID `json:"family_id" db:"family_id"`
	Timezone          string    `json:"timezone" db:"timezone"`                     // Family timezone (e.g., "America/New_York")
	Currency          string    `json:"currency" db:"currency"`                     // Currency code (e.g., "USD")
	WeekStartDay      int       `json:"week_start_day" db:"week_start_day"`         // 0=Sunday, 1=Monday
	ThemeColor        string    `json:"theme_color" db:"theme_color"`               // Primary color for family branding
	CustomDomain      *string   `json:"custom_domain,omitempty" db:"custom_domain"` // Optional custom domain
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// FamilyMember represents a user's membership in a family
type FamilyMember struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	FamilyID  uuid.UUID  `json:"family_id" db:"family_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Role      string     `json:"role" db:"role"` // "parent", "child", "admin"
	Active    bool       `json:"active" db:"active"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}
