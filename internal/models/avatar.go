package models

import "time"

type Avatar struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	AvatarLink  string    `json:"avatar_link"`
	AddDateTime time.Time `json:"add_date_time"`
}
