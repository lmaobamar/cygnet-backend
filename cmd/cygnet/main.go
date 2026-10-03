package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmaobamar/cygnet-backend/internal/auth"
	"github.com/lmaobamar/cygnet-backend/internal/config"
	"github.com/lmaobamar/cygnet-backend/internal/database"
	"github.com/lmaobamar/cygnet-backend/internal/users"
)

func main() {
	loadStart := time.Now()
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer) // a panic in a handler returns a 500 instead of killing the server

	cfg := config.Get()

	// infra
	db := database.Connect(cfg.DatabaseURL)
	// todo: redis

	// svcs
	usersSvc := users.NewService(db)

	// handlers
	authH := auth.New(usersSvc, []byte(cfg.JWTSecret))

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		authH.Mount(r)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("ready on %v after %v!", addr, time.Since(loadStart))
	log.Fatal(http.ListenAndServe(addr, r))
}
