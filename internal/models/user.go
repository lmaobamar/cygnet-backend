package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// everything is "-" so we dont accidentally send the model
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
	Handle       string    `gorm:"uniqueIndex;size:32;not null" json:"-"`
	DisplayName  string    `gorm:"size:50;not null" json:"-"`
	Email        string    `gorm:"uniqueIndex;size:255;not null" json:"-"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Admin        bool      `gorm:"not null;default:false" json:"-"`
	Verified     bool      `gorm:"not null;default:false" json:"-"`
	CreatedAt    time.Time `json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		u.ID = id
	}
	return nil
}
