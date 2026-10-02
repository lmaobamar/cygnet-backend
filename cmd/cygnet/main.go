package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmaobamar/cygnet-backend/internal/config"
	"github.com/lmaobamar/cygnet-backend/internal/database"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer) // a panic in a handler returns a 500 instead of killing the server

	cfg := config.Get()
	db := database.Connect(cfg.DatabaseURL)
	_ = db

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Println("serving on " + addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
