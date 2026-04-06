package economy2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DeployItem removes one unit of a deployable item from stock and creates
// an active facility at the given node.
// For extractors, an initial continuous extraction order is created automatically.
func DeployItem(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID, nodeID uuid.UUID,
	planetID *uuid.UUID,
	itemID string,
	catalog ItemCatalog,
	recipes RecipeBook,
) (*Facility, error) {
	def, ok := catalog[itemID]
	if !ok {
		return nil, fmt.Errorf("economy2: %q is not a deployable item", itemID)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Consume one unit (available = total − allocated).
	var available float64
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(total - allocated, 0) FROM econ2_item_stock
		 WHERE node_id=$1 AND item_id=$2 FOR UPDATE`,
		nodeID, itemID,
	).Scan(&available); err != nil {
		return nil, fmt.Errorf("economy2: deploy %s: stock query: %w", itemID, err)
	}
	if available < 1 {
		return nil, fmt.Errorf("economy2: deploy %s: not enough in stock (available=%.1f)", itemID, available)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE econ2_item_stock SET total=total-1, updated_at=now()
		 WHERE node_id=$1 AND item_id=$2`,
		nodeID, itemID,
	); err != nil {
		return nil, err
	}

	// Extractors on orbital nodes may have no planet_id — resolve via home planet.
	if def.FactoryType == FactoryTypeExtractor && planetID == nil {
		if pid, err2 := FindHomePlanet(ctx, db, starID); err2 == nil {
			planetID = pid
		}
	}

	cfg := FacilityConfig{
		Level:         def.Level,
		MaxRate:       def.MaxRate,
		DepositGoodID: def.DepositGoodID,
	}
	cfgRaw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	var facilityID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO econ2_facilities (player_id, star_id, node_id, factory_type, status, config)
		VALUES ($1,$2,$3,$4,'idle',$5) RETURNING id
	`, playerID, starID, nodeID, def.FactoryType, cfgRaw).Scan(&facilityID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	f := &Facility{
		ID:          facilityID,
		PlayerID:    playerID,
		StarID:      starID,
		NodeID:      nodeID,
		FactoryType: def.FactoryType,
		Status:      "idle",
		Config:      cfg,
		PlanetID:    planetID,
	}

	// Auto-start extraction for newly deployed extractors.
	if def.FactoryType == FactoryTypeExtractor && def.DepositGoodID != "" {
		_ = createExtractionOrder(ctx, db, playerID, starID, nodeID, def.DepositGoodID, recipes)
	}

	return f, nil
}
