package user_repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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



func (r *Repo) IsAdmin(ctx context.Context, userID any) (bool, bool, error) {
	var isAdmin bool
	err := r.pool.QueryRow(ctx, `SELECT role = 'admin' FROM users WHERE id = $1`, userID).Scan(&isAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("check user role: %w", err)
	}
	return isAdmin, true, nil
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
func (r *Repo) ListUsers(ctx context.Context) ([]models.Profile, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, email, first_name, last_name FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Profile, error) {
		var (
			id        int64
			profile   models.Profile
			firstName *string
			lastName  *string
		)
		if err := row.Scan(&id, &profile.Email, &firstName, &lastName); err != nil {
			return models.Profile{}, err
		}
		profile.ID = strconv.FormatInt(id, 10)
		if firstName != nil {
			profile.FirstName = *firstName
		}
		if lastName != nil {
			profile.LastName = *lastName
		}
		return profile, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}
