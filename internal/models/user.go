package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user account (can belong to multiple families)
type User struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Username      string     `json:"username" db:"username"`
	Email         string     `json:"email" db:"email"`
	PasswordHash  string     `json:"-" db:"password_hash"` // Never expose in JSON
	FirstName     string     `json:"first_name" db:"first_name"`
	LastName      string     `json:"last_name" db:"last_name"`
	AvatarURL     *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	EmailVerified bool       `json:"email_verified" db:"email_verified"`
	Active        bool       `json:"active" db:"active"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}
