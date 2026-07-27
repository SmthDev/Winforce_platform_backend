package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecodearcher/limen"
)


const ContextUserKey = "user"




func RequireAuth(instance *limen.Limen) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := instance.GetSession(c.Request)
		if err != nil {
			Logger(c).Error("DEBUG raw session error", slog.String("error_text", err.Error()), slog.String("error_type", fmt.Sprintf("%T", err)))
			if errors.Is(err, limen.ErrSessionNotFound) || errors.Is(err, limen.ErrSessionExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
				return
			}

			Logger(c).Error("session validation failed", slog.Any("error", err))
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"message": "service unavailable"})
			return
		}

		c.Set(ContextUserKey, session.User)
		c.Next()
	}
}


func CurrentUser(c *gin.Context) (*limen.User, bool) {
	value, exists := c.Get(ContextUserKey)
	if !exists {
		return nil, false
	}
	user, ok := value.(*limen.User)
	return user, ok
}
