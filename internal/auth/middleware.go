package auth

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/lmaobamar/cygnet-backend/internal/httpx"
)

type ctxKey struct{}

// Lets other packages read who is logged in
func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxKey{}).(uuid.UUID)
	return id, ok
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail := func(msg string) {
			httpx.WriteJson(w, http.StatusUnauthorized, httpx.ErrorResponse{
				Code: httpx.UnauthorizedError, Message: msg,
			})
		}

		c, err := r.Cookie("session")
		if err != nil {
			fail("not logged in")
			return
		}
		claims := &jwt.RegisteredClaims{}
		tok, err := jwt.ParseWithClaims(c.Value, claims,
			func(t *jwt.Token) (any, error) { return h.JWTSecret, nil },
			jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !tok.Valid {
			fail("invalid session")
			return
		}
		id, err := uuid.Parse(claims.Subject)
		if err != nil {
			fail("invalid session")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	})
}
