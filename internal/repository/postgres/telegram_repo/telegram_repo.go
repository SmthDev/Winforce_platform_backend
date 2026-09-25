package telegram_repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"platform/backend/internal/models"
)

const (
	uniqueViolationCode = "23505"

	telegramIDConstraint = "idx_telegram_accounts_telegram_id"
)

var ErrTelegramAccountTaken = errors.New("telegram account already linked to another user")

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) LinkAccount(ctx context.Context, userID any, telegramID int64, username, firstName, lastName, photoURL string) (models.TelegramAccount, error) {
	var (
		acc                                                 models.TelegramAccount
		usernamePtr, firstNamePtr, lastNamePtr, photoURLPtr *string
	)

	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO telegram_accounts (user_id, telegram_id, username, first_name, last_name, photo_url)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id) DO UPDATE
		 SET telegram_id = EXCLUDED.telegram_id,
		     username = EXCLUDED.username,
		     first_name = EXCLUDED.first_name,
		     last_name = EXCLUDED.last_name,
		     photo_url = EXCLUDED.photo_url,
		     linked_at = now()
		 RETURNING id, user_id, telegram_id, username, first_name, last_name, photo_url, linked_at`,
		userID, telegramID, nullableString(username), nullableString(firstName), nullableString(lastName), nullableString(photoURL),
	).Scan(&acc.ID, &acc.UserID, &acc.TelegramID, &usernamePtr, &firstNamePtr, &lastNamePtr, &photoURLPtr, &acc.LinkedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode && pgErr.ConstraintName == telegramIDConstraint {
			return models.TelegramAccount{}, ErrTelegramAccountTaken
		}
		return models.TelegramAccount{}, fmt.Errorf("link telegram account: %w", err)
	}

	acc.Username = derefString(usernamePtr)
	acc.FirstName = derefString(firstNamePtr)
	acc.LastName = derefString(lastNamePtr)
	acc.PhotoURL = derefString(photoURLPtr)
	return acc, nil
}

func (r *Repo) GetByUserID(ctx context.Context, userID any) (models.TelegramAccount, bool, error) {
	var (
		acc                                                 models.TelegramAccount
		usernamePtr, firstNamePtr, lastNamePtr, photoURLPtr *string
	)

	err := r.pool.QueryRow(
		ctx,
		`SELECT id, user_id, telegram_id, username, first_name, last_name, photo_url, linked_at
		 FROM telegram_accounts
		 WHERE user_id = $1`,
		userID,
	).Scan(&acc.ID, &acc.UserID, &acc.TelegramID, &usernamePtr, &firstNamePtr, &lastNamePtr, &photoURLPtr, &acc.LinkedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.TelegramAccount{}, false, nil
	}
	if err != nil {
		return models.TelegramAccount{}, false, fmt.Errorf("get telegram account: %w", err)
	}

	acc.Username = derefString(usernamePtr)
	acc.FirstName = derefString(firstNamePtr)
	acc.LastName = derefString(lastNamePtr)
	acc.PhotoURL = derefString(photoURLPtr)
	return acc, true, nil
}

func (r *Repo) Unlink(ctx context.Context, userID any) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM telegram_accounts WHERE user_id = $1`, userID)
	if err != nil {
		return false, fmt.Errorf("unlink telegram account: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}


func (r *Repo) ListBelowBalance(ctx context.Context, thresholdMinor int64, defaultCurrency string) ([]models.LowBalanceRecipient, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT t.user_id, t.telegram_id, COALESCE(b.amount_minor, 0), COALESCE(b.currency, $2)
		 FROM telegram_accounts t
		 LEFT JOIN balances b ON b.user_id = t.user_id
		 WHERE COALESCE(b.amount_minor, 0) < $1
		 ORDER BY t.user_id`,
		thresholdMinor, defaultCurrency,
	)
	if err != nil {
		return nil, fmt.Errorf("list low balance accounts: %w", err)
	}

	recipients, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.LowBalanceRecipient, error) {
		var rcp models.LowBalanceRecipient
		err := row.Scan(&rcp.UserID, &rcp.TelegramID, &rcp.AmountMinor, &rcp.Currency)
		return rcp, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan low balance accounts: %w", err)
	}
	return recipients, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
