package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a family member in the family database
type User struct {
	ID                        uuid.UUID  `json:"id" db:"id"`
	Username                  string     `json:"username" db:"username"`
	DisplayName               string     `json:"display_name" db:"display_name"`
	Age                       *int       `json:"age,omitempty" db:"age"`
	ColorTheme                string     `json:"color_theme" db:"color_theme"`
	AvatarURL                 *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	TotalPoints               int        `json:"total_points" db:"total_points"`
	WeeklyPoints              int        `json:"weekly_points" db:"weekly_points"`
	Level                     int        `json:"level" db:"level"`
	XP                        int        `json:"xp" db:"xp"`
	StreakDays                int        `json:"streak_days" db:"streak_days"`
	LastActive                time.Time  `json:"last_active" db:"last_active"`
	Preferences               string     `json:"preferences,omitempty" db:"preferences"` // JSONB stored as string
	CreatedAt                 time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at" db:"updated_at"`
	IsParent                  bool       `json:"is_parent" db:"is_parent"`
	Birthdate                 *time.Time `json:"birthdate,omitempty" db:"birthdate"`
	AvailabilityNotifications bool       `json:"availability_notifications" db:"availability_notifications"`
	AutoApproveWork           bool       `json:"auto_approve_work" db:"auto_approve_work"`
	Email                     *string    `json:"email,omitempty" db:"email"`
	PhoneNumber               *string    `json:"phone_number,omitempty" db:"phone_number"`
	School                    *string    `json:"school,omitempty" db:"school"`
	Notes                     *string    `json:"notes,omitempty" db:"notes"`
	LoginEnabled              bool       `json:"login_enabled" db:"login_enabled"`
	DailyGoal                 int        `json:"daily_goal" db:"daily_goal"`
	UsuallyEatsDinner         bool       `json:"usually_eats_dinner" db:"usually_eats_dinner"`
	TotalPointsConverted      int        `json:"total_points_converted" db:"total_points_converted"`
	LifetimePointsEarned      int        `json:"lifetime_points_earned" db:"lifetime_points_earned"`
	AvailablePoints           int        `json:"available_points" db:"available_points"`
	LifetimePoints            int        `json:"lifetime_points" db:"lifetime_points"`
	LastLogin                 *time.Time `json:"last_login,omitempty" db:"last_login"`
	IsActive                  bool       `json:"is_active" db:"is_active"`
}

// UserListResponse is the simplified response for user lists
type UserListResponse struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	Age          *int      `json:"age,omitempty"`
	ColorTheme   string    `json:"color_theme"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	TotalPoints  int       `json:"total_points"`
	WeeklyPoints int       `json:"weekly_points"`
	Level        int       `json:"level"`
	IsParent     bool      `json:"is_parent"`
	IsActive     bool      `json:"is_active"`
}

// UserDetailResponse includes more information for single user requests
type UserDetailResponse struct {
	ID                        uuid.UUID  `json:"id"`
	Username                  string     `json:"username"`
	DisplayName               string     `json:"display_name"`
	Age                       *int       `json:"age,omitempty"`
	ColorTheme                string     `json:"color_theme"`
	AvatarURL                 *string    `json:"avatar_url,omitempty"`
	TotalPoints               int        `json:"total_points"`
	WeeklyPoints              int        `json:"weekly_points"`
	Level                     int        `json:"level"`
	XP                        int        `json:"xp"`
	StreakDays                int        `json:"streak_days"`
	LastActive                time.Time  `json:"last_active"`
	IsParent                  bool       `json:"is_parent"`
	Birthdate                 *time.Time `json:"birthdate,omitempty"`
	AvailabilityNotifications bool       `json:"availability_notifications"`
	AutoApproveWork           bool       `json:"auto_approve_work"`
	Email                     *string    `json:"email,omitempty"`
	PhoneNumber               *string    `json:"phone_number,omitempty"`
	School                    *string    `json:"school,omitempty"`
	DailyGoal                 int        `json:"daily_goal"`
	UsuallyEatsDinner         bool       `json:"usually_eats_dinner"`
	AvailablePoints           int        `json:"available_points"`
	LifetimePointsEarned      int        `json:"lifetime_points_earned"`
	IsActive                  bool       `json:"is_active"`
	CreatedAt                 time.Time  `json:"created_at"`
}

// ToListResponse converts User to UserListResponse
func (u *User) ToListResponse() UserListResponse {
	return UserListResponse{
		ID:           u.ID,
		Username:     u.Username,
		DisplayName:  u.DisplayName,
		Age:          u.Age,
		ColorTheme:   u.ColorTheme,
		AvatarURL:    u.AvatarURL,
		TotalPoints:  u.TotalPoints,
		WeeklyPoints: u.WeeklyPoints,
		Level:        u.Level,
		IsParent:     u.IsParent,
		IsActive:     u.IsActive,
	}
}

// ToDetailResponse converts User to UserDetailResponse
func (u *User) ToDetailResponse() UserDetailResponse {
	return UserDetailResponse{
		ID:                        u.ID,
		Username:                  u.Username,
		DisplayName:               u.DisplayName,
		Age:                       u.Age,
		ColorTheme:                u.ColorTheme,
		AvatarURL:                 u.AvatarURL,
		TotalPoints:               u.TotalPoints,
		WeeklyPoints:              u.WeeklyPoints,
		Level:                     u.Level,
		XP:                        u.XP,
		StreakDays:                u.StreakDays,
		LastActive:                u.LastActive,
		IsParent:                  u.IsParent,
		Birthdate:                 u.Birthdate,
		AvailabilityNotifications: u.AvailabilityNotifications,
		AutoApproveWork:           u.AutoApproveWork,
		Email:                     u.Email,
		PhoneNumber:               u.PhoneNumber,
		School:                    u.School,
		DailyGoal:                 u.DailyGoal,
		UsuallyEatsDinner:         u.UsuallyEatsDinner,
		AvailablePoints:           u.AvailablePoints,
		LifetimePointsEarned:      u.LifetimePointsEarned,
		IsActive:                  u.IsActive,
		CreatedAt:                 u.CreatedAt,
	}
}
