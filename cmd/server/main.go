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
	"galaxis/internal/economy"
	"galaxis/internal/economy2"
	"galaxis/internal/jobs"
	"galaxis/internal/tick"

)

func main() {
	configPath  := flag.String("config",      "game-params_v1.6.yaml",                 "Path to game-params YAML")
	migrateOnly := flag.Bool("migrate-only",  false,                                    "Run migrations and exit")
	addr        := flag.String("addr",         ":8080",                                 "HTTP listen address")
	assetsDir   := flag.String("assets-dir",   "assets",                                "Directory to serve under /assets/")
	catalogPath := flag.String("catalog",      "galaxy_morphology_catalog_v1.0.yaml",   "Path to morphology catalog YAML")
	recipesPath := flag.String("recipes",      "econ2_recipes_v1.0.yaml",               "Path to economy2 recipes YAML")
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

	// ── Economy Registries (altes System) ────────────────────────────────────
	reg, err := economy.LoadRegistries("recipes_v1.1.yaml", cfg)
	if err != nil {
		log.Fatalf("economy registries: %v", err)
	}
	log.Printf("economy: loaded %d recipes", len(reg.Recipes))

	// ── Economy2 Recipes (neues System) ──────────────────────────────────────
	recipes, err := economy2.LoadRecipes(*recipesPath)
	if err != nil {
		log.Fatalf("economy2: load recipes: %v", err)
	}
	log.Printf("economy2: loaded %d recipes", len(recipes))

	// ── Tick Engine ───────────────────────────────────────────────────────────
	tickDuration := time.Duration(cfg.Time.StrategyTickMinutes) * time.Minute
	engine := tick.NewEngine(tickDuration)

	// SSE broadcast bus for tick events (altes System).
	bus := economy.NewBroadcaster()

	// Altes Economy-System
	engine.Register(economy.SchedulerHandler(pool, reg))
	engine.Register(economy.ProductionHandler(pool, reg))

	// Neues Economy2-System
	engine.Register(economy2.SchedulerHandler(pool, recipes))
	engine.Register(economy2.ProductionHandler(pool, recipes))
	engine.Register(economy2.ShipTickHandler(pool))

	engine.Start(ctx)
	defer engine.Stop()
	log.Printf("tick engine: started (tick = %v)", tickDuration)

	// ── Job Store ─────────────────────────────────────────────────────────────
	jobStore := jobs.NewStore()

	// ── HTTP Server ───────────────────────────────────────────────────────────
	router := api.NewRouter(pool, cfg, jobStore, *assetsDir, *catalogPath, reg, bus, engine, recipes)
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

