package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"platform/backend/internal/models"
)

const maxOpponentLen = 255

var ErrInvalidGame = errors.New("invalid game")

type GameRepository interface {
	CreateGame(ctx context.Context, playedOn time.Time, opponent string) (models.Game, error)
	ListGames(ctx context.Context, limit, offset int) ([]models.Game, error)
}

type GameService struct {
	repo GameRepository
}

func NewGame(repo GameRepository) *GameService {
	return &GameService{repo: repo}
}

func (s *GameService) CreateGame(ctx context.Context, playedOn, opponent string) (models.Game, error) {
	date, err := time.Parse(models.GameDateLayout, strings.TrimSpace(playedOn))
	if err != nil {
		return models.Game{}, fmt.Errorf("%w: played_on must be YYYY-MM-DD", ErrInvalidGame)
	}
	opponent = strings.TrimSpace(opponent)
	if opponent == "" || utf8.RuneCountInString(opponent) > maxOpponentLen {
		return models.Game{}, fmt.Errorf("%w: opponent must be 1..%d characters", ErrInvalidGame, maxOpponentLen)
	}
	return s.repo.CreateGame(ctx, date, opponent)
}

func (s *GameService) ListGames(ctx context.Context, limit, offset int) ([]models.Game, error) {
	return s.repo.ListGames(ctx, clampLimit(limit), clampOffset(offset))
}
