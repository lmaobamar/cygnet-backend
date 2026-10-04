package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/lmaobamar/cygnet-backend/internal/httpx"
)

const cookieMaxAge = 400 * 24 * time.Hour

func (h *Handler) sessionCookie(ctx context.Context, userID uuid.UUID, oldToken, userAgent string) (http.Cookie, error) {
	if oldToken != "" {
		_ = h.Sessions.Delete(ctx, oldToken)
	}
	meta := httpx.MetaFrom(ctx)
	token, err := h.Sessions.Create(ctx, userID, meta.IP, userAgent)
	if err != nil {
		return http.Cookie{}, err
	}
	return http.Cookie{
		Name: "session", Value: token, Path: "/",
		HttpOnly: true, Secure: meta.HTTPS, SameSite: http.SameSiteLaxMode,
		MaxAge: int(cookieMaxAge.Seconds()),
	}, nil
}

func expiredCookie(ctx context.Context) http.Cookie {
	return http.Cookie{
		Name: "session", Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: httpx.MetaFrom(ctx).HTTPS, SameSite: http.SameSiteLaxMode,
	}
}
