package handlers

import (
	"net/http"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
)

// ListChores returns all active chores for the family
func ListChores(c *gin.Context) {
	// Get family database connection from context
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	family, _ := middleware.GetFamily(c)

	// Query chores from family database
	query := `
		SELECT
			id, name, category, base_points, estimated_minutes,
			difficulty, icon, active, assignment_type
		FROM chores
		WHERE is_active = true
		ORDER BY category, name
	`

	rows, err := db.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query chores",
		})
		return
	}
	defer rows.Close()

	chores := []models.ChoreListResponse{}
	for rows.Next() {
		var chore models.ChoreListResponse
		err := rows.Scan(
			&chore.ID,
			&chore.Name,
			&chore.Category,
			&chore.BasePoints,
			&chore.EstimatedMinutes,
			&chore.Difficulty,
			&chore.Icon,
			&chore.Active,
			&chore.AssignmentType,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to parse chore data",
			})
			return
		}
		chores = append(chores, chore)
	}

	c.JSON(http.StatusOK, gin.H{
		"family_id":   family.ID,
		"family_name": family.Name,
		"chores":      chores,
		"count":       len(chores),
	})
}
