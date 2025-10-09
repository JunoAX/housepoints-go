package handlers

import (
	"net/http"
	"strings"

	"github.com/JunoAX/housepoints-go/internal/auth"
	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token    string    `json:"token"`
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	IsParent bool      `json:"is_parent"`
	FamilyID uuid.UUID `json:"family_id"`
}

// Login authenticates a user and returns a JWT token
func Login(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := middleware.GetFamilyDB(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
			return
		}

		family, ok := middleware.GetFamily(c)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Family context required"})
			return
		}

		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Normalize username to lowercase
		username := strings.ToLower(strings.TrimSpace(req.Username))

		// Query user from family database
		query := `
			SELECT id, username, password_hash, is_parent, login_enabled
			FROM users
			WHERE LOWER(username) = $1
		`

		var userID uuid.UUID
		var dbUsername string
		var passwordHash *string
		var isParent bool
		var loginEnabled bool

		err := db.QueryRow(c.Request.Context(), query, username).Scan(
			&userID, &dbUsername, &passwordHash, &isParent, &loginEnabled,
		)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		// Check if login is enabled
		if !loginEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "Login is disabled for this user"})
			return
		}

		// Check if password_hash exists
		if passwordHash == nil || *passwordHash == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Password authentication not configured for this user"})
			return
		}

		// Verify password
		err = bcrypt.CompareHashAndPassword([]byte(*passwordHash), []byte(req.Password))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		// Generate JWT token
		token, err := jwtService.GenerateToken(userID, family.ID, dbUsername, isParent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Return token and user info
		c.JSON(http.StatusOK, LoginResponse{
			Token:    token,
			UserID:   userID,
			Username: dbUsername,
			IsParent: isParent,
			FamilyID: family.ID,
		})
	}
}
