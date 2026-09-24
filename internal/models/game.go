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
