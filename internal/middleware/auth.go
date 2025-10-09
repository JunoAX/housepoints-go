package middleware

import (
	"net/http"
	"strings"

	"github.com/JunoAX/housepoints-go/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	authUserKey     = "auth_user_id"
	authUsernameKey = "auth_username"
	authIsParentKey = "auth_is_parent"
)

// RequireAuth validates JWT token and sets user context
func RequireAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check for Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format. Use: Bearer <token>"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			c.Abort()
			return
		}

		// Verify token belongs to the correct family
		family, exists := GetFamily(c)
		if exists && family.ID != claims.FamilyID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Token does not belong to this family"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set(authUserKey, claims.UserID)
		c.Set(authUsernameKey, claims.Username)
		c.Set(authIsParentKey, claims.IsParent)

		c.Next()
	}
}

// RequireParent ensures the authenticated user is a parent
func RequireParent() gin.HandlerFunc {
	return func(c *gin.Context) {
		isParent, exists := c.Get(authIsParentKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		if !isParent.(bool) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Parent access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetAuthUserID retrieves the authenticated user ID from context
func GetAuthUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get(authUserKey)
	if !exists {
		return uuid.Nil, false
	}
	return userID.(uuid.UUID), true
}

// GetAuthUsername retrieves the authenticated username from context
func GetAuthUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get(authUsernameKey)
	if !exists {
		return "", false
	}
	return username.(string), true
}

// GetAuthIsParent retrieves whether the authenticated user is a parent
func GetAuthIsParent(c *gin.Context) (bool, bool) {
	isParent, exists := c.Get(authIsParentKey)
	if !exists {
		return false, false
	}
	return isParent.(bool), true
}
