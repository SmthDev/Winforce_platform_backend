package game_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"platform/backend/internal/models"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) CreateGame(ctx context.Context, playedOn time.Time, opponent string) (models.Game, error) {
	row := r.pool.QueryRow(
		ctx,
		`INSERT INTO games (played_on, opponent)
		 VALUES ($1, $2)
		 RETURNING id, played_on, opponent, charged_at, created_at`,
		playedOn, opponent,
	)
	game, err := scanGame(row)
	if err != nil {
		return models.Game{}, fmt.Errorf("create game: %w", err)
	}
	return game, nil
}

func (r *Repo) ListGames(ctx context.Context, limit, offset int) ([]models.Game, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, played_on, opponent, charged_at, created_at
		 FROM games
		 ORDER BY played_on DESC, id DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}

	games, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Game, error) {
		return scanGame(row)
	})
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	return games, nil
}

func scanGame(row pgx.Row) (models.Game, error) {
	var (
		game     models.Game
		playedOn time.Time
	)
	if err := row.Scan(&game.ID, &playedOn, &game.Opponent, &game.ChargedAt, &game.CreatedAt); err != nil {
		return models.Game{}, err
	}
	game.PlayedOn = playedOn.Format(models.GameDateLayout)
	return game, nil
}
