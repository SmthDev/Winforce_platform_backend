package user_repo

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"platform/backend/internal/models"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) UpdateName(ctx context.Context, userID any, firstName, lastName string) error {
	tag, err := r.pool.Exec(
		ctx,
		`UPDATE users SET first_name = $1, last_name = $2, updated_at = now() WHERE id = $3`,
		firstName, lastName, userID,
	)
	if err != nil {
		return fmt.Errorf("update user name: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update user name: no user with id %v", userID)
	}
	return nil
}



func (r *Repo) GetUserProfile(ctx context.Context, userID any) (models.Profile, error) {
	var (
		id        int64
		email     string
		firstName *string
		lastName  *string
	)

	row := r.pool.QueryRow(ctx, `SELECT id, email, first_name, last_name FROM users WHERE id = $1`, userID)
	if err := row.Scan(&id, &email, &firstName, &lastName); err != nil {
		return models.Profile{}, fmt.Errorf("get user profile: %w", err)
	}

	profile := models.Profile{
		ID:    strconv.FormatInt(id, 10),
		Email: email,
	}
	if firstName != nil {
		profile.FirstName = *firstName
	}
	if lastName != nil {
		profile.LastName = *lastName
	}
	return profile, nil
}	