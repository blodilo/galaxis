package economy2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Factory type constants — use these instead of raw strings to get compile-time safety.
const (
	FactoryTypeExtractor        = "extractor"
	FactoryTypeRefinery         = "refinery"
	FactoryTypePlant            = "plant"
	FactoryTypeAssemblyPlant    = "assembly_plant"
	FactoryTypeConstructionYard = "construction_yard"
)

// FacilityConfig is stored as JSONB in econ2_facilities.config.
type FacilityConfig struct {
	Level          int     `json:"level"`
	// MaxRate is the extraction rate (units/tick) for mine facilities.
	// Actual output = MaxRate × deposit.Quality.
	// Upgradeable via tech; set to game-params mine.base_max_rate on build. [BALANCING]
	MaxRate        float64 `json:"max_rate,omitempty"`
	TicksRemaining int     `json:"ticks_remaining"`
	EfficiencyAcc  float64 `json:"efficiency_acc"`
	// DepositGoodID is set for mine facilities and names the good being extracted.
	DepositGoodID string `json:"deposit_good_id,omitempty"`
}

// Facility is the in-memory representation of an econ2_facilities row.
// PlanetID is NOT a DB column on this table — it is populated by joining
// with econ2_nodes when the node's location is needed (e.g. processMine).
type Facility struct {
	ID             uuid.UUID      `json:"id"`
	PlayerID       uuid.UUID      `json:"player_id"`
	StarID         uuid.UUID      `json:"star_id"`
	NodeID         uuid.UUID      `json:"node_id"`
	FactoryType    string         `json:"factory_type"`
	Status         string         `json:"status"`
	Config         FacilityConfig `json:"config"`
	CurrentOrderID *uuid.UUID     `json:"current_order_id"`
	// PlanetID is derived from the node (populated via JOIN, not stored on facilities).
	PlanetID *uuid.UUID `json:"planet_id,omitempty"`
}

// Destroy cancels all orders, wipes node stock, suspends incoming routes, and marks facility destroyed.
// Per spec: "alles verloren" — no partial rollback.
func (f *Facility) Destroy(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Cancel all active orders for this facility
	if _, err := tx.Exec(ctx, `
		UPDATE econ2_orders
		SET status='cancelled', updated_at=now()
		WHERE facility_id=$1 AND status NOT IN ('completed','cancelled')
	`, f.ID); err != nil {
		return err
	}

	// Destroy facility
	if _, err := tx.Exec(ctx, `
		UPDATE econ2_facilities
		SET status='destroyed', current_order_id=NULL, updated_at=now()
		WHERE id=$1
	`, f.ID); err != nil {
		return err
	}

	// Wipe all stock in this facility's node (total + allocated)
	if _, err := tx.Exec(ctx,
		`DELETE FROM econ2_item_stock WHERE node_id=$1`, f.NodeID,
	); err != nil {
		return err
	}

	// Suspend all routes delivering to this node
	if _, err := tx.Exec(ctx, `
		UPDATE econ2_routes SET status='suspended', updated_at=now()
		WHERE to_node_id=$1
	`, f.NodeID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// CreateFacility inserts a new facility row and sets f.ID.
// The facility's physical location is determined by its node (econ2_nodes.planet_id / moon_id),
// not by a column on this table.
func CreateFacility(ctx context.Context, db *pgxpool.Pool, f *Facility) error {
	cfg, err := json.Marshal(f.Config)
	if err != nil {
		return fmt.Errorf("economy2: marshal facility config: %w", err)
	}
	return db.QueryRow(ctx, `
		INSERT INTO econ2_facilities (player_id, star_id, node_id, factory_type, status, config)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id
	`, f.PlayerID, f.StarID, f.NodeID, f.FactoryType, f.Status, cfg).Scan(&f.ID)
}

// LoadFacilityByID loads a single facility by primary key.
// PlanetID is populated via JOIN with econ2_nodes.
func LoadFacilityByID(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*Facility, error) {
	var (
		f      Facility
		cfgRaw []byte
	)
	err := db.QueryRow(ctx, `
		SELECT f.id, f.player_id, f.star_id, f.node_id, f.factory_type, f.status, f.config, f.current_order_id,
		       n.planet_id
		FROM econ2_facilities f
		JOIN econ2_nodes n ON n.id = f.node_id
		WHERE f.id=$1
	`, id).Scan(
		&f.ID, &f.PlayerID, &f.StarID, &f.NodeID,
		&f.FactoryType, &f.Status, &cfgRaw, &f.CurrentOrderID,
		&f.PlanetID,
	)
	if err != nil {
		return nil, fmt.Errorf("economy2: load facility %s: %w", id, err)
	}
	if err := json.Unmarshal(cfgRaw, &f.Config); err != nil {
		return nil, fmt.Errorf("economy2: facility config %s: %w", id, err)
	}
	return &f, nil
}

// saveFacilityConfig persists the config JSONB for a facility.
func saveFacilityConfig(ctx context.Context, db *pgxpool.Pool, f *Facility) error {
	raw, err := json.Marshal(f.Config)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx,
		`UPDATE econ2_facilities SET config=$1, updated_at=now() WHERE id=$2`,
		raw, f.ID,
	)
	return err
}
