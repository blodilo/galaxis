package economy2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FacilityConfig is stored as JSONB in econ2_facilities.config.
type FacilityConfig struct {
	Level          int     `json:"level"`
	TicksRemaining int     `json:"ticks_remaining"`
	EfficiencyAcc  float64 `json:"efficiency_acc"`
}

// Facility is the in-memory representation of an econ2_facilities row.
type Facility struct {
	ID             uuid.UUID
	PlayerID       uuid.UUID
	StarID         uuid.UUID
	PlanetID       *uuid.UUID
	NodeID         uuid.UUID
	FactoryType    string
	Status         string
	Config         FacilityConfig
	CurrentOrderID *uuid.UUID
}

// Destroy cancels all orders, wipes node stock, suspends incoming routes, and marks facility destroyed.
// Per spec: "alles verloren" — no partial rollback.
func (f *Facility) Destroy(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

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
func CreateFacility(ctx context.Context, db *pgxpool.Pool, f *Facility) error {
	cfg, err := json.Marshal(f.Config)
	if err != nil {
		return fmt.Errorf("economy2: marshal facility config: %w", err)
	}
	return db.QueryRow(ctx, `
		INSERT INTO econ2_facilities (player_id, star_id, planet_id, node_id, factory_type, status, config)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id
	`, f.PlayerID, f.StarID, f.PlanetID, f.NodeID, f.FactoryType, f.Status, cfg).Scan(&f.ID)
}

// LoadFacilityByID loads a single facility by primary key.
func LoadFacilityByID(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*Facility, error) {
	var (
		f      Facility
		cfgRaw []byte
	)
	err := db.QueryRow(ctx, `
		SELECT id, player_id, star_id, planet_id, node_id, factory_type, status, config, current_order_id
		FROM econ2_facilities WHERE id=$1
	`, id).Scan(
		&f.ID, &f.PlayerID, &f.StarID, &f.PlanetID, &f.NodeID,
		&f.FactoryType, &f.Status, &cfgRaw, &f.CurrentOrderID,
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
