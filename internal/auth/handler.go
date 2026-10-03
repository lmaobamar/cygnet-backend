package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/lmaobamar/cygnet-backend/internal/httpx"
	"github.com/lmaobamar/cygnet-backend/internal/users"
)

type Handler struct {
	Users     *users.Service
	JWTSecret []byte
}

func New(u *users.Service, secret []byte) *Handler {
	return &Handler{Users: u, JWTSecret: secret}
}

func (h *Handler) Mount(r chi.Router) {
	r.Post("/signup", httpx.Handle(h.signup))
	r.Post("/login", httpx.Handle(h.login))
	r.Post("/logout", httpx.Handle(h.logout))
	r.Group(func(r chi.Router) {
		r.Use(h.RequireAuth)
		r.Get("/me", httpx.Handle(h.me))
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

	if err := h.issueCookie(r, w, user.ID); err != nil {
		return err
	}
	httpx.WriteJson(w, http.StatusCreated, user)
	return nil
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) error {
	var in loginRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.InvalidRequestError, "invalid request body")
	}

	user, err := h.Users.Authenticate(r.Context(), in.Email, in.Password)
	switch {
	case errors.Is(err, users.ErrInvalidCredentials):
		return httpx.NewError(http.StatusUnauthorized, httpx.InvalidCredentialsError, "invalid email or password")
	case err != nil:
		return err
	}

	if err := h.issueCookie(r, w, user.ID); err != nil {
		return err
	}
	httpx.WriteJson(w, http.StatusOK, user)
	return nil
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
	})
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
	httpx.WriteJson(w, http.StatusOK, user)
	return nil
}
