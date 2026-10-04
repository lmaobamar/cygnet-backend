package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Post struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	AuthorID uuid.UUID `gorm:"type:uuid;not null;index:idx_posts_author_created,priority:1"`
	Author   User      `gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`

	Body  string      `gorm:"size:2000;not null;default:''"`
	Media []PostMedia `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`

	LikeCount    int `gorm:"not null;default:0"`
	DislikeCount int `gorm:"not null;default:0"`
	CommentCount int `gorm:"not null;default:0"`

	CreatedAt time.Time      `gorm:"index:idx_posts_author_created,priority:2,sort:desc;index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (p *Post) BeforeCreate(tx *gorm.DB) error { return newID(&p.ID) }

type MediaKind string

const (
	MediaImage MediaKind = "image"
	MediaVideo MediaKind = "video"
)

type PostMedia struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	PostID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Position    int       `gorm:"not null;default:0"`
	Kind        MediaKind `gorm:"size:16;not null"`
	Key         string    `gorm:"size:255;not null"` // storage key, not the URL
	ContentType string    `gorm:"size:64;not null"`
	Size        int64     `gorm:"not null"`
}

func (m *PostMedia) BeforeCreate(tx *gorm.DB) error { return newID(&m.ID) }
