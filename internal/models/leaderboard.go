package models

import "github.com/google/uuid"

// LeaderboardEntry represents a user's position on the leaderboard
type LeaderboardEntry struct {
	Rank                int       `json:"rank"`
	UserID              uuid.UUID `json:"user_id"`
	Username            string    `json:"username"`
	DisplayName         string    `json:"display_name"`
	AvatarURL           *string   `json:"avatar_url,omitempty"`
	ColorTheme          string    `json:"color_theme"`
	Points              int       `json:"points"`
	WeeklyPoints        int       `json:"weekly_points"`
	TotalPoints         int       `json:"total_points"`
	LifetimePoints      int       `json:"lifetime_points_earned"`
	Level               int       `json:"level"`
	ChoresCompleted     int       `json:"chores_completed"`
	StreakDays          int       `json:"streak_days"`
}

// LeaderboardResponse is the API response for leaderboards
type LeaderboardResponse struct {
	Period      string             `json:"period"`
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
	TotalUsers  int                `json:"total_users"`
}
