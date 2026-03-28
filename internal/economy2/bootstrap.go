package economy2

import (
	"context"
	"fmt"
	"log"

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
// For mine facilities it also auto-creates a continuous mine order so that mines
// start producing immediately without manual order creation.
func RunBootstrap(ctx context.Context, db *pgxpool.Pool, playerID, starID uuid.UUID, cfg BootstrapConfig, recipes RecipeBook) (*BootstrapResult, error) {
	homePlanetID, err := FindHomePlanet(ctx, db, starID)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: home planet: %w", err)
	}
	if err := EnsureDeposits(ctx, db, *homePlanetID); err != nil {
		return nil, fmt.Errorf("bootstrap: ensure deposits: %w", err)
	}

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

		// Auto-create continuous mine order so the mine starts working immediately.
		if fac.FactoryType == "mine" && fac.DepositGoodID != "" {
			if err := createMineOrder(ctx, db, playerID, starID, nodeID, fac.DepositGoodID, recipes); err != nil {
				log.Printf("economy2: bootstrap auto-order for %s: %v", fac.DepositGoodID, err)
			}
		}
	}

	return &BootstrapResult{
		NodeID:           nodeID,
		SeededStock:      cfg.Stock,
		SeededFacilities: len(cfg.Facilities),
	}, nil
}

// createMineOrder creates a continuous mine production order for the given deposit good.
// Mine orders have no goods-storage inputs, so AllocateOrder makes them ready immediately.
// Used by both bootstrap and finishBuildOrder to ensure newly created mines start working.
func createMineOrder(ctx context.Context, db *pgxpool.Pool, playerID, starID, nodeID uuid.UUID, goodID string, recipes RecipeBook) error {
	key := RecipeKey{ProductID: goodID, FactoryType: "mine"}
	recipe, ok := recipes[key]
	if !ok {
		return fmt.Errorf("economy2: no mine recipe for good %q", goodID)
	}
	order := &ProductionOrder{
		PlayerID:        playerID,
		StarID:          starID,
		NodeID:          nodeID,
		OrderType:       OrderTypeContinuous,
		Status:          OrderStatusPending,
		RecipeID:        recipe.RecipeID,
		ProductID:       recipe.ProductID,
		FactoryType:     recipe.FactoryType,
		Inputs:          recipe.Inputs,
		BaseYield:       recipe.BaseYield,
		RecipeTicks:     recipe.Ticks,
		Efficiency:      recipe.Efficiency,
		TargetQty:       999_999,
		AllocatedInputs: map[string]float64{},
		Priority:        8,
	}
	if err := CreateOrder(ctx, db, order); err != nil {
		return err
	}
	// Mine orders have no inputs → immediate allocation → status becomes ready.
	_ = AllocateOrder(ctx, db, nodeID, order, map[string]float64{})
	return nil
}
