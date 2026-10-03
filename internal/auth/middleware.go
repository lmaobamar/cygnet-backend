package auth

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"

	"github.com/lmaobamar/cygnet-backend/internal/httpx"
	"github.com/lmaobamar/cygnet-backend/internal/session"
)

type sessionCtxKey struct{}

func SessionInfo(ctx context.Context) (*session.Info, bool) {
	info, ok := ctx.Value(sessionCtxKey{}).(*session.Info)
	return info, ok
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	info, ok := SessionInfo(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return info.UserID, true
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		unauthorized := func(msg string) {
			httpx.WriteJson(w, http.StatusUnauthorized, httpx.ErrorResponse{
				Code: httpx.UnauthorizedError, Message: msg,
			})
		}

		c, err := r.Cookie("session")
		if err != nil {
			unauthorized("not logged in")
			return
		}
		info, err := h.Sessions.Get(r.Context(), c.Value)
		if errors.Is(err, session.ErrNotFound) {
			clearCookie(w, r)
			unauthorized("invalid session")
			return
		}
		if err != nil { // Dragonfly problem: fail closed
			log.Printf("session lookup failed: %v", err)
			httpx.WriteJson(w, http.StatusInternalServerError, httpx.ErrorResponse{
				Code: httpx.InternalError, Message: "something went wrong",
			})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionCtxKey{}, info)))
	})
}
