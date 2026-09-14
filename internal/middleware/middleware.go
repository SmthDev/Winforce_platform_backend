package middleware

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

const defaultAllowedOrigin = "http://localhost:5173"



func CORS(allowedOrigins ...string) gin.HandlerFunc {
	origins := make([]string, 0, len(allowedOrigins))
	patterns := make([]*regexp.Regexp, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		switch {
		case origin == "":
		case strings.Contains(origin, "*"):
			patterns = append(patterns, compileOriginPattern(origin))
		default:
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		origins = []string{defaultAllowedOrigin}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if !originAllowed(origin, origins, patterns) {
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

func originAllowed(origin string, origins []string, patterns []*regexp.Regexp) bool {
	if origin == "" {
		return false
	}
	if slices.Contains(origins, origin) {
		return true
	}
	return slices.ContainsFunc(patterns, func(pattern *regexp.Regexp) bool {
		return pattern.MatchString(origin)
	})
}


func compileOriginPattern(pattern string) *regexp.Regexp {
	parts := strings.Split(pattern, "*")
	for i, part := range parts {
		parts[i] = regexp.QuoteMeta(part)
	}
	return regexp.MustCompile("^" + strings.Join(parts, "[^/]*") + "$")
}
