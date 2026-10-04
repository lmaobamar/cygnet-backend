package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/lmaobamar/cygnet-backend/internal/httpx"
	"github.com/lmaobamar/cygnet-backend/internal/ratelimit"
	"github.com/lmaobamar/cygnet-backend/internal/session"
	"github.com/lmaobamar/cygnet-backend/internal/users"
)

type Handler struct {
	Users    *users.Service
	Sessions *session.Store
	Limiter  *ratelimit.Limiter
}

func New(u *users.Service, s *session.Store, l *ratelimit.Limiter) *Handler {
	return &Handler{Users: u, Sessions: s, Limiter: l}
}

func (h *Handler) Register(api huma.API) {
	tags := []string{"auth"}
	protected := huma.Middlewares{h.RequireAuth(api)}

	// public
	huma.Register(api, huma.Operation{OperationID: "signup", Method: http.MethodPost, Path: "/api/signup", Tags: tags, DefaultStatus: http.StatusCreated}, h.signup)
	huma.Register(api, huma.Operation{OperationID: "login", Method: http.MethodPost, Path: "/api/login", Tags: tags}, h.login)
	huma.Register(api, huma.Operation{OperationID: "logout", Method: http.MethodPost, Path: "/api/logout", Tags: tags, DefaultStatus: http.StatusNoContent}, h.logout)

	// protected
	huma.Register(api, huma.Operation{OperationID: "me", Method: http.MethodGet, Path: "/api/me", Tags: tags, Middlewares: protected, Security: Secured}, h.me)
	huma.Register(api, huma.Operation{OperationID: "logout-all", Method: http.MethodPost, Path: "/api/logout-all", Tags: tags, DefaultStatus: http.StatusNoContent, Middlewares: protected, Security: Secured}, h.logoutAll)
	huma.Register(api, huma.Operation{OperationID: "list-sessions", Method: http.MethodGet, Path: "/api/sessions", Tags: tags, Middlewares: protected, Security: Secured}, h.listSessions)
	huma.Register(api, huma.Operation{OperationID: "revoke-session", Method: http.MethodDelete, Path: "/api/sessions/{id}", Tags: tags, DefaultStatus: http.StatusNoContent, Middlewares: protected, Security: Secured}, h.revokeSession)
}

func tooMany(retry time.Duration) error {
	return httpx.NewProblem(http.StatusTooManyRequests, httpx.RateLimitedError,
		fmt.Sprintf("too many requests, try again in %d seconds", int(retry.Seconds())+1))
}

type signupIn struct {
	UserAgent string `header:"User-Agent"`
	Session   string `cookie:"session"`
	Body      struct {
		Handle      string `json:"handle" minLength:"3" maxLength:"32" pattern:"^[a-z0-9_]+$" doc:"lowercase letters, digits and underscore"`
		DisplayName string `json:"display_name,omitempty" maxLength:"50" doc:"optional; defaults to the handle"`
		Email       string `json:"email" format:"email" maxLength:"255"`
		Password    string `json:"password" minLength:"8" maxLength:"72" doc:"72 is bcrypt's limit"`
	}
}

type sessionOut struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      users.SelfUser
}

func (h *Handler) signup(ctx context.Context, in *signupIn) (*sessionOut, error) {
	if ok, retry := h.Limiter.Check(ctx, "signup", httpx.MetaFrom(ctx).IP, ratelimit.PerHour(10)); !ok {
		return nil, tooMany(retry)
	}

	user, err := h.Users.Create(ctx, in.Body.Handle, strings.TrimSpace(in.Body.DisplayName),
		strings.ToLower(strings.TrimSpace(in.Body.Email)), in.Body.Password)
	switch {
	case errors.Is(err, users.ErrHandleTaken):
		return nil, httpx.NewProblem(http.StatusConflict, httpx.HandleTakenError, "handle taken",
			httpx.FieldError{Field: "handle", Message: "taken"})
	case errors.Is(err, users.ErrEmailTaken):
		return nil, httpx.NewProblem(http.StatusConflict, httpx.EmailAlreadyInUseError, "email already in use",
			httpx.FieldError{Field: "email", Message: "already in use"})
	case err != nil:
		return nil, err
	}

	cookie, err := h.sessionCookie(ctx, user.ID, in.Session, in.UserAgent)
	if err != nil {
		return nil, err
	}
	return &sessionOut{SetCookie: cookie, Body: users.SelfOf(user)}, nil
}

