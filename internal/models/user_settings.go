package models

import (
	"time"

	"github.com/google/uuid"
)

// UserProfile is the complete user profile returned by GET /api/users/me
type UserProfile struct {
	ID                       uuid.UUID `json:"id"`
	Username                 string    `json:"username"`
	DisplayName              string    `json:"display_name"`
	Email                    *string   `json:"email,omitempty"`
	PhoneNumber              *string   `json:"phone_number,omitempty"`
	ColorTheme               string    `json:"color_theme"`
	AvatarURL                *string   `json:"avatar_url,omitempty"`
	IsParent                 bool      `json:"is_parent"`
	Birthdate                *string   `json:"birthdate,omitempty"`
	Age                      *int      `json:"age,omitempty"`
	DailyGoal                int       `json:"daily_goal"`
	UsuallyEatsDinner        bool      `json:"usually_eats_dinner"`
	TotalPoints              int       `json:"total_points"`
	AvailablePoints          int       `json:"available_points"`
	WeeklyPoints             int       `json:"weekly_points"`
	LifetimePointsEarned     int       `json:"lifetime_points_earned"`
	Level                    int       `json:"level"`
	XP                       int       `json:"xp"`
	StreakDays               int       `json:"streak_days"`
	AutoApproveWork          bool      `json:"auto_approve_work"`
	AvailabilityNotifications bool     `json:"availability_notifications"`
	LoginEnabled             bool      `json:"login_enabled"`
	IsActive                 bool      `json:"is_active"`
	Preferences              map[string]interface{} `json:"preferences"`
	LastActive               *time.Time `json:"last_active,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// UpdateProfileRequest is the request body for PATCH /api/users/me
type UpdateProfileRequest struct {
	DisplayName              *string `json:"display_name,omitempty"`
	Email                    *string `json:"email,omitempty"`
	PhoneNumber              *string `json:"phone_number,omitempty"`
	ColorTheme               *string `json:"color_theme,omitempty"`
	AvatarURL                *string `json:"avatar_url,omitempty"`
	Birthdate                *string `json:"birthdate,omitempty"`
	DailyGoal                *int    `json:"daily_goal,omitempty"`
	UsuallyEatsDinner        *bool   `json:"usually_eats_dinner,omitempty"`
	AutoApproveWork          *bool   `json:"auto_approve_work,omitempty"`
	AvailabilityNotifications *bool   `json:"availability_notifications,omitempty"`
}

// UpdatePreferencesRequest is the request body for PUT /api/users/me/preferences
type UpdatePreferencesRequest struct {
	Preferences map[string]interface{} `json:"preferences" binding:"required"`
}

// UserPreferences contains structured preference data
type UserPreferences struct {
	Notifications *NotificationPreferences `json:"notifications,omitempty"`
	General       *GeneralPreferences      `json:"general,omitempty"`
	Security      *SecurityPreferences     `json:"security,omitempty"`
}

// NotificationPreferences for notification settings
type NotificationPreferences struct {
	EmailEnabled         bool   `json:"email_enabled"`
	PushEnabled          bool   `json:"push_enabled"`
	SMSEnabled           bool   `json:"sms_enabled"`
	ChoreReminders       bool   `json:"chore_reminders"`
	ChoreAssignments     bool   `json:"chore_assignments"`
	ChoreCompletions     bool   `json:"chore_completions"`
	RewardEarned         bool   `json:"reward_earned"`
	WeeklySummary        bool   `json:"weekly_summary"`
	ReminderTime         string `json:"reminder_time"`
	ReminderDaysBefore   int    `json:"reminder_days_before"`
}

// GeneralPreferences for general app settings
type GeneralPreferences struct {
	Timezone       string `json:"timezone"`
	DateFormat     string `json:"date_format"`
	TimeFormat     string `json:"time_format"`
	WeekStartsOn   string `json:"week_starts_on"`
	Theme          string `json:"theme"`
	Language       string `json:"language"`
	SoundEffects   bool   `json:"sound_effects"`
	Animations     bool   `json:"animations"`
	DefaultView    string `json:"default_view"`
	ItemsPerPage   int    `json:"items_per_page"`
	CurrencySymbol string `json:"currency_symbol"`
	PointValue     int    `json:"point_value"`
}

// SecurityPreferences for security settings
type SecurityPreferences struct {
	TwoFactorEnabled         bool    `json:"two_factor_enabled"`
	SessionTimeout           int     `json:"session_timeout"`
	PasswordExpiresDays      int     `json:"password_expires_days"`
	RequireStrongPassword    bool    `json:"require_strong_password"`
	LoginNotifications       bool    `json:"login_notifications"`
	SuspiciousActivityAlerts bool    `json:"suspicious_activity_alerts"`
	LastPasswordChange       *string `json:"last_password_change,omitempty"`
	ActiveSessions           int     `json:"active_sessions"`
}
