package models

import "time"

const GameDateLayout = "2006-01-02"

type Game struct {
	ID        int64      `json:"id"`
	PlayedOn  string     `json:"played_on"`
	Opponent  string     `json:"opponent"`
	ChargedAt *time.Time `json:"charged_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type GamePlayer struct {
	UserID           int64     `json:"user_id"`
	Email            string    `json:"email"`
	FirstName        string    `json:"first_name,omitempty"`
	LastName         string    `json:"last_name,omitempty"`
	TelegramUsername string    `json:"telegram_username,omitempty"`
	Amount           string    `json:"amount"`
	AmountMinor      int64     `json:"amount_minor"`
	Currency         string    `json:"currency"`
	ChargedAt        time.Time `json:"charged_at"`
}
