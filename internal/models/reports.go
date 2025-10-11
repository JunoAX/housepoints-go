package models

import "time"

// WeeklySummaryResponse represents the weekly summary report
type WeeklySummaryResponse struct {
	WeekStart string                 `json:"week_start"`
	WeekEnd   string                 `json:"week_end"`
	Summary   WeeklySummary          `json:"summary"`
	Children  []ChildWeeklyPerformance `json:"children"`
}

// WeeklySummary contains aggregate statistics for the week
type WeeklySummary struct {
	TotalAssignments int     `json:"total_assignments"`
	Completed        int     `json:"completed"`
	Verified         int     `json:"verified"`
	Overdue          int     `json:"overdue"`
	TotalPoints      int     `json:"total_points"`
	ActiveChildren   int     `json:"active_children"`
	CompletionRate   float64 `json:"completion_rate"`
}

// ChildWeeklyPerformance represents performance for a single child
type ChildWeeklyPerformance struct {
	ID               string  `json:"id"`
	Username         string  `json:"username"`
	DisplayName      string  `json:"display_name"`
	ColorTheme       string  `json:"color_theme"`
	TotalAssignments int     `json:"total_assignments"`
	Completed        int     `json:"completed"`
	Verified         int     `json:"verified"`
	PointsEarned     int     `json:"points_earned"`
	CompletionRate   float64 `json:"completion_rate"`
}

// ChildPerformanceResponse represents detailed performance for a specific child
type ChildPerformanceResponse struct {
	ChildID        string          `json:"child_id"`
	Period         DateRange       `json:"period"`
	Stats          PerformanceStats `json:"stats"`
	DailyBreakdown []DailyStats    `json:"daily_breakdown"`
}

// DateRange represents a date range
type DateRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// PerformanceStats represents overall performance statistics
type PerformanceStats struct {
	TotalAssignments int `json:"total_assignments"`
	Completed        int `json:"completed"`
	Verified         int `json:"verified"`
	Overdue          int `json:"overdue"`
	TotalPoints      int `json:"total_points"`
}

// DailyStats represents daily performance breakdown
type DailyStats struct {
	Date        string `json:"date"`
	Assignments int    `json:"assignments"`
	Completed   int    `json:"completed"`
	Points      int    `json:"points"`
}

// CategoryBreakdownResponse represents breakdown by chore category
type CategoryBreakdownResponse struct {
	Period     DateRange          `json:"period"`
	Categories []CategoryStats    `json:"categories"`
}

// CategoryStats represents statistics for a single category
type CategoryStats struct {
	Category       string  `json:"category"`
	TotalAssignments int   `json:"total_assignments"`
	Completed      int     `json:"completed"`
	CompletionRate float64 `json:"completion_rate"`
	PointsEarned   int     `json:"points_earned"`
}

// PerformanceTrendsResponse represents performance trend data for charts
type PerformanceTrendsResponse struct {
	Trends           []TrendDataPoint      `json:"trends"`
	ChildPerformance []ChildTrendPerformance `json:"childPerformance"`
}

// TrendDataPoint represents a single data point in the trend
type TrendDataPoint struct {
	Date      string `json:"date"`
	Completed int    `json:"completed"`
	Points    int    `json:"points"`
	Verified  int    `json:"verified"`
}

// ChildTrendPerformance represents child performance in trend view
type ChildTrendPerformance struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Completion      string `json:"completion"`
	AvgPointsPerDay int    `json:"avgPointsPerDay"`
	Streak          string `json:"streak"`
	Trend           string `json:"trend"` // "up", "down", "neutral"
}

// PendingRedemptionsResponse represents pending redemptions
type PendingRedemptionsResponse struct {
	Redemptions []PendingRedemption `json:"redemptions"`
}

// PendingRedemption represents a single pending redemption
type PendingRedemption struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	ChildName         string     `json:"child_name"`
	RewardName        string     `json:"reward_name"`
	Points            int        `json:"points"`
	RequestedDate     *time.Time `json:"requested_date,omitempty"`
	Status            string     `json:"status"`
	Notes             *string    `json:"notes,omitempty"`
	ChildTotalPoints  int        `json:"child_total_points"`
}

// RedemptionActionRequest represents approve/reject request body
type RedemptionActionRequest struct {
	Notes *string `json:"notes,omitempty"`
}
