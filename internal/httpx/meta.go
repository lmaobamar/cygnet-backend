package httpx

import (
	"context"
	"net/http"

	"github.com/tomasen/realip"
)

type Meta struct {
	IP    string
	HTTPS bool
}

type metaKey struct{}

func IsHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func WithMeta(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := Meta{
			IP:    realip.FromRequest(r),
			HTTPS: IsHTTPS(r),
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), metaKey{}, m)))
	})
}

func MetaFrom(ctx context.Context) Meta {
	m, _ := ctx.Value(metaKey{}).(Meta)
	return m
}
