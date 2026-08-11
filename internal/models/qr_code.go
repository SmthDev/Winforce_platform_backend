package models

import "time"

type QRCode struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	TargetLink  string    `json:"target_link"`
	QRLink      string    `json:"qr_link"`
	AddDateTime time.Time `json:"add_date_time"`
}
