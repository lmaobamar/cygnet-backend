package models

import "time"

type User struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:uniqueIndex;size:32;not null" json:"username"`
	Email        string    `gorm:"uniqueIndex;size:255;not null" json:"-"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
