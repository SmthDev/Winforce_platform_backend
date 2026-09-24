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

type GameService interface {
	CreateGame(ctx context.Context, playedOn, opponent string) (models.Game, error)
	ListGames(ctx context.Context, limit, offset int) ([]models.Game, error)
}

type createGameRequest struct {
	PlayedOn string `json:"played_on"`
	Opponent string `json:"opponent"`
}

func CreateGame(gameService GameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createGameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "played_on and opponent are required"})
			return
		}

		game, err := gameService.CreateGame(c.Request.Context(), req.PlayedOn, req.Opponent)
		if err != nil {
			if errors.Is(err, service.ErrInvalidGame) {
				c.JSON(http.StatusBadRequest, gin.H{"message": "played_on must be YYYY-MM-DD and opponent must be non-empty (max 255 characters)"})
				return
			}
			middleware.Logger(c).Error("create game failed", slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create game"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"game": game})
	}
}

func ListGames(gameService GameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset := paginationParams(c)

		games, err := gameService.ListGames(c.Request.Context(), limit, offset)
		if err != nil {
			middleware.Logger(c).Error("list games failed", slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list games"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"games": games})
	}
}
