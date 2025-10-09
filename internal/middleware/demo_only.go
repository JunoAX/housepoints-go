package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DemoOnly middleware restricts access to demo family only
// Use this for endpoints that aren't ready for production families yet
func DemoOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		slug, exists := GetFamilySlug(c)
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Family context required",
			})
			c.Abort()
			return
		}

		if slug != "demo" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "This endpoint is currently only available for the demo family at demo.housepoints.ai",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
