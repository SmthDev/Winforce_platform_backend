package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiBaseURL     = "https://api.telegram.org"
	requestTimeout = 15 * time.Second
	maxErrorBody   = 256
)

var ErrNotConfigured = errors.New("telegram bot token is not configured")

type Bot struct {
	httpClient *http.Client
	token      string
}

func NewBot(token string) *Bot {
	return &Bot{
		httpClient: &http.Client{Timeout: requestTimeout},
		token:      token,
	}
}

func (b *Bot) SendMessage(ctx context.Context, chatID int64, text string) error {
	if b.token == "" {
		return ErrNotConfigured
	}

	body, err := json.Marshal(map[string]any{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return fmt.Errorf("build sendMessage request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"/bot"+b.token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build sendMessage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {

		return errors.New("call sendMessage: request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return fmt.Errorf("sendMessage returned %d: %s", resp.StatusCode, payload)
	}
	return nil
}
