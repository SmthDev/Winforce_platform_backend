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
	ErrDuplicateFile          = errors.New("receipt file already uploaded")
	ErrDuplicateCheckNumber   = errors.New("receipt check number already used")
	ErrCurrencyMismatch       = errors.New("receipt currency does not match balance currency")
	ErrReceiptNotFound        = errors.New("receipt not found")
	ErrReceiptAlreadyReversed = errors.New("receipt already reversed")
	ErrGameSettingsMissing    = errors.New("game settings not configured")
	ErrGameNotFound           = errors.New("game not found")
	ErrGameAlreadyCharged     = errors.New("game already charged")
)

type GameShare struct {
	UserID      int64
	AmountMinor int64
}

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

func (r *Repo) MissingUsers(ctx context.Context, userIDs []int64) ([]int64, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT req.id FROM unnest($1::bigint[]) AS req(id)
		 WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.id = req.id)
		 ORDER BY req.id`,
		userIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("check users exist: %w", err)
	}
	missing, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, fmt.Errorf("check users exist: %w", err)
	}
	return missing, nil
}

func (r *Repo) GameChargeAmount(ctx context.Context) (int64, error) {
	var amountMinor int64
	err := r.pool.QueryRow(ctx, `SELECT charge_amount_minor FROM game_settings WHERE id = 1`).Scan(&amountMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrGameSettingsMissing
	}
	if err != nil {
		return 0, fmt.Errorf("get game charge amount: %w", err)
	}
	return amountMinor, nil
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

func (r *Repo) ChargeGame(ctx context.Context, gameID int64, shares []GameShare, currency string) (models.Game, []models.Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Game{}, nil, fmt.Errorf("charge game: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		game     models.Game
		playedOn time.Time
	)
	err = tx.QueryRow(
		ctx,
		`SELECT id, played_on, opponent, charged_at, created_at FROM games WHERE id = $1 FOR UPDATE`,
		gameID,
	).Scan(&game.ID, &playedOn, &game.Opponent, &game.ChargedAt, &game.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Game{}, nil, ErrGameNotFound
	}
	if err != nil {
		return models.Game{}, nil, fmt.Errorf("lock game: %w", err)
	}
	if game.ChargedAt != nil {
		return models.Game{}, nil, ErrGameAlreadyCharged
	}
	game.PlayedOn = playedOn.Format(models.GameDateLayout)
	comment := fmt.Sprintf("Игра %s против %s", game.PlayedOn, game.Opponent)

	userIDs := make([]int64, len(shares))
	for i, share := range shares {
		userIDs[i] = share.UserID
	}

	rows, err := tx.Query(
		ctx,
		`SELECT user_id, currency FROM balances
		 WHERE user_id = ANY($1)
		 ORDER BY user_id
		 FOR UPDATE`,
		userIDs,
	)
	if err != nil {
		return models.Game{}, nil, fmt.Errorf("lock balances: %w", err)
	}

	currencies := make(map[int64]string, len(shares))
	for rows.Next() {
		var (
			userID       int64
			userCurrency string
		)
		if err := rows.Scan(&userID, &userCurrency); err != nil {
			rows.Close()
			return models.Game{}, nil, fmt.Errorf("lock balances: %w", err)
		}
		currencies[userID] = userCurrency
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return models.Game{}, nil, fmt.Errorf("lock balances: %w", err)
	}

	for _, share := range shares {
		if userCurrency, ok := currencies[share.UserID]; ok && userCurrency != currency {
			return models.Game{}, nil, fmt.Errorf("%w: user %d balance is in %s, charge is in %s",
				ErrCurrencyMismatch, share.UserID, userCurrency, currency)
		}
	}

	balances := make([]models.Balance, 0, len(shares))
	for _, share := range shares {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO balance_transactions (user_id, amount_minor, currency, kind, game_id, comment)
			 VALUES ($1, $2, $3, 'game_charge', $4, $5)`,
			share.UserID, -share.AmountMinor, currency, gameID, comment,
		); err != nil {
			return models.Game{}, nil, fmt.Errorf("charge game ledger: %w", err)
		}

		balance, err := applyBalanceDelta(ctx, tx, share.UserID, -share.AmountMinor, currency)
		if err != nil {
			return models.Game{}, nil, err
		}
		balances = append(balances, balance)
	}

	if err := tx.QueryRow(
		ctx,
		`UPDATE games SET charged_at = now() WHERE id = $1 RETURNING charged_at`,
		gameID,
	).Scan(&game.ChargedAt); err != nil {
		return models.Game{}, nil, fmt.Errorf("mark game charged: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Game{}, nil, fmt.Errorf("charge game: %w", err)
	}
	return game, balances, nil
}

