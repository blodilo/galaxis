package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"galaxis/internal/api"
	"galaxis/internal/config"
	"galaxis/internal/db"
	"galaxis/internal/tick"
)

func main() {
	configPath := flag.String("config", "game-params_v1.0.yaml", "Path to game-params YAML")
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations and exit")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// ── Migrations ────────────────────────────────────────────────────────────
	log.Println("running database migrations...")
	if err := db.Migrate(cfg.DatabaseURL, "file://migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrations: ok")

	if *migrateOnly {
		fmt.Println("--migrate-only: done")
		os.Exit(0)
	}

	// ── DB Pool ───────────────────────────────────────────────────────────────
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	log.Println("database: connected")

	// ── Tick Engine ───────────────────────────────────────────────────────────
	tickDuration := time.Duration(cfg.Time.StrategyTickMinutes) * time.Minute
	engine := tick.NewEngine(tickDuration)
	engine.Start(ctx)
	defer engine.Stop()
	log.Printf("tick engine: started (tick = %v)", tickDuration)

	// ── HTTP Server ───────────────────────────────────────────────────────────
	router := api.NewRouter(pool)
	srv := &http.Server{
		Addr:         *addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server: listening on %s", *addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-quit
	log.Println("server: shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server: forced shutdown: %v", err)
	}
	log.Println("server: stopped")
}
