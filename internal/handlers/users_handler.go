package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"platform/backend/internal/middleware"
	"platform/backend/internal/models"
)

type UserLister interface {
	ListUsers(ctx context.Context) ([]models.Profile, error)
}

func ListUsers(userLister UserLister) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := userLister.ListUsers(c.Request.Context())
		if err != nil {
			middleware.Logger(c).Error("list users failed", slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list users"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}