func (r *Repo) ListTransactions(ctx context.Context, userID any, limit, offset int) ([]models.BalanceTransaction, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, user_id, amount_minor, currency, kind, receipt_id, game_id, comment, created_at
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
			&tr.ReceiptID, &tr.GameID, &comment, &tr.CreatedAt); err != nil {
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


func (r *Repo) ListGamePayments(ctx context.Context, userID any, limit, offset int) ([]models.GamePayment, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT bt.id, -bt.amount_minor, bt.currency, bt.created_at,
			g.id, g.played_on, g.opponent, g.charged_at, g.created_at
		 FROM balance_transactions bt
		 LEFT JOIN games g ON g.id = bt.game_id
		 WHERE bt.user_id = $1 AND bt.kind = 'game_charge'
		 ORDER BY bt.created_at DESC, bt.id DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list game payments: %w", err)
	}

	payments, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.GamePayment, error) {
		var (
			p             models.GamePayment
			gameID        *int64
			playedOn      *time.Time
			opponent      *string
			chargedAt     *time.Time
			gameCreatedAt *time.Time
		)
		if err := row.Scan(&p.TransactionID, &p.AmountMinor, &p.Currency, &p.CreatedAt,
			&gameID, &playedOn, &opponent, &chargedAt, &gameCreatedAt); err != nil {
			return models.GamePayment{}, err
		}
		if gameID != nil {
			p.Game = &models.Game{
				ID:        *gameID,
				PlayedOn:  playedOn.Format(models.GameDateLayout),
				Opponent:  *opponent,
				ChargedAt: chargedAt,
				CreatedAt: *gameCreatedAt,
			}
		}
		p.Amount = models.FormatMinor(p.AmountMinor)
		return p, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list game payments: %w", err)
	}
	return payments, nil
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


const userOverviewSelect = `SELECT u.id, u.email, u.first_name, u.last_name, u.role::text, u.created_at,
		b.amount_minor, b.currency, b.updated_at,
		COALESCE(s.games_count, 0), COALESCE(s.games_paid_minor, 0), COALESCE(s.topped_up_minor, 0),
		COALESCE(rc.receipts_count, 0), s.last_game_on
	 FROM users u
	 LEFT JOIN balances b ON b.user_id = u.id
	 LEFT JOIN LATERAL (
		SELECT count(*) FILTER (WHERE bt.kind = 'game_charge') AS games_count,
			(-COALESCE(sum(bt.amount_minor) FILTER (WHERE bt.kind = 'game_charge'), 0))::bigint AS games_paid_minor,
			COALESCE(sum(bt.amount_minor) FILTER (WHERE bt.kind IN ('receipt_credit', 'receipt_reversal')), 0)::bigint AS topped_up_minor,
			max(g.played_on) AS last_game_on
		FROM balance_transactions bt
		LEFT JOIN games g ON g.id = bt.game_id
		WHERE bt.user_id = u.id
	 ) s ON true
	 LEFT JOIN LATERAL (
		SELECT count(*) AS receipts_count FROM receipts r WHERE r.user_id = u.id AND r.status = 'credited'
	 ) rc ON true`

const userSearchFilter = `($1 = '' OR u.email ILIKE '%' || $1 || '%'
		OR u.first_name ILIKE '%' || $1 || '%' OR u.last_name ILIKE '%' || $1 || '%')`

func (r *Repo) ListUserOverviews(ctx context.Context, search string, limit, offset int) ([]models.UserOverview, int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM users u WHERE `+userSearchFilter, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	rows, err := r.pool.Query(
		ctx,
		userOverviewSelect+`
		 WHERE `+userSearchFilter+`
		 ORDER BY u.id
		 LIMIT $2 OFFSET $3`,
		search, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list user overviews: %w", err)
	}

	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.UserOverview, error) {
		return scanUserOverview(row)
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list user overviews: %w", err)
	}
	return users, total, nil
}

func (r *Repo) GetUserOverview(ctx context.Context, userID any) (models.UserOverview, bool, error) {
	user, err := scanUserOverview(r.pool.QueryRow(ctx, userOverviewSelect+` WHERE u.id = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.UserOverview{}, false, nil
	}
	if err != nil {
		return models.UserOverview{}, false, fmt.Errorf("get user overview: %w", err)
	}
	return user, true, nil
}


func scanUserOverview(row rowScanner) (models.UserOverview, error) {
	var (
		user             models.UserOverview
		firstName        *string
		lastName         *string
		balanceMinor     *int64
		balanceCurrency  *string
		balanceUpdatedAt *time.Time
		lastGameOn       *time.Time
	)
	if err := row.Scan(
		&user.ID, &user.Email, &firstName, &lastName, &user.Role, &user.CreatedAt,
		&balanceMinor, &balanceCurrency, &balanceUpdatedAt,
		&user.Stats.GamesCount, &user.Stats.GamesPaidMinor, &user.Stats.ToppedUpMinor,
		&user.Stats.ReceiptsCount, &lastGameOn,
	); err != nil {
		return models.UserOverview{}, err
	}

	user.FirstName = deref(firstName)
	user.LastName = deref(lastName)

	user.Balance.UserID = user.ID
	if balanceMinor != nil {
		user.Balance.AmountMinor = *balanceMinor
		user.Balance.Currency = *balanceCurrency
		user.Balance.UpdatedAt = balanceUpdatedAt
	}
	user.Balance.Amount = models.FormatMinor(user.Balance.AmountMinor)

	user.Stats.GamesPaid = models.FormatMinor(user.Stats.GamesPaidMinor)
	user.Stats.ToppedUp = models.FormatMinor(user.Stats.ToppedUpMinor)
	if lastGameOn != nil {
		formatted := lastGameOn.Format(models.GameDateLayout)
		user.Stats.LastGameOn = &formatted
	}
	return user, nil
}
