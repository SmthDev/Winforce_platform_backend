package game_repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"platform/backend/internal/models"
)

var ErrGameNotFound = errors.New("game not found")

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

func (r *Repo) GetGame(ctx context.Context, id int64) (models.Game, error) {
	game, err := scanGame(r.pool.QueryRow(
		ctx,
		`SELECT id, played_on, opponent, charged_at, created_at FROM games WHERE id = $1`,
		id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Game{}, ErrGameNotFound
	}
	if err != nil {
		return models.Game{}, fmt.Errorf("get game: %w", err)
	}
	return game, nil
}

func (r *Repo) ListGamePlayers(ctx context.Context, gameID int64) ([]models.GamePlayer, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT u.id, u.email, u.first_name, u.last_name, tg.username,
			-bt.amount_minor, bt.currency, bt.created_at
		 FROM balance_transactions bt
		 JOIN users u ON u.id = bt.user_id
		 LEFT JOIN LATERAL (
			SELECT ta.username FROM telegram_accounts ta WHERE ta.user_id = u.id ORDER BY ta.linked_at DESC LIMIT 1
		 ) tg ON true
		 WHERE bt.game_id = $1 AND bt.kind = 'game_charge'
		 ORDER BY u.first_name NULLS LAST, u.last_name NULLS LAST, u.email`,
		gameID,
	)
	if err != nil {
		return nil, fmt.Errorf("list game players: %w", err)
	}

	players, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.GamePlayer, error) {
		var (
			p         models.GamePlayer
			firstName *string
			lastName  *string
			username  *string
		)
		if err := row.Scan(&p.UserID, &p.Email, &firstName, &lastName, &username,
			&p.AmountMinor, &p.Currency, &p.ChargedAt); err != nil {
			return models.GamePlayer{}, err
		}
		if firstName != nil {
			p.FirstName = *firstName
		}
		if lastName != nil {
			p.LastName = *lastName
		}
		if username != nil {
			p.TelegramUsername = *username
		}
		p.Amount = models.FormatMinor(p.AmountMinor)
		return p, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list game players: %w", err)
	}
	return players, nil
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
