package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

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

func (h *Handler) Mount(r chi.Router) {
	r.With(h.Limiter.Limit("signup", ratelimit.PerHour(10))).Post("/signup", httpx.Handle(h.signup))
	r.With(h.Limiter.Limit("login", ratelimit.PerMinute(10))).Post("/login", httpx.Handle(h.login))
	r.Post("/logout", httpx.Handle(h.logout))
	r.Group(func(r chi.Router) {
		r.Use(h.RequireAuth)
		r.Get("/me", httpx.Handle(h.me))
		r.Post("/logout-all", httpx.Handle(h.logoutAll))
		r.Get("/sessions", httpx.Handle(h.listSessions))
		r.Delete("/sessions/{id}", httpx.Handle(h.revokeSession))
	})
}

var handleRe = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

type signupRequest struct {
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"` // optional
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) signup(w http.ResponseWriter, r *http.Request) error {
	var in signupRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.InvalidRequestError, "invalid request body")
	}
	in.Handle = strings.ToLower(strings.TrimSpace(in.Handle))
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	var fields []httpx.FieldError
	if !handleRe.MatchString(in.Handle) {
		fields = append(fields, httpx.FieldError{Field: "handle", Message: "3-32 characters: a-z, 0-9 and _"})
	}
	if utf8.RuneCountInString(in.DisplayName) > 50 {
		fields = append(fields, httpx.FieldError{Field: "display_name", Message: "must be 50 characters or fewer"})
	}
	if in.Email == "" {
		fields = append(fields, httpx.FieldError{Field: "email", Message: "required"})
	}
	if len(in.Password) < 8 {
		fields = append(fields, httpx.FieldError{Field: "password", Message: "must be at least 8 characters"})
	}
	if len(fields) > 0 {
		return httpx.NewError(http.StatusUnprocessableEntity, httpx.ValidationFailedError, "some fields are invalid", fields...)
	}

	user, err := h.Users.Create(r.Context(), in.Handle, in.DisplayName, in.Email, in.Password)
	switch {
	case errors.Is(err, users.ErrHandleTaken):
		return httpx.NewError(http.StatusConflict, httpx.HandleTakenError, "handle taken",
			httpx.FieldError{Field: "handle", Message: "taken"})
	case errors.Is(err, users.ErrEmailTaken):
		return httpx.NewError(http.StatusConflict, httpx.EmailAlreadyInUseError, "email already in use",
			httpx.FieldError{Field: "email", Message: "already in use"})
	case err != nil:
		return err
	}

	if err := h.startSession(w, r, user.ID); err != nil {
		return err
	}
	httpx.WriteJson(w, http.StatusCreated, users.SelfOf(user))
	return nil
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) error {
	var in loginRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.InvalidRequestError, "invalid request body")
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))

	// per account limit, so one account can't be hammered from many IPs
	if ok, retry := h.Limiter.Check(r.Context(), "login_email", email, ratelimit.PerMinute(5)); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
		return httpx.NewError(http.StatusTooManyRequests, httpx.RateLimitedError, "too many attempts, try again shortly")
	}

	user, err := h.Users.Authenticate(r.Context(), email, in.Password)
	switch {
	case errors.Is(err, users.ErrInvalidCredentials):
		return httpx.NewError(http.StatusUnauthorized, httpx.InvalidCredentialsError, "invalid email or password")
	case err != nil:
		return err
	}

	if err := h.startSession(w, r, user.ID); err != nil {
		return err
	}
	httpx.WriteJson(w, http.StatusOK, users.SelfOf(user))
	return nil
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) error {
	if c, err := r.Cookie("session"); err == nil {
		if err := h.Sessions.Delete(r.Context(), c.Value); err != nil {
			return err
		}
	}
	clearCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *Handler) logoutAll(w http.ResponseWriter, r *http.Request) error {
	id, ok := UserID(r.Context())
	if !ok {
		return httpx.NewError(http.StatusUnauthorized, httpx.UnauthorizedError, "not logged in")
	}
	if err := h.Sessions.DeleteAll(r.Context(), id); err != nil {
		return err
	}
	clearCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) error {
	id, ok := UserID(r.Context())
	if !ok {
		return httpx.NewError(http.StatusUnauthorized, httpx.UnauthorizedError, "not logged in")
	}
	user, err := h.Users.ByID(r.Context(), id)
	switch {
	case errors.Is(err, users.ErrNotFound):
		return httpx.NewError(http.StatusUnauthorized, httpx.UnauthorizedError, "invalid session")
	case err != nil:
		return err
	}
	httpx.WriteJson(w, http.StatusOK, users.SelfOf(user))
	return nil
}

type sessionDTO struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Current   bool      `json:"current"`
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) error {
	cur, _ := SessionInfo(r.Context())
	list, err := h.Sessions.List(r.Context(), cur.UserID)
	if err != nil {
		return err
	}
	out := make([]sessionDTO, 0, len(list))
	for _, s := range list {
		out = append(out, sessionDTO{
			ID: s.ID, CreatedAt: s.CreatedAt, LastSeen: s.LastSeen,
			IP: s.IP, UserAgent: s.UserAgent, Current: s.ID == cur.ID,
		})
	}
	httpx.WriteJson(w, http.StatusOK, out)
	return nil
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) error {
	cur, _ := SessionInfo(r.Context())
	err := h.Sessions.DeleteByID(r.Context(), cur.UserID, chi.URLParam(r, "id"))
	if errors.Is(err, session.ErrNotFound) {
		return httpx.NewError(http.StatusNotFound, httpx.InvalidRequestError, "session not found")
	}
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
