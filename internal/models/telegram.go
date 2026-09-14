package models

import "time"

type TelegramAccount struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username,omitempty"`
	FirstName  string    `json:"first_name,omitempty"`
	LastName   string    `json:"last_name,omitempty"`
	PhotoURL   string    `json:"photo_url,omitempty"`
	LinkedAt   time.Time `json:"linked_at"`
}

type TelegramAuthPayload struct {
	ID        int64  `json:"id" binding:"required"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date" binding:"required"`
	Hash      string `json:"hash" binding:"required"`
}
