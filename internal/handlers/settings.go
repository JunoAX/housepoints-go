package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
)

// GetSettings returns all system settings
func GetSettings(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	query := `
		SELECT setting_key, setting_value, setting_type
		FROM system_settings
		ORDER BY setting_key
	`

	rows, err := db.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query settings", "details": err.Error()})
		return
	}
	defer rows.Close()

	settings := make(models.SettingsResponse)
	for rows.Next() {
		var key, value, dataType string

		err := rows.Scan(&key, &value, &dataType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse setting", "details": err.Error()})
			return
		}

		// Convert value based on data type
		var convertedValue interface{}
		switch dataType {
		case "int":
			convertedValue, _ = strconv.Atoi(value)
		case "float":
			convertedValue, _ = strconv.ParseFloat(value, 64)
		case "bool":
			convertedValue = value == "true" || value == "1" || value == "yes"
		case "dict", "list":
			json.Unmarshal([]byte(value), &convertedValue)
		default:
			convertedValue = value
		}

		settings[key] = convertedValue
	}

	c.JSON(http.StatusOK, settings)
}

// GetSetting returns a specific system setting
func GetSetting(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	key := c.Param("key")

	query := `
		SELECT setting_value, setting_type
		FROM system_settings
		WHERE setting_key = $1
	`

	var value, dataType string
	err := db.QueryRow(c.Request.Context(), query, key).Scan(&value, &dataType)
	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Setting not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query setting", "details": err.Error()})
		}
		return
	}

	// Convert value based on data type
	var convertedValue interface{}
	switch dataType {
	case "int":
		convertedValue, _ = strconv.Atoi(value)
	case "float":
		convertedValue, _ = strconv.ParseFloat(value, 64)
	case "bool":
		convertedValue = value == "true" || value == "1" || value == "yes"
	case "dict", "list":
		json.Unmarshal([]byte(value), &convertedValue)
	default:
		convertedValue = value
	}

	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"value": convertedValue,
	})
}

// UpdateSetting updates a specific system setting (parent only)
func UpdateSetting(c *gin.Context) {
	db, ok := middleware.GetFamilyDB(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Check if user is a parent
	isParent, _ := middleware.GetAuthIsParent(c)
	if !isParent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can update settings"})
		return
	}

	userID, _ := middleware.GetAuthUserID(c)
	key := c.Param("key")

	var req models.SettingUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Check if setting exists
	var currentValue, dataType string
	err := db.QueryRow(c.Request.Context(),
		"SELECT setting_value, setting_type FROM system_settings WHERE setting_key = $1",
		key,
	).Scan(&currentValue, &dataType)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Setting not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query setting", "details": err.Error()})
		}
		return
	}

	// Convert value to string based on data type
	var stringValue string
	switch dataType {
	case "int":
		// Accept int or float
		switch v := req.Value.(type) {
		case float64:
			stringValue = strconv.Itoa(int(v))
		case int:
			stringValue = strconv.Itoa(v)
		default:
			stringValue = fmt.Sprintf("%v", v)
		}
	case "float":
		switch v := req.Value.(type) {
		case float64:
			stringValue = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			stringValue = fmt.Sprintf("%v", v)
		}
	case "bool":
		switch v := req.Value.(type) {
		case bool:
			stringValue = strconv.FormatBool(v)
		default:
			stringValue = fmt.Sprintf("%v", v)
		}
	case "dict", "list":
		bytes, err := json.Marshal(req.Value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON value"})
			return
		}
		stringValue = string(bytes)
	default:
		stringValue = fmt.Sprintf("%v", req.Value)
	}

	// Update the setting
	_, err = db.Exec(c.Request.Context(), `
		UPDATE system_settings
		SET setting_value = $1, updated_at = NOW(), updated_by = $2
		WHERE setting_key = $3
	`, stringValue, userID, key)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":     key,
		"value":   req.Value,
		"message": "Setting updated successfully",
	})
}
