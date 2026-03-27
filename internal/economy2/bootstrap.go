package economy2

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BootstrapConfig defines the starting kit seeded when a player first colonises a star.
// Loaded from the economy2_bootstrap section of game-params YAML.
type BootstrapConfig struct {
	Stock      map[string]float64  `yaml:"stock"`
	Facilities []BootstrapFacility `yaml:"facilities"`
}

// BootstrapFacility is one pre-built facility in the starting kit.
type BootstrapFacility struct {
	FactoryType   string `yaml:"factory_type"`
	DepositGoodID string `yaml:"deposit_good_id,omitempty"`
}

// BootstrapResult is the JSON response returned by the bootstrap handler.
type BootstrapResult struct {
	NodeID           uuid.UUID          `json:"node_id"`
	SeededStock      map[string]float64 `json:"seeded_stock"`
	SeededFacilities int                `json:"seeded_facilities"`
}

// RunBootstrap seeds a new node with the configured starting stock and facilities.
// All facilities are placed on a planet-level node on the star's home planet,
// so mines sit on the body whose deposits they deplete.
// planet_deposits is lazily initialised from the planet's resource_deposits quality map.
// It is additive — calling it twice gives twice the stock. Idempotency guard belongs
// at the game-logic layer (player state), not here.
func RunBootstrap(ctx context.Context, db *pgxpool.Pool, playerID, starID uuid.UUID, cfg BootstrapConfig) (*BootstrapResult, error) {
	// Find the home planet (first by orbit_index) and ensure deposits exist.
	homePlanetID, err := FindHomePlanet(ctx, db, starID)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: home planet: %w", err)
	}
	if err := EnsureDeposits(ctx, db, *homePlanetID); err != nil {
		return nil, fmt.Errorf("bootstrap: ensure deposits: %w", err)
	}

	// All bootstrap facilities share one planet-level node on the home planet.
	nodeID, err := GetOrCreateNode(ctx, db, playerID, starID, homePlanetID)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: get/create planet node: %w", err)
	}

	for itemID, qty := range cfg.Stock {
		if err := AddToStock(ctx, db, nodeID, itemID, qty); err != nil {
			return nil, fmt.Errorf("bootstrap: seed stock %s: %w", itemID, err)
		}
	}

	for _, fac := range cfg.Facilities {
		f := &Facility{
			PlayerID:    playerID,
			StarID:      starID,
			NodeID:      nodeID,
			FactoryType: fac.FactoryType,
			Status:      "idle",
			Config:      FacilityConfig{Level: 1, DepositGoodID: fac.DepositGoodID},
		}
		if err := CreateFacility(ctx, db, f); err != nil {
			return nil, fmt.Errorf("bootstrap: seed facility %s: %w", fac.FactoryType, err)
		}
	}

	return &BootstrapResult{
		NodeID:           nodeID,
		SeededStock:      cfg.Stock,
		SeededFacilities: len(cfg.Facilities),
	}, nil
}
