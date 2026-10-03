package auth

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/tomasen/realip"
)

const cookieMaxAge = 400 * 24 * 3600

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, userID uuid.UUID) error {
	if old, err := r.Cookie("session"); err == nil {
		_ = h.Sessions.Delete(r.Context(), old.Value)
	}
	token, err := h.Sessions.Create(r.Context(), userID, realip.FromRequest(r), r.UserAgent())
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   cookieMaxAge,
	})
	return nil
}

func clearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode,
	})
}
