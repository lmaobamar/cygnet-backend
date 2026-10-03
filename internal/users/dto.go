package users

import (
	"time"

	"github.com/google/uuid"
	"github.com/lmaobamar/cygnet-backend/internal/models"
)

type PublicUser struct {
	ID          uuid.UUID `json:"id"`
	Handle      string    `json:"handle"`
	DisplayName string    `json:"display_name"`
	Verified    bool      `json:"verified"`
	CreatedAt   time.Time `json:"created_at"`
}

type SelfUser struct {
	PublicUser
	Email string `json:"email"`
	Admin bool   `json:"admin"`
}

func PublicOf(u *models.User) PublicUser {
	return PublicUser{
		ID:          u.ID,
		Handle:      u.Handle,
		DisplayName: u.DisplayName,
		Verified:    u.Verified,
		CreatedAt:   u.CreatedAt,
	}
}

func SelfOf(u *models.User) SelfUser {
	return SelfUser{PublicUser: PublicOf(u), Email: u.Email, Admin: u.Admin}
}
