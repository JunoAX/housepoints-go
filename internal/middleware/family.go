package middleware

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FamilyContextKey is the key for storing family context
type contextKey string

const (
	FamilyContextKey contextKey = "family"
	FamilyIDKey      contextKey = "family_id"
	FamilySlugKey    contextKey = "family_slug"
	FamilyDBKey      contextKey = "family_db"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// FamilyDBProvider interface for getting family database connections
type FamilyDBProvider interface {
	GetFamilyDBBySlug(ctx context.Context, slug string) (*pgxpool.Pool, *models.Family, error)
}

// ExtractFamilySlug extracts the family slug from subdomain
// Examples:
//   - gamull.housepoints.ai → "gamull"
//   - smith-nyc.housepoints.ai → "smith-nyc"
//   - staging.housepoints.ai → "staging" (would need to be handled specially)
//   - api.housepoints.ai → "" (no family, API-only)
func ExtractFamilySlug(host string, baseDomain string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	host = strings.ToLower(host)
	baseDomain = strings.ToLower(baseDomain)

	// If host equals base domain or www, no slug
	if host == baseDomain || host == "www."+baseDomain {
		return ""
	}

	// Check if host ends with base domain
	if !strings.HasSuffix(host, "."+baseDomain) {
		return ""
	}

	// Extract subdomain
	slug := strings.TrimSuffix(host, "."+baseDomain)

	// Reserved subdomains that are not family slugs
	reserved := map[string]bool{
		"api":     true,
		"www":     true,
		"app":     true,
		"admin":   true,
		"staging": true,
		"dev":     true,
	}

	if reserved[slug] {
		return ""
	}

	return slug
}

// FamilyMiddleware extracts family slug from subdomain and loads family context + DB connection
func FamilyMiddleware(dbProvider FamilyDBProvider, baseDomain string) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		slug := ExtractFamilySlug(host, baseDomain)

		// If no slug, continue without family context (API-only routes)
		if slug == "" {
			c.Next()
			return
		}

		// Validate slug format
		if !ValidateSlug(slug) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid family identifier",
			})
			c.Abort()
			return
		}

		// Look up family and get database connection
		familyDB, family, err := dbProvider.GetFamilyDBBySlug(c.Request.Context(), slug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Family not found",
				"slug":  slug,
			})
			c.Abort()
			return
		}

		// Check if family is active
		if family.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Family account is inactive",
			})
			c.Abort()
			return
		}

		// Store family info and DB connection in context
		c.Set(string(FamilyIDKey), family.ID)
		c.Set(string(FamilySlugKey), family.Slug)
		c.Set(string(FamilyContextKey), family)
		c.Set(string(FamilyDBKey), familyDB)

		c.Next()
	}
}

// RequireFamily ensures a family context exists
func RequireFamily() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get(string(FamilyIDKey))
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Family context required. Please access via your family subdomain (e.g., yourfamily.housepoints.ai)",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetFamilyID retrieves family ID from context
func GetFamilyID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get(string(FamilyIDKey))
	if !exists {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}

// GetFamilySlug retrieves family slug from context
func GetFamilySlug(c *gin.Context) (string, bool) {
	val, exists := c.Get(string(FamilySlugKey))
	if !exists {
		return "", false
	}
	slug, ok := val.(string)
	return slug, ok
}

// GetFamilyDB retrieves family database connection from context
func GetFamilyDB(c *gin.Context) (*pgxpool.Pool, bool) {
	val, exists := c.Get(string(FamilyDBKey))
	if !exists {
		return nil, false
	}
	db, ok := val.(*pgxpool.Pool)
	return db, ok
}

// GetFamily retrieves full family object from context
func GetFamily(c *gin.Context) (*models.Family, bool) {
	val, exists := c.Get(string(FamilyContextKey))
	if !exists {
		return nil, false
	}
	family, ok := val.(*models.Family)
	return family, ok
}

// ValidateSlug checks if a slug is valid
// Rules:
//   - 3-50 characters
//   - Lowercase letters, numbers, hyphens only
//   - Must start and end with letter or number
//   - Cannot have consecutive hyphens
func ValidateSlug(slug string) bool {
	if len(slug) < 3 || len(slug) > 50 {
		return false
	}

	if !slugRegex.MatchString(slug) {
		return false
	}

	// No consecutive hyphens
	if strings.Contains(slug, "--") {
		return false
	}

	return true
}
