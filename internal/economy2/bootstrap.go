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
// Mine facilities are placed on the star's first planet; planet_deposits is
// lazily initialised from the planet's resource_deposits quality map if needed.
// It is additive — calling it twice gives twice the stock. Idempotency guard belongs
// at the game-logic layer (player state), not here.
func RunBootstrap(ctx context.Context, db *pgxpool.Pool, playerID, starID uuid.UUID, cfg BootstrapConfig) (*BootstrapResult, error) {
	nodeID, err := GetOrCreateNode(ctx, db, playerID, starID, nil)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: get/create node: %w", err)
	}

	// Resolve home planet for mine placement.
	homePlanetID, err := FindHomePlanet(ctx, db, starID)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: home planet: %w", err)
	}
	if err := EnsureDeposits(ctx, db, *homePlanetID); err != nil {
		return nil, fmt.Errorf("bootstrap: ensure deposits: %w", err)
	}

	for itemID, qty := range cfg.Stock {
		if err := AddToStock(ctx, db, nodeID, itemID, qty); err != nil {
			return nil, fmt.Errorf("bootstrap: seed stock %s: %w", itemID, err)
		}
	}

	for _, fac := range cfg.Facilities {
		var planetID *uuid.UUID
		if fac.FactoryType == "mine" {
			planetID = homePlanetID
		}
		f := &Facility{
			PlayerID:    playerID,
			StarID:      starID,
			PlanetID:    planetID,
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
