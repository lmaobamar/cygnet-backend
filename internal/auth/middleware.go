package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/lmaobamar/cygnet-backend/internal/httpx"
	"github.com/lmaobamar/cygnet-backend/internal/session"
)

const schemeName = "session"

var SecuritySchemes = map[string]*huma.SecurityScheme{
	schemeName: {Type: "apiKey", In: "cookie", Name: "session"},
}

var Secured = []map[string][]string{{schemeName: {}}}

type sessionKey struct{}

// RequireAuth is Huma operation middleware, attach it to any operation that needs a login:
//
//	middlewares: huma.Middlewares{authH.RequireAuth(api)},
//	security:    auth.Secured,
func (h *Handler) RequireAuth(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		cookie, err := huma.ReadCookie(ctx, "session")
		if err != nil || cookie.Value == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "not logged in")
			return
		}

		info, err := h.Sessions.Get(ctx.Context(), cookie.Value)
		if errors.Is(err, session.ErrNotFound) {
			expired := expiredCookie(ctx.Context()) // tell the browser to drop the dead cookie
			ctx.AppendHeader("Set-Cookie", expired.String())
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid session")
			return
		}
		if err != nil { // Redis problem: fail closed
			huma.WriteErr(api, ctx, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		next(huma.WithValue(ctx, sessionKey{}, info))
	}
}

// SessionFrom reads the session that RequireAuth stored, handlers call this first.
// if the middleware wasn't attached, it returns a 401, so forgetting it fails closed
func SessionFrom(ctx context.Context) (*session.Info, error) {
	info, ok := ctx.Value(sessionKey{}).(*session.Info)
	if !ok || info == nil {
		return nil, httpx.NewProblem(http.StatusUnauthorized, httpx.UnauthorizedError, "not logged in")
	}
	return info, nil
}
