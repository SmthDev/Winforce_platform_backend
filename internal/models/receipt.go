package models

import "time"

type ParsedReceipt struct {
	CheckNumber string  `json:"check_number,omitempty"`
	Date        string  `json:"date,omitempty"`
	Bank        string  `json:"bank,omitempty"`
	Card        string  `json:"card,omitempty"`
	Payer       string  `json:"payer,omitempty"`
	Service     string  `json:"service,omitempty"`
	Account     string  `json:"account,omitempty"`
	Recipient   string  `json:"recipient,omitempty"`
	Amount      float64 `json:"amount,omitempty"`
	Currency    string  `json:"currency,omitempty"`
}

type Receipt struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Status      string    `json:"status"`
	Amount      string    `json:"amount"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	CheckNumber string    `json:"check_number,omitempty"`
	CheckDate   string    `json:"check_date,omitempty"`
	Bank        string    `json:"bank,omitempty"`
	Card        string    `json:"card,omitempty"`
	Payer       string    `json:"payer,omitempty"`
	Service     string    `json:"service,omitempty"`
	Account     string    `json:"account,omitempty"`
	Recipient   string    `json:"recipient,omitempty"`
	FileObject  string    `json:"-"`
	FileURL     string    `json:"file_url,omitempty"`
	AddDateTime time.Time `json:"add_date_time"`
}
