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
	"galaxis/internal/bus"
	"galaxis/internal/bus/inprocbus"
	"galaxis/internal/bus/natsbus"
	"galaxis/internal/config"
	"galaxis/internal/db"
	"galaxis/internal/economy"
	"galaxis/internal/economy2"
	"galaxis/internal/jobs"
	"galaxis/internal/tick"

	"gopkg.in/yaml.v3"
)

func main() {
	configPath  := flag.String("config",      "game-params_v1.6.yaml",                 "Path to game-params YAML")
	migrateOnly := flag.Bool("migrate-only",  false,                                    "Run migrations and exit")
	addr        := flag.String("addr",         ":8080",                                 "HTTP listen address")
	assetsDir   := flag.String("assets-dir",   "assets",                                "Directory to serve under /assets/")
	catalogPath := flag.String("catalog",      "galaxy_morphology_catalog_v1.0.yaml",   "Path to morphology catalog YAML")
	recipesPath := flag.String("recipes",      "econ2_recipes_v1.0.yaml",               "Path to economy2 recipes YAML")
	natsURL     := flag.String("nats",         "",                                       "NATS URL (e.g. nats://localhost:4222); empty = in-process bus")
	natsWsURL   := flag.String("nats-ws",      "ws://localhost:4223",                    "NATS WebSocket URL returned by /api/v1/auth/nats-token")
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

	// ── Economy2 Mine Params ──────────────────────────────────────────────────
	mineParams, err := loadMineParams(*configPath)
	if err != nil {
		log.Fatalf("economy2: load mine params: %v", err)
	}
	log.Printf("economy2: mine base_rate=%.1f levels=%d", mineParams.BaseRate, len(mineParams.LevelMultiplier))

	// ── Economy2 Bootstrap Config ─────────────────────────────────────────────
	bootstrapCfg, err := loadBootstrapConfig(*configPath)
	if err != nil {
		log.Fatalf("economy2: load bootstrap config: %v", err)
	}
	log.Printf("economy2: bootstrap kit: %d items, %d facilities", len(bootstrapCfg.Stock), len(bootstrapCfg.Facilities))

	// ── Message Bus ───────────────────────────────────────────────────────────
	var msgBus bus.Bus
	if *natsURL != "" {
		nb, err := natsbus.New(*natsURL)
		if err != nil {
			log.Fatalf("nats: %v", err)
		}
		defer func() { _ = nb.Close() }()
		log.Printf("nats: connected to %s", *natsURL)
		msgBus = nb
	} else {
		msgBus = inprocbus.New()
		log.Println("nats: no URL given — using in-process bus")
	}

	// Ensure JetStream streams (idempotent — safe to run on every start)
	streamDefs := []bus.StreamConfig{
		{Name: "TICK",    Subjects: []string{"galaxis.tick.>"}, MaxAge: 7 * 24 * time.Hour},
		{Name: "ECONOMY", Subjects: []string{"galaxis.economy.>"}, MaxAge: 7 * 24 * time.Hour},
		{Name: "COMBAT",  Subjects: []string{"galaxis.combat.*.state"}, MaxAge: 24 * time.Hour},
		{Name: "PLAYER",  Subjects: []string{"galaxis.player.>"}, MaxAge: 30 * 24 * time.Hour},
	}
	for _, sc := range streamDefs {
		if err := msgBus.EnsureStream(ctx, sc); err != nil {
			log.Printf("bus: EnsureStream %s: %v (inprocbus silently ignores unknown streams for Tier-1)", sc.Name, err)
		}
	}

	// ── Tick Engine ───────────────────────────────────────────────────────────
	tickDuration := time.Duration(cfg.Time.StrategyTickMinutes) * time.Minute
	engine := tick.NewEngine(tickDuration)

	// SSE broadcast bus for tick events (altes System — bleibt bis Frontend auf NATS migriert).
	sseBus := economy.NewBroadcaster()

	// Altes Economy-System
	engine.Register(economy.SchedulerHandler(pool, reg))
	engine.Register(economy.ProductionHandler(pool, reg))

	// Neues Economy2-System
	engine.Register(economy2.SchedulerHandler(pool, recipes))
	engine.Register(economy2.BuildTickHandler(pool, recipes))
	engine.Register(economy2.ProductionHandler(pool, recipes, mineParams))
	engine.Register(economy2.ShipTickHandler(pool))

	// Tick-Event auf Bus publizieren (letzter Handler — nach allen Economy-Updates)
	engine.Register(tickAdvancePublisher(msgBus))

	engine.Start(ctx)
	defer engine.Stop()
	log.Printf("tick engine: started (tick = %v)", tickDuration)

	// ── Job Store ─────────────────────────────────────────────────────────────
	jobStore := jobs.NewStore()

	// ── HTTP Server ───────────────────────────────────────────────────────────
	router := api.NewRouter(pool, cfg, jobStore, *assetsDir, *catalogPath, reg, sseBus, engine, recipes, bootstrapCfg, mineParams, *natsWsURL)
	srv := &http.Server{
		Addr:        *addr,
		Handler:     router,
		ReadTimeout: 15 * time.Second,
		WriteTimeout: 0, // 0 = kein Limit — nötig für SSE-Streams (planet gen, economy events)
		IdleTimeout: 60 * time.Second,
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

// tickAdvancePublisher returns a tick.Handler that publishes galaxis.tick.advance (Tier 2).
func tickAdvancePublisher(b bus.Bus) tick.Handler {
	return func(ctx context.Context, tickN int64) {
		payload := fmt.Appendf(nil, `{"tick":%d,"ts":"%s"}`, tickN, time.Now().UTC().Format(time.RFC3339))
		if err := b.PublishDurable(ctx, "TICK", bus.Message{
			Subject: "galaxis.tick.advance",
			Payload: payload,
		}); err != nil {
			log.Printf("bus: publish tick.advance #%d: %v", tickN, err)
		}
	}
}

// loadBootstrapConfig reads the economy2_bootstrap: section from the game-params YAML file.
func loadBootstrapConfig(path string) (economy2.BootstrapConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return economy2.BootstrapConfig{}, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg struct {
		Bootstrap economy2.BootstrapConfig `yaml:"economy2_bootstrap"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return economy2.BootstrapConfig{}, fmt.Errorf("parse bootstrap config: %w", err)
	}
	return cfg.Bootstrap, nil
}

// loadMineParams reads the mine: section from the game-params YAML file.
func loadMineParams(path string) (economy2.MineParams, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return economy2.MineParams{}, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg struct {
		Mine economy2.MineParams `yaml:"mine"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return economy2.MineParams{}, fmt.Errorf("parse mine params: %w", err)
	}
	if cfg.Mine.BaseRate <= 0 {
		cfg.Mine.BaseRate = 5.0
	}
	if len(cfg.Mine.LevelMultiplier) == 0 {
		cfg.Mine.LevelMultiplier = []float64{0.5}
	}
	return cfg.Mine, nil
}
