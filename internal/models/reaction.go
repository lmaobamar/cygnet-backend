package models

import (
	"time"

	"github.com/google/uuid"
)

type ReactionKind string

const (
	ReactionLike    ReactionKind = "like"
	ReactionDislike ReactionKind = "dislike"
)

type Reaction struct {
	UserID uuid.UUID
	PostID uuid.UUID
	Kind   ReactionKind

	User User
	Post Post

	CreatedAt time.Time
	UpdatedAt time.Time
}
