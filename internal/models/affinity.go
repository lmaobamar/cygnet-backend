package models

import (
	"time"

	"github.com/google/uuid"
)

type UserTagAffinity struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	TagID     int64     `gorm:"primaryKey"`
	Score     float64   `gorm:"not null;default:0"`
	UpdatedAt time.Time `gorm:"not null"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Tag       Tag       `gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE"`
}
