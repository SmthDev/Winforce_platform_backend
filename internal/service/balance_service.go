package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"platform/backend/internal/models"
	"platform/backend/internal/repository/parsing"
	"platform/backend/internal/repository/postgres/balance_repo"
)

const (
	receiptURLTTL    = 15 * time.Minute
	defaultListLimit = 50
	maxListLimit     = 200
)

var (
	ErrReceiptDuplicate = errors.New("receipt already uploaded")
	ErrReceiptNotRecognized = errors.New("receipt amount not recognized")
	ErrReceiptBadFile = errors.New("receipt file rejected")
	ErrBalanceCurrencyMismatch = errors.New("receipt currency does not match balance currency")
	ErrParsingUnavailable = errors.New("parsing service unavailable")
	ErrReceiptNotFound = errors.New("receipt not found")
	ErrReceiptAlreadyReversed = errors.New("receipt already reversed")
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidReceiptStatus = errors.New("unknown receipt status")
)

var ReceiptStatuses = []string{"credited", "rejected"}

type BalanceRepository interface {
	GetBalance(ctx context.Context, userID any) (balance models.Balance, found bool, err error)
	UserExists(ctx context.Context, userID any) (bool, error)
	ReceiptExistsByFileHash(ctx context.Context, fileHash string) (bool, error)
	CreditReceipt(ctx context.Context, userID any, in balance_repo.ReceiptInsert) (models.Receipt, models.Balance, error)
	ReverseReceipt(ctx context.Context, receiptID int64, comment string) (models.Balance, error)
	AdjustBalance(ctx context.Context, userID any, amountMinor int64, currency, comment string) (models.Balance, error)
	ListTransactions(ctx context.Context, userID any, limit, offset int) ([]models.BalanceTransaction, error)
	ListReceipts(ctx context.Context, userID any, limit, offset int) ([]models.Receipt, error)
	ListAllReceipts(ctx context.Context, status *string, limit, offset int) ([]models.Receipt, int64, error)
}

type ReceiptParser interface {
	ParseReceipt(ctx context.Context, fileName string, file io.Reader) (models.ParsedReceipt, []byte, error)
}

type ReceiptStorage interface {
	Upload(ctx context.Context, bucket, objectName string, reader io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, bucket, objectName string) error
	SignedURL(bucket, objectName string, expiry time.Duration) (string, error)
}

type BalanceService struct {
	repo            BalanceRepository
	parser          ReceiptParser
	storage         ReceiptStorage
	log             *slog.Logger
	defaultCurrency string
}

func NewBalance(repo BalanceRepository, parser ReceiptParser, storage ReceiptStorage, log *slog.Logger, defaultCurrency string) *BalanceService {
	return &BalanceService{
		repo:            repo,
		parser:          parser,
		storage:         storage,
		log:             log,
		defaultCurrency: defaultCurrency,
	}
}

func (s *BalanceService) GetBalance(ctx context.Context, userID any) (models.Balance, error) {
	balance, found, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		return models.Balance{}, fmt.Errorf("load balance: %w", err)
	}
	if !found {
		return models.Balance{
			Amount:      models.FormatMinor(0),
			AmountMinor: 0,
			Currency:    s.defaultCurrency,
		}, nil
	}
	return balance, nil
}

func (s *BalanceService) CreditFromReceipt(ctx context.Context, userID any, bucket, fileName, contentType string, data []byte) (models.Receipt, models.Balance, error) {
	sum := sha256.Sum256(data)
	fileHash := hex.EncodeToString(sum[:])

	exists, err := s.repo.ReceiptExistsByFileHash(ctx, fileHash)
	if err != nil {
		return models.Receipt{}, models.Balance{}, err
	}
	if exists {
		return models.Receipt{}, models.Balance{}, ErrReceiptDuplicate
	}

	parsed, raw, err := s.parser.ParseReceipt(ctx, fileName, bytes.NewReader(data))
	if err != nil {
		return models.Receipt{}, models.Balance{}, translateParsingError(err)
	}

	amountMinor := models.MinorFromFloat(parsed.Amount)
	currency := strings.ToUpper(strings.TrimSpace(parsed.Currency))
	if amountMinor <= 0 || currency == "" {
		return models.Receipt{}, models.Balance{}, fmt.Errorf("%w: amount=%v currency=%q",
			ErrReceiptNotRecognized, parsed.Amount, parsed.Currency)
	}

	objectName := uuid.NewString() + strings.ToLower(filepath.Ext(fileName))
	if err := s.storage.Upload(ctx, bucket, objectName, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		return models.Receipt{}, models.Balance{}, fmt.Errorf("upload receipt: %w", err)
	}

	receipt, balance, err := s.repo.CreditReceipt(ctx, userID, balance_repo.ReceiptInsert{
		FileObject:  objectName,
		FileHash:    fileHash,
		AmountMinor: amountMinor,
		Currency:    currency,
		CheckNumber: strings.TrimSpace(parsed.CheckNumber),
		CheckDate:   strings.TrimSpace(parsed.Date),
		Bank:        strings.TrimSpace(parsed.Bank),
		Card:        strings.TrimSpace(parsed.Card),
		Payer:       strings.TrimSpace(parsed.Payer),
		Service:     strings.TrimSpace(parsed.Service),
		Account:     strings.TrimSpace(parsed.Account),
		Recipient:   strings.TrimSpace(parsed.Recipient),
		RawResponse: raw,
	})
	if err != nil {
		if delErr := s.storage.Delete(ctx, bucket, objectName); delErr != nil {
			s.log.Error("delete orphaned receipt object failed",
				slog.Any("error", delErr),
				slog.String("object_name", objectName),
				slog.Any("user_id", userID),
			)
		}
		return models.Receipt{}, models.Balance{}, translateRepoError(err)
	}

	receipt.FileURL = s.receiptURL(bucket, receipt.FileObject)
	return receipt, balance, nil
}

