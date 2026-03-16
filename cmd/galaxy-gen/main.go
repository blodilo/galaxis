// galaxy-gen generates a Galaxis galaxy and writes it to PostgreSQL.
// Run once before starting a game session.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"galaxis/internal/config"
	"galaxis/internal/db"
	"galaxis/internal/galaxy"
)

func main() {
	configPath := flag.String("config", "game-params_v1.2.yaml", "Path to game-params YAML")
	name := flag.String("name", "", "Galaxy name (default: instance name from config)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	galaxyName := cfg.Server.InstanceName
	if *name != "" {
		galaxyName = *name
	}

	ctx := context.Background()

	// Run migrations before generating
	log.Println("running database migrations...")
	if err := db.Migrate(cfg.DatabaseURL, "file://migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	// Serialize config for audit trail in DB
	cfgJSON, _ := json.Marshal(cfg.Galaxy)

	galaxyID, err := db.CreateGalaxy(ctx, pool, galaxyName, cfg.Galaxy.Seed, cfgJSON)
	if err != nil {
		log.Fatalf("create galaxy: %v", err)
	}
	fmt.Printf("galaxy-gen: created galaxy %s (id=%s)\n", galaxyName, galaxyID)
	fmt.Printf("galaxy-gen: seed=%d  stars=%d  radius=%.0f ly\n",
		cfg.Galaxy.Seed, cfg.Galaxy.NumStars, cfg.Galaxy.RadiusLY)

	gen := galaxy.NewGenerator(cfg, pool)
	if err := gen.Run(ctx, galaxyID); err != nil {
		// Mark galaxy as failed so the UI can show an error state
		_ = db.SetGalaxyStatus(ctx, pool, galaxyID, "error")
		log.Fatalf("generation failed: %v", err)
	}

	fmt.Printf("galaxy-gen: done — galaxy %s is ready\n", galaxyID)
	os.Exit(0)
}
