// Package api wires up the HTTP router and all handlers.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"galaxis/internal/config"
	"galaxis/internal/jobs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter creates and returns the chi router with all middleware and routes.
func NewRouter(db *pgxpool.Pool, cfg *config.Config, store *jobs.Store, assetsDir, catalogPath string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS: allow the Vite dev server on both default and configured ports
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Static assets (morphology images etc.)
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))

	r.Get("/health", healthHandler(db))

	r.Route("/api/v1", func(r chi.Router) {
		registerGalaxyRoutes(r, db, cfg, store)
		registerCatalogRoutes(r, cfg, catalogPath)
		registerGenerateRoutes(r, db, cfg, store, assetsDir, catalogPath)
	})

	return r
}

func registerCatalogRoutes(r chi.Router, cfg *config.Config, catalogPath string) {
	r.Get("/catalog/morphologies", listMorphologies(catalogPath))
	r.Get("/params/defaults", getDefaultParams(cfg))
}

func registerGenerateRoutes(r chi.Router, pool *pgxpool.Pool, cfg *config.Config, store *jobs.Store, assetsDir, catalogPath string) {
	r.Post("/generate", triggerGenerate(pool, cfg, store))
	r.Post("/generate/step1", triggerStep1(pool, cfg, store, assetsDir, catalogPath))
	r.Get("/generate/{jobID}/status", getGenerateStatus(store))
	// SSE endpoint: the handler itself bypasses the global 60s timeout via a
	// deadline-free context that still tracks client disconnects.
	r.Get("/generate/{jobID}/progress", getGenerateProgress(store))
}

func healthHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "degraded", "database": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
