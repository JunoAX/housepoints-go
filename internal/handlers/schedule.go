package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
)

// GetFamilySchedule returns the family schedule for a date range
func GetFamilySchedule(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Get query parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	days := c.DefaultQuery("days", "30")

	// If no start_date provided, use today
	if startDate == "" {
		startDate = time.Now().Format("2006-01-02")
	}

	// If no end_date but days is provided, calculate end_date
	if endDate == "" && days != "" {
		start, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use YYYY-MM-DD"})
			return
		}
		daysInt := 30
		if _, err := time.ParseDuration(days + "h"); err == nil {
			// Parse days as integer
			var d int
			if _, err := fmt.Sscanf(days, "%d", &d); err == nil {
				daysInt = d
			}
		}
		endDate = start.AddDate(0, 0, daysInt).Format("2006-01-02")
	}

	query := `
		SELECT
			id,
			date,
			weekday,
			day_type,
			presence_data,
			kids_present,
			total_kids_present,
			is_john_weekend,
			notes,
			transition_time,
			transition_type,
			custody_priority,
			created_at,
			updated_at
		FROM family_schedule
		WHERE date >= $1 AND date <= $2
		ORDER BY date ASC
	`

	rows, err := db.Query(c.Request.Context(), query, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query schedule", "details": err.Error()})
		return
	}
	defer rows.Close()

	schedule := []models.FamilyScheduleEntry{}

	for rows.Next() {
		var entry models.FamilyScheduleEntry
		var presenceDataJSON []byte
		var dateTime time.Time
		var transitionTime *time.Time

		err := rows.Scan(
			&entry.ID,
			&dateTime,
			&entry.Weekday,
			&entry.DayType,
			&presenceDataJSON,
			&entry.KidsPresent,
			&entry.TotalKidsPresent,
			&entry.IsJohnWeekend,
			&entry.Notes,
			&transitionTime,
			&entry.TransitionType,
			&entry.CustodyPriority,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse schedule data", "details": err.Error()})
			return
		}

		// Format date as string
		entry.Date = dateTime.Format("2006-01-02")

		// Format transition time if present
		if transitionTime != nil {
			timeStr := transitionTime.Format("15:04:05")
			entry.TransitionTime = &timeStr
		}

		// Parse presence_data JSON
		if len(presenceDataJSON) > 0 {
			if err := json.Unmarshal(presenceDataJSON, &entry.PresenceData); err != nil {
				entry.PresenceData = make(map[string]interface{})
			}
		} else {
			entry.PresenceData = make(map[string]interface{})
		}

		// Initialize empty array if nil
		if entry.KidsPresent == nil {
			entry.KidsPresent = []string{}
		}

		schedule = append(schedule, entry)
	}

	c.JSON(http.StatusOK, models.FamilyScheduleResponse{
		StartDate: startDate,
		EndDate:   endDate,
		Schedule:  schedule,
		TotalDays: len(schedule),
	})
}
