package economy

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DepositState is the runtime representation of a single resource deposit
// inside planet_deposits.state JSONB.
type DepositState struct {
	Remaining     float64 `json:"remaining"`
	MaxRate       float64 `json:"max_rate"`
	Slots         int     `json:"slots"`
	SurveyQuality float64 `json:"survey_quality"`
}

// PlanetDeposits represents the full planet_deposits row.
type PlanetDeposits struct {
	PlanetID uuid.UUID
	// State maps good_id → DepositState.
	State map[string]DepositState
}

// InitDeposits creates a planet_deposits row from the planet's resource_deposits
// JSONB field (which holds quality values keyed by good_id) and the DepositRegistry.
// If the row already exists it is left unchanged (lazy init: first survey wins).
// Returns the initialised PlanetDeposits (which may have been read from DB if it
// already existed).
func InitDeposits(
	ctx context.Context,
	db *pgxpool.Pool,
	planetID uuid.UUID,
	resourceQualities map[string]float64, // from planets.resource_deposits
	reg DepositRegistry,
) (*PlanetDeposits, error) {
	// Try to read existing row first.
	pd, err := GetDeposits(ctx, db, planetID)
	if err == nil {
		return pd, nil // already initialised
	}
	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("deposit: read existing: %w", err)
	}

	// Build initial state from quality values.
	state := make(map[string]DepositState, len(resourceQualities))
	for goodID, quality := range resourceQualities {
		spec, ok := reg[goodID]
		if !ok {
			continue // good not in registry (e.g. organics or water_ice without spec)
		}
		slots := int(math.Round(float64(spec.BaseSlots) * quality))
		if slots < 1 {
			slots = 1
		}
		state[goodID] = DepositState{
			Remaining:     spec.BaseUnits * quality,
			MaxRate:       spec.BaseMaxRate * (0.5 + quality*0.5),
			Slots:         slots,
			SurveyQuality: quality,
		}
	}

	raw, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("deposit: marshal state: %w", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO planet_deposits (planet_id, state)
		 VALUES ($1, $2)
		 ON CONFLICT (planet_id) DO NOTHING`,
		planetID, raw,
	)
	if err != nil {
		return nil, fmt.Errorf("deposit: insert: %w", err)
	}

	// Re-read to get any concurrent insert that won the race.
	return GetDeposits(ctx, db, planetID)
}

// GetDeposits loads the planet_deposits row for the given planet.
// Returns pgx.ErrNoRows if not yet initialised.
func GetDeposits(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID) (*PlanetDeposits, error) {
	row := db.QueryRow(ctx,
		`SELECT state FROM planet_deposits WHERE planet_id = $1`, planetID,
	)

	var raw []byte
	if err := row.Scan(&raw); err != nil {
		return nil, err
	}

	var state map[string]DepositState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("deposit: unmarshal state: %w", err)
	}

	return &PlanetDeposits{PlanetID: planetID, State: state}, nil
}

// Deplete reduces the remaining amount of goodID by qty and saves the updated
// state to DB.  Returns (depleted bool, err).
// depleted is true when remaining has reached 0 after this deduction.
func Deplete(
	ctx context.Context,
	db *pgxpool.Pool,
	planetID uuid.UUID,
	goodID string,
	qty float64,
) (depleted bool, err error) {
	pd, err := GetDeposits(ctx, db, planetID)
	if err != nil {
		return false, fmt.Errorf("deposit: deplete read: %w", err)
	}

	ds, ok := pd.State[goodID]
	if !ok {
		return false, fmt.Errorf("deposit: good %q not found on planet %s", goodID, planetID)
	}

	ds.Remaining -= qty
	if ds.Remaining < 0 {
		ds.Remaining = 0
	}
	depleted = ds.Remaining <= 0
	pd.State[goodID] = ds

	if err := saveDeposits(ctx, db, pd); err != nil {
		return false, err
	}
	return depleted, nil
}

// saveDeposits persists the full state JSONB back to planet_deposits.
func saveDeposits(ctx context.Context, db *pgxpool.Pool, pd *PlanetDeposits) error {
	raw, err := json.Marshal(pd.State)
	if err != nil {
		return fmt.Errorf("deposit: marshal: %w", err)
	}
	_, err = db.Exec(ctx,
		`UPDATE planet_deposits SET state = $1, updated_at = now() WHERE planet_id = $2`,
		raw, pd.PlanetID,
	)
	return err
}
