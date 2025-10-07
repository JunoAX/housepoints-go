package middleware

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FamilyContextKey is the key for storing family context
type contextKey string

const (
	FamilyContextKey contextKey = "family"
	FamilyIDKey      contextKey = "family_id"
	FamilySlugKey    contextKey = "family_slug"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// FamilyResolver interface for looking up families by slug
type FamilyResolver interface {
	GetFamilyBySlug(ctx context.Context, slug string) (*FamilyInfo, error)
}

// FamilyInfo contains basic family information for middleware
type FamilyInfo struct {
	ID     uuid.UUID
	Slug   string
	Name   string
	Active bool
	Plan   string
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

// FamilyMiddleware extracts family slug from subdomain and loads family context
func FamilyMiddleware(resolver FamilyResolver, baseDomain string) gin.HandlerFunc {
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

		// Look up family
		family, err := resolver.GetFamilyBySlug(c.Request.Context(), slug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Family not found",
				"slug":  slug,
			})
			c.Abort()
			return
		}

		// Check if family is active
		if !family.Active {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Family account is inactive",
			})
			c.Abort()
			return
		}

		// Store family info in context
		c.Set(string(FamilyIDKey), family.ID)
		c.Set(string(FamilySlugKey), family.Slug)
		c.Set(string(FamilyContextKey), family)

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
