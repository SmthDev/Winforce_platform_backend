package qr_repo

import (
	"context"
	"fmt"

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

func (r *Repo) CreateQRCode(ctx context.Context, userID any, targetLink, qrLink string) (models.QRCode, error) {
	var qr models.QRCode
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO qr_codes (user_id, target_link, qr_link)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, target_link, qr_link, add_date_time`,
		userID, targetLink, qrLink,
	).Scan(&qr.ID, &qr.UserID, &qr.TargetLink, &qr.QRLink, &qr.AddDateTime)
	if err != nil {
		return models.QRCode{}, fmt.Errorf("create qr code: %w", err)
	}
	return qr, nil
}

func (r *Repo) ListQRCodes(ctx context.Context) ([]models.QRCode, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, user_id, target_link, qr_link, add_date_time
		 FROM qr_codes
		 ORDER BY add_date_time DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list qr codes: %w", err)
	}

	codes, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.QRCode, error) {
		var qr models.QRCode
		err := row.Scan(&qr.ID, &qr.UserID, &qr.TargetLink, &qr.QRLink, &qr.AddDateTime)
		return qr, err
	})
	if err != nil {
		return nil, fmt.Errorf("list qr codes: %w", err)
	}
	return codes, nil
}
