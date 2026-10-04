package models

import "github.com/google/uuid"

type Tag struct {
	ID   int64  `gorm:"primaryKey;autoIncrement"`
	Slug string `gorm:"size:32;not null;uniqueIndex"` // lowercase plssss
}

type PostTag struct {
	PostID uuid.UUID `gorm:"type:uuid;primaryKey"`
	TagID  int64     `gorm:"primaryKey;index"` // index = all posts with tag X
	Post   Post      `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	Tag    Tag       `gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE"`
}
