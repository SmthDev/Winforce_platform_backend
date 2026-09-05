package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

const defaultAllowedOrigin = "http://localhost:5173"


func CORS(allowedOrigins ...string) gin.HandlerFunc {
	origins := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		origins = []string{defaultAllowedOrigin}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if !slices.Contains(origins, origin) {
			origin = origins[0]
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
