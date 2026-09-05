package balance_repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"platform/backend/internal/models"
)

const (
	uniqueViolationCode = "23505"

	fileHashConstraint    = "idx_receipts_file_hash"
	checkNumberConstraint = "idx_receipts_check_number"

	receiptColumns = `id, user_id, status, file_object, check_number, amount_minor, currency,
		check_date, bank, card, payer, service, account, recipient, add_date_time`
)

var (

	ErrDuplicateFile = errors.New("receipt file already uploaded")
	ErrDuplicateCheckNumber = errors.New("receipt check number already used")
	ErrCurrencyMismatch = errors.New("receipt currency does not match balance currency")
	ErrReceiptNotFound = errors.New("receipt not found")
	ErrReceiptAlreadyReversed = errors.New("receipt already reversed")
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

type ReceiptInsert struct {
	FileObject  string
	FileHash    string
	AmountMinor int64
	Currency    string
	CheckNumber string
	CheckDate   string
	Bank        string
	Card        string
	Payer       string
	Service     string
	Account     string
	Recipient   string
	RawResponse []byte
}

func (r *Repo) GetBalance(ctx context.Context, userID any) (models.Balance, bool, error) {
	var (
		balance   models.Balance
		updatedAt time.Time
	)
	err := r.pool.QueryRow(
		ctx,
		`SELECT user_id, amount_minor, currency, updated_at FROM balances WHERE user_id = $1`,
		userID,
	).Scan(&balance.UserID, &balance.AmountMinor, &balance.Currency, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Balance{}, false, nil
	}
	if err != nil {
		return models.Balance{}, false, fmt.Errorf("get balance: %w", err)
	}

	balance.Amount = models.FormatMinor(balance.AmountMinor)
	balance.UpdatedAt = &updatedAt
	return balance, true, nil
}

func (r *Repo) UserExists(ctx context.Context, userID any) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}
	return exists, nil
}

func (r *Repo) ReceiptExistsByFileHash(ctx context.Context, fileHash string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM receipts WHERE file_hash = $1)`,
		fileHash,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check receipt file hash: %w", err)
	}
	return exists, nil
}


func (r *Repo) CreditReceipt(ctx context.Context, userID any, in ReceiptInsert) (models.Receipt, models.Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Receipt{}, models.Balance{}, fmt.Errorf("credit receipt: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := checkBalanceCurrency(ctx, tx, userID, in.Currency); err != nil {
		return models.Receipt{}, models.Balance{}, err
	}

	receipt, err := scanReceipt(tx.QueryRow(
		ctx,
		`INSERT INTO receipts (user_id, status, file_object, file_hash, check_number, amount_minor,
			currency, check_date, bank, card, payer, service, account, recipient, raw_response)
		 VALUES ($1, 'credited', $2, $3, NULLIF($4, ''), $5, $6, NULLIF($7, ''), NULLIF($8, ''),
			NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''), $14)
		 RETURNING `+receiptColumns,
		userID, in.FileObject, in.FileHash, in.CheckNumber, in.AmountMinor, in.Currency,
		in.CheckDate, in.Bank, in.Card, in.Payer, in.Service, in.Account, in.Recipient, in.RawResponse,
	))
	if err != nil {
		return models.Receipt{}, models.Balance{}, duplicateError(err)
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO balance_transactions (user_id, amount_minor, currency, kind, receipt_id)
		 VALUES ($1, $2, $3, 'receipt_credit', $4)`,
		userID, in.AmountMinor, in.Currency, receipt.ID,
	); err != nil {
		return models.Receipt{}, models.Balance{}, fmt.Errorf("credit receipt ledger: %w", err)
	}

	balance, err := applyBalanceDelta(ctx, tx, userID, in.AmountMinor, in.Currency)
	if err != nil {
		return models.Receipt{}, models.Balance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Receipt{}, models.Balance{}, fmt.Errorf("credit receipt: %w", err)
	}
	return receipt, balance, nil
}

func (r *Repo) ReverseReceipt(ctx context.Context, receiptID int64, comment string) (models.Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Balance{}, fmt.Errorf("reverse receipt: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		userID      int64
		status      string
		amountMinor int64
		currency    string
	)
	err = tx.QueryRow(
		ctx,
		`SELECT user_id, status, amount_minor, currency FROM receipts WHERE id = $1 FOR UPDATE`,
		receiptID,
	).Scan(&userID, &status, &amountMinor, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Balance{}, ErrReceiptNotFound
	}
	if err != nil {
		return models.Balance{}, fmt.Errorf("reverse receipt: %w", err)
	}
	if status == "rejected" {
		return models.Balance{}, ErrReceiptAlreadyReversed
	}

	if _, err := tx.Exec(ctx, `UPDATE receipts SET status = 'rejected' WHERE id = $1`, receiptID); err != nil {
		return models.Balance{}, fmt.Errorf("reverse receipt status: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO balance_transactions (user_id, amount_minor, currency, kind, receipt_id, comment)
		 VALUES ($1, $2, $3, 'receipt_reversal', $4, NULLIF($5, ''))`,
		userID, -amountMinor, currency, receiptID, comment,
	); err != nil {
		return models.Balance{}, fmt.Errorf("reverse receipt ledger: %w", err)
	}

	balance, err := applyBalanceDelta(ctx, tx, userID, -amountMinor, currency)
	if err != nil {
		return models.Balance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Balance{}, fmt.Errorf("reverse receipt: %w", err)
	}
	return balance, nil
}

func (r *Repo) AdjustBalance(ctx context.Context, userID any, amountMinor int64, currency, comment string) (models.Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Balance{}, fmt.Errorf("adjust balance: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := checkBalanceCurrency(ctx, tx, userID, currency); err != nil {
		return models.Balance{}, err
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO balance_transactions (user_id, amount_minor, currency, kind, comment)
		 VALUES ($1, $2, $3, 'admin_adjustment', NULLIF($4, ''))`,
		userID, amountMinor, currency, comment,
	); err != nil {
		return models.Balance{}, fmt.Errorf("adjust balance ledger: %w", err)
	}

	balance, err := applyBalanceDelta(ctx, tx, userID, amountMinor, currency)
	if err != nil {
		return models.Balance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Balance{}, fmt.Errorf("adjust balance: %w", err)
	}
	return balance, nil
}

func (r *Repo) ListTransactions(ctx context.Context, userID any, limit, offset int) ([]models.BalanceTransaction, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, user_id, amount_minor, currency, kind, receipt_id, comment, created_at
		 FROM balance_transactions
		 WHERE user_id = $1
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list balance transactions: %w", err)
	}

	transactions, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.BalanceTransaction, error) {
		var (
			tr      models.BalanceTransaction
			comment *string
		)
		if err := row.Scan(&tr.ID, &tr.UserID, &tr.AmountMinor, &tr.Currency, &tr.Kind,
			&tr.ReceiptID, &comment, &tr.CreatedAt); err != nil {
			return models.BalanceTransaction{}, err
		}
		if comment != nil {
			tr.Comment = *comment
		}
		tr.Amount = models.FormatMinor(tr.AmountMinor)
		return tr, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list balance transactions: %w", err)
	}
	return transactions, nil
}

