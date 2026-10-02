package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lmaobamar/cygnet-backend/internal/httpx"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	DB        *gorm.DB
	JWTSecret []byte
}

func New(db *gorm.DB, secret []byte) *Handler {
	return &Handler{DB: db, JWTSecret: secret}
}

func (h *Handler) Mount(r chi.Router) {
	// r.post "signup", "login", "logout"
	// then r.group with r.Use requireAuth with r.get "me"
	r.Post("/signup", h.signup)
}

type credentials struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) signup(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}

	in.Username = strings.TrimSpace(in.Username)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Username == "" || in.Email == "" || len(in.Password) < 8 {
		httpx.WriteError(w, http.StatusBadRequest, "username, email and a password of 8+ characters are required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "bcrypt fail")
		return
	}

	// user := models.User{Username: in.Username, Email: in.Email, PasswordHash: string(hash)}
	_ = hash // TODO: finish this off. thats just to make the compiler happy
}
