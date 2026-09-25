package models

import (
	"fmt"
	"math"
	"time"
)

const minorUnits = 100


type Balance struct {
	UserID      int64      `json:"user_id,omitempty"`
	Amount      string     `json:"amount"`
	AmountMinor int64      `json:"amount_minor"`
	Currency    string     `json:"currency"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type BalanceTransaction struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Amount      string    `json:"amount"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	Kind        string    `json:"kind"`
	ReceiptID   *int64    `json:"receipt_id,omitempty"`
	GameID      *int64    `json:"game_id,omitempty"`
	Comment     string    `json:"comment,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}


func FormatMinor(amountMinor int64) string {
	sign := ""
	if amountMinor < 0 {
		sign = "-"
		amountMinor = -amountMinor
	}
	return fmt.Sprintf("%s%d.%02d", sign, amountMinor/minorUnits, amountMinor%minorUnits)
}


func MinorFromFloat(amount float64) int64 {
	return int64(math.Round(amount * minorUnits))
}

type GameChargeShare struct {
	UserID      int64   `json:"user_id"`
	Amount      string  `json:"amount"`
	AmountMinor int64   `json:"amount_minor"`
	Balance     Balance `json:"balance"`
}

type GameCharge struct {
	Game        Game              `json:"game"`
	Amount      string            `json:"amount"`
	AmountMinor int64             `json:"amount_minor"`
	Currency    string            `json:"currency"`
	Charges     []GameChargeShare `json:"charges"`
}

type GamePayment struct {
	TransactionID int64     `json:"transaction_id"`
	Game          *Game     `json:"game,omitempty"`
	Amount        string    `json:"amount"`
	AmountMinor   int64     `json:"amount_minor"`
	Currency      string    `json:"currency"`
	CreatedAt     time.Time `json:"created_at"`
}

type UserStats struct {
	GamesCount     int64   `json:"games_count"`
	GamesPaid      string  `json:"games_paid"`
	GamesPaidMinor int64   `json:"games_paid_minor"`
	ToppedUp       string  `json:"topped_up"`
	ToppedUpMinor  int64   `json:"topped_up_minor"`
	ReceiptsCount  int64   `json:"receipts_count"`
	LastGameOn     *string `json:"last_game_on,omitempty"`
}


type UserOverview struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	Balance   Balance   `json:"balance"`
	Stats     UserStats `json:"stats"`
	TelegramUsername string `json:"telegram_username,omitempty"`
}

type UserDashboard struct {
	User     UserOverview  `json:"user"`
	Games    []GamePayment `json:"games"`
	Receipts []Receipt     `json:"receipts"`
}