func (s *BalanceService) ListTransactions(ctx context.Context, userID any, limit, offset int) ([]models.BalanceTransaction, error) {
	transactions, err := s.repo.ListTransactions(ctx, userID, clampLimit(limit), clampOffset(offset))
	if err != nil {
		return nil, fmt.Errorf("load balance transactions: %w", err)
	}
	return transactions, nil
}

func (s *BalanceService) ListReceipts(ctx context.Context, userID any, bucket string, limit, offset int) ([]models.Receipt, error) {
	receipts, err := s.repo.ListReceipts(ctx, userID, clampLimit(limit), clampOffset(offset))
	if err != nil {
		return nil, fmt.Errorf("load receipts: %w", err)
	}

	for i := range receipts {
		receipts[i].FileURL = s.receiptURL(bucket, receipts[i].FileObject)
	}
	return receipts, nil
}

func (s *BalanceService) GetUserBalance(ctx context.Context, userID any) (models.Balance, error) {
	if err := s.requireUser(ctx, userID); err != nil {
		return models.Balance{}, err
	}
	return s.GetBalance(ctx, userID)
}

func (s *BalanceService) ListUserTransactions(ctx context.Context, userID any, limit, offset int) ([]models.BalanceTransaction, error) {
	if err := s.requireUser(ctx, userID); err != nil {
		return nil, err
	}
	return s.ListTransactions(ctx, userID, limit, offset)
}

func (s *BalanceService) ListUserReceipts(ctx context.Context, userID any, bucket string, limit, offset int) ([]models.Receipt, error) {
	if err := s.requireUser(ctx, userID); err != nil {
		return nil, err
	}
	return s.ListReceipts(ctx, userID, bucket, limit, offset)
}

func (s *BalanceService) ListAllReceipts(ctx context.Context, status, bucket string, limit, offset int) ([]models.Receipt, int64, error) {
	var filter *string
	if status = strings.ToLower(strings.TrimSpace(status)); status != "" {
		if !slices.Contains(ReceiptStatuses, status) {
			return nil, 0, fmt.Errorf("%w: %q", ErrInvalidReceiptStatus, status)
		}
		filter = &status
	}

	receipts, total, err := s.repo.ListAllReceipts(ctx, filter, clampLimit(limit), clampOffset(offset))
	if err != nil {
		return nil, 0, fmt.Errorf("load all receipts: %w", err)
	}

	for i := range receipts {
		receipts[i].FileURL = s.receiptURL(bucket, receipts[i].FileObject)
	}
	return receipts, total, nil
}

func (s *BalanceService) RejectReceipt(ctx context.Context, receiptID int64, comment string) (models.Balance, error) {
	balance, err := s.repo.ReverseReceipt(ctx, receiptID, comment)
	if err != nil {
		return models.Balance{}, translateRepoError(err)
	}
	return balance, nil
}

func (s *BalanceService) AdjustBalance(ctx context.Context, userID any, amountMinor int64, currency, comment string) (models.Balance, error) {
	if err := s.requireUser(ctx, userID); err != nil {
		return models.Balance{}, err
	}

	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = s.defaultCurrency
	}

	balance, err := s.repo.AdjustBalance(ctx, userID, amountMinor, currency, comment)
	if err != nil {
		return models.Balance{}, translateRepoError(err)
	}
	return balance, nil
}

func (s *BalanceService) receiptURL(bucket, objectName string) string {
	if objectName == "" {
		return ""
	}

	url, err := s.storage.SignedURL(bucket, objectName, receiptURLTTL)
	if err != nil {
		s.log.Error("sign receipt link failed",
			slog.Any("error", err),
			slog.String("object_name", objectName),
		)
		return ""
	}
	return url
}

func (s *BalanceService) requireUser(ctx context.Context, userID any) error {
	exists, err := s.repo.UserExists(ctx, userID)
	if err != nil {
		return fmt.Errorf("check user: %w", err)
	}
	if !exists {
		return fmt.Errorf("%w: id=%v", ErrUserNotFound, userID)
	}
	return nil
}

func translateParsingError(err error) error {
	switch {
	case errors.Is(err, parsing.ErrNotRecognized):
		return fmt.Errorf("%w: %v", ErrReceiptNotRecognized, err)
	case errors.Is(err, parsing.ErrBadFile):
		return fmt.Errorf("%w: %v", ErrReceiptBadFile, err)
	case errors.Is(err, parsing.ErrNotConfigured), errors.Is(err, parsing.ErrUnavailable):
		return fmt.Errorf("%w: %v", ErrParsingUnavailable, err)
	default:
		return fmt.Errorf("parse receipt: %w", err)
	}
}

func translateRepoError(err error) error {
	switch {
	case errors.Is(err, balance_repo.ErrDuplicateFile), errors.Is(err, balance_repo.ErrDuplicateCheckNumber):
		return fmt.Errorf("%w: %v", ErrReceiptDuplicate, err)
	case errors.Is(err, balance_repo.ErrCurrencyMismatch):
		return fmt.Errorf("%w: %v", ErrBalanceCurrencyMismatch, err)
	case errors.Is(err, balance_repo.ErrReceiptNotFound):
		return fmt.Errorf("%w: %v", ErrReceiptNotFound, err)
	case errors.Is(err, balance_repo.ErrReceiptAlreadyReversed):
		return fmt.Errorf("%w: %v", ErrReceiptAlreadyReversed, err)
	default:
		return err
	}
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

func clampOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