type loginIn struct {
	UserAgent string `header:"User-Agent"`
	Session   string `cookie:"session"`
	Body      struct {
		HandleOrEmail string `json:"handle_or_email"`
		Password      string `json:"password" minLength:"1" maxLength:"72"`
	}
}

func (h *Handler) login(ctx context.Context, in *loginIn) (*sessionOut, error) {
	handleOrEmail := strings.ToLower(strings.TrimSpace(in.Body.HandleOrEmail))

	if ok, retry := h.Limiter.Check(ctx, "login", httpx.MetaFrom(ctx).IP, ratelimit.PerMinute(10)); !ok {
		return nil, tooMany(retry)
	}
	if ok, retry := h.Limiter.Check(ctx, "login_email", handleOrEmail, ratelimit.PerMinute(5)); !ok {
		return nil, tooMany(retry)
	}

	user, err := h.Users.Authenticate(ctx, handleOrEmail, in.Body.Password)
	switch {
	case errors.Is(err, users.ErrInvalidCredentials):
		return nil, httpx.NewProblem(http.StatusUnauthorized, httpx.InvalidCredentialsError, "invalid email or password")
	case err != nil:
		return nil, err
	}

	cookie, err := h.sessionCookie(ctx, user.ID, in.Session, in.UserAgent)
	if err != nil {
		return nil, err
	}
	return &sessionOut{SetCookie: cookie, Body: users.SelfOf(user)}, nil
}

type cookieIn struct {
	Session string `cookie:"session"`
}

type clearCookieOut struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

func (h *Handler) logout(ctx context.Context, in *cookieIn) (*clearCookieOut, error) {
	if in.Session != "" {
		if err := h.Sessions.Delete(ctx, in.Session); err != nil {
			return nil, err
		}
	}
	return &clearCookieOut{SetCookie: expiredCookie(ctx)}, nil
}

// needs auth below:

func (h *Handler) logoutAll(ctx context.Context, _ *struct{}) (*clearCookieOut, error) {
	info, err := SessionFrom(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.Sessions.DeleteAll(ctx, info.UserID); err != nil {
		return nil, err
	}
	return &clearCookieOut{SetCookie: expiredCookie(ctx)}, nil
}

type meOut struct{ Body users.SelfUser }

func (h *Handler) me(ctx context.Context, _ *struct{}) (*meOut, error) {
	info, err := SessionFrom(ctx)
	if err != nil {
		return nil, err
	}
	user, err := h.Users.ByID(ctx, info.UserID)
	switch {
	case errors.Is(err, users.ErrNotFound):
		return nil, httpx.NewProblem(http.StatusUnauthorized, httpx.UnauthorizedError, "invalid session")
	case err != nil:
		return nil, err
	}
	return &meOut{Body: users.SelfOf(user)}, nil
}

type sessionDTO struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Current   bool      `json:"current"`
}

type sessionsOut struct{ Body []sessionDTO }

func (h *Handler) listSessions(ctx context.Context, _ *struct{}) (*sessionsOut, error) {
	cur, err := SessionFrom(ctx)
	if err != nil {
		return nil, err
	}
	list, err := h.Sessions.List(ctx, cur.UserID)
	if err != nil {
		return nil, err
	}
	out := make([]sessionDTO, 0, len(list))
	for _, s := range list {
		out = append(out, sessionDTO{
			ID: s.ID, CreatedAt: s.CreatedAt, LastSeen: s.LastSeen,
			IP: s.IP, UserAgent: s.UserAgent, Current: s.ID == cur.ID,
		})
	}
	return &sessionsOut{Body: out}, nil
}

type revokeIn struct {
	ID string `path:"id"`
}

func (h *Handler) revokeSession(ctx context.Context, in *revokeIn) (*struct{}, error) {
	cur, err := SessionFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.Sessions.DeleteByID(ctx, cur.UserID, in.ID)
	if errors.Is(err, session.ErrNotFound) {
		return nil, httpx.NewProblem(http.StatusNotFound, httpx.InvalidRequestError, "session not found")
	}
	if err != nil {
		return nil, err
	}
	return nil, nil
}
