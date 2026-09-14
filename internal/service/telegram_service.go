package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"platform/backend/internal/models"
	"platform/backend/internal/repository/postgres/telegram_repo"
)

const telegramAuthMaxAge = 24 * time.Hour

var (
	ErrTelegramNotConfigured    = errors.New("telegram integration is not configured")
	ErrInvalidTelegramSignature = errors.New("invalid telegram signature")
	ErrTelegramAuthExpired      = errors.New("telegram auth data expired")
	ErrTelegramAccountTaken     = errors.New("telegram account already linked to another user")
	ErrTelegramNotLinked        = errors.New("telegram account is not linked")
)

type TelegramRepository interface {
	LinkAccount(ctx context.Context, userID any, telegramID int64, username, firstName, lastName, photoURL string) (models.TelegramAccount, error)
	GetByUserID(ctx context.Context, userID any) (models.TelegramAccount, bool, error)
	Unlink(ctx context.Context, userID any) (bool, error)
}

type TelegramService struct {
	repo     TelegramRepository
	botToken string
}

func NewTelegram(repo TelegramRepository, botToken string) *TelegramService {
	return &TelegramService{repo: repo, botToken: botToken}
}

func (s *TelegramService) LinkAccount(ctx context.Context, userID any, payload models.TelegramAuthPayload) (models.TelegramAccount, error) {
	if s.botToken == "" {
		return models.TelegramAccount{}, ErrTelegramNotConfigured
	}

	if err := verifyTelegramAuth(payload, s.botToken); err != nil {
		return models.TelegramAccount{}, err
	}

	if time.Since(time.Unix(payload.AuthDate, 0)) > telegramAuthMaxAge {
		return models.TelegramAccount{}, ErrTelegramAuthExpired
	}

	acc, err := s.repo.LinkAccount(ctx, userID, payload.ID, payload.Username, payload.FirstName, payload.LastName, payload.PhotoURL)
	if err != nil {
		if errors.Is(err, telegram_repo.ErrTelegramAccountTaken) {
			return models.TelegramAccount{}, ErrTelegramAccountTaken
		}
		return models.TelegramAccount{}, fmt.Errorf("link telegram account: %w", err)
	}
	return acc, nil
}

func (s *TelegramService) GetLink(ctx context.Context, userID any) (models.TelegramAccount, error) {
	acc, found, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return models.TelegramAccount{}, fmt.Errorf("get telegram link: %w", err)
	}
	if !found {
		return models.TelegramAccount{}, ErrTelegramNotLinked
	}
	return acc, nil
}

func (s *TelegramService) Unlink(ctx context.Context, userID any) error {
	removed, err := s.repo.Unlink(ctx, userID)
	if err != nil {
		return fmt.Errorf("unlink telegram account: %w", err)
	}
	if !removed {
		return ErrTelegramNotLinked
	}
	return nil
}

func verifyTelegramAuth(payload models.TelegramAuthPayload, botToken string) error {
	fields := map[string]string{
		"id":         strconv.FormatInt(payload.ID, 10),
		"first_name": payload.FirstName,
		"last_name":  payload.LastName,
		"username":   payload.Username,
		"photo_url":  payload.PhotoURL,
		"auth_date":  strconv.FormatInt(payload.AuthDate, 10),
	}

	keys := make([]string, 0, len(fields))
	for k, v := range fields {
		if v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+fields[k])
	}
	dataCheckString := strings.Join(pairs, "\n")

	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(strings.ToLower(payload.Hash))) {
		return ErrInvalidTelegramSignature
	}
	return nil
}
