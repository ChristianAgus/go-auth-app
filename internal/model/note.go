package model

import "time"

type Note struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"-"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}