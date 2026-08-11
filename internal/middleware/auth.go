package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecodearcher/limen"
)


const ContextUserKey = "user"

type AdminChecker interface {
	IsAdmin(ctx context.Context, userID any) (isAdmin bool, found bool, err error)
}




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

func RequireAdmin(access AdminChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		isAdmin, found, err := access.IsAdmin(c.Request.Context(), user.ID)
		if err != nil {
			Logger(c).Error("check admin access failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "failed to check permissions"})
			return
		}
		if !found {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}
		if !isAdmin {
			Logger(c).Warn("admin route access denied", slog.Any("user_id", user.ID))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			return
		}

		c.Next()
	}
}
