package avatar_repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) GetAvatarLink(ctx context.Context, userID any) (string, bool, error) {
	var link string
	err := r.pool.QueryRow(ctx, `SELECT avatar_link FROM avatars WHERE user_id = $1`, userID).Scan(&link)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get avatar link: %w", err)
	}
	return link, true, nil
}

func (r *Repo) UpsertAvatarLink(ctx context.Context, userID any, avatarLink string) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO avatars (user_id, avatar_link)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET avatar_link = EXCLUDED.avatar_link, add_date_time = now()`,
		userID, avatarLink,
	)
	if err != nil {
		return fmt.Errorf("upsert avatar link: %w", err)
	}
	return nil
}
