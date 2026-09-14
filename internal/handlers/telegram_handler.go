package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"platform/backend/internal/middleware"
	"platform/backend/internal/models"
	"platform/backend/internal/service"
)

type TelegramService interface {
	LinkAccount(ctx context.Context, userID any, payload models.TelegramAuthPayload) (models.TelegramAccount, error)
	GetLink(ctx context.Context, userID any) (models.TelegramAccount, error)
	Unlink(ctx context.Context, userID any) error
}

func LinkTelegram(telegramService TelegramService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		var payload models.TelegramAuthPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid telegram auth payload"})
			return
		}

		acc, err := telegramService.LinkAccount(c.Request.Context(), user.ID, payload)
		if err != nil {
			writeTelegramError(c, err, user.ID)
			return
		}

		c.JSON(http.StatusOK, acc)
	}
}

func GetTelegramLink(telegramService TelegramService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		acc, err := telegramService.GetLink(c.Request.Context(), user.ID)
		if err != nil {
			writeTelegramError(c, err, user.ID)
			return
		}

		c.JSON(http.StatusOK, acc)
	}
}

func UnlinkTelegram(telegramService TelegramService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		if err := telegramService.Unlink(c.Request.Context(), user.ID); err != nil {
			writeTelegramError(c, err, user.ID)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "telegram account unlinked"})
	}
}

func writeTelegramError(c *gin.Context, err error, userID any) {
	switch {
	case errors.Is(err, service.ErrTelegramNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "telegram integration is not configured"})
	case errors.Is(err, service.ErrInvalidTelegramSignature):
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid telegram signature"})
	case errors.Is(err, service.ErrTelegramAuthExpired):
		c.JSON(http.StatusBadRequest, gin.H{"message": "telegram auth data expired, please try again"})
	case errors.Is(err, service.ErrTelegramAccountTaken):
		c.JSON(http.StatusConflict, gin.H{"message": "this telegram account is already linked to another user"})
	case errors.Is(err, service.ErrTelegramNotLinked):
		c.JSON(http.StatusNotFound, gin.H{"message": "telegram account is not linked"})
	default:
		middleware.Logger(c).Error("telegram operation failed", slog.Any("error", err), slog.Any("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to process telegram request"})
	}
}