func (r *Repo) ListReceipts(ctx context.Context, userID any, limit, offset int) ([]models.Receipt, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT `+receiptColumns+`
		 FROM receipts
		 WHERE user_id = $1
		 ORDER BY add_date_time DESC, id DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list receipts: %w", err)
	}

	receipts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Receipt, error) {
		return scanReceipt(row)
	})
	if err != nil {
		return nil, fmt.Errorf("list receipts: %w", err)
	}
	return receipts, nil
}

func (r *Repo) ListAllReceipts(ctx context.Context, status *string, limit, offset int) ([]models.Receipt, int64, error) {
	var total int64
	err := r.pool.QueryRow(
		ctx,
		`SELECT count(*) FROM receipts WHERE ($1::text IS NULL OR status = $1::receipt_status)`,
		status,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count receipts: %w", err)
	}

	rows, err := r.pool.Query(
		ctx,
		`SELECT `+receiptColumns+`
		 FROM receipts
		 WHERE ($1::text IS NULL OR status = $1::receipt_status)
		 ORDER BY add_date_time DESC, id DESC
		 LIMIT $2 OFFSET $3`,
		status, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list all receipts: %w", err)
	}

	receipts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Receipt, error) {
		return scanReceipt(row)
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list all receipts: %w", err)
	}
	return receipts, total, nil
}

func checkBalanceCurrency(ctx context.Context, tx pgx.Tx, userID any, currency string) error {
	var existing string
	err := tx.QueryRow(ctx, `SELECT currency FROM balances WHERE user_id = $1 FOR UPDATE`, userID).Scan(&existing)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lock balance: %w", err)
	}
	if existing != currency {
		return fmt.Errorf("%w: balance is in %s, receipt is in %s", ErrCurrencyMismatch, existing, currency)
	}
	return nil
}

func applyBalanceDelta(ctx context.Context, tx pgx.Tx, userID any, amountMinor int64, currency string) (models.Balance, error) {
	var (
		balance   models.Balance
		updatedAt time.Time
	)
	err := tx.QueryRow(
		ctx,
		`INSERT INTO balances (user_id, amount_minor, currency)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE
		 SET amount_minor = balances.amount_minor + EXCLUDED.amount_minor, updated_at = now()
		 RETURNING user_id, amount_minor, currency, updated_at`,
		userID, amountMinor, currency,
	).Scan(&balance.UserID, &balance.AmountMinor, &balance.Currency, &updatedAt)
	if err != nil {
		return models.Balance{}, fmt.Errorf("apply balance delta: %w", err)
	}

	balance.Amount = models.FormatMinor(balance.AmountMinor)
	balance.UpdatedAt = &updatedAt
	return balance, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanReceipt(row rowScanner) (models.Receipt, error) {
	var (
		receipt     models.Receipt
		checkNumber *string
		checkDate   *string
		bank        *string
		card        *string
		payer       *string
		service     *string
		account     *string
		recipient   *string
	)

	if err := row.Scan(
		&receipt.ID, &receipt.UserID, &receipt.Status, &receipt.FileObject, &checkNumber,
		&receipt.AmountMinor, &receipt.Currency, &checkDate, &bank, &card, &payer,
		&service, &account, &recipient, &receipt.AddDateTime,
	); err != nil {
		return models.Receipt{}, err
	}

	receipt.Amount = models.FormatMinor(receipt.AmountMinor)
	receipt.CheckNumber = deref(checkNumber)
	receipt.CheckDate = deref(checkDate)
	receipt.Bank = deref(bank)
	receipt.Card = deref(card)
	receipt.Payer = deref(payer)
	receipt.Service = deref(service)
	receipt.Account = deref(account)
	receipt.Recipient = deref(recipient)
	return receipt, nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func duplicateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		switch pgErr.ConstraintName {
		case fileHashConstraint:
			return ErrDuplicateFile
		case checkNumberConstraint:
			return ErrDuplicateCheckNumber
		}
	}
	return fmt.Errorf("create receipt: %w", err)
}
