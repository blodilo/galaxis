package economy2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// depositState mirrors the per-good entry in planet_deposits.state.
// In economy2: max_rate is treated as max_mines (max simultaneous mine facilities).
type depositState struct {
	Remaining float64 `json:"remaining"`
	MaxRate   float64 `json:"max_rate"` // = max_mines in economy2 context
}

// readDeposit loads the deposit state for a single good on a planet.
func readDeposit(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID, goodID string) (depositState, error) {
	var raw []byte
	if err := db.QueryRow(ctx,
		`SELECT state FROM planet_deposits WHERE planet_id = $1`,
		planetID,
	).Scan(&raw); err != nil {
		return depositState{}, fmt.Errorf("economy2: read deposit planet %s: %w", planetID, err)
	}

	var state map[string]depositState
	if err := json.Unmarshal(raw, &state); err != nil {
		return depositState{}, fmt.Errorf("economy2: parse deposit state: %w", err)
	}

	ds, ok := state[goodID]
	if !ok {
		return depositState{}, fmt.Errorf("economy2: good %q not found in deposits for planet %s", goodID, planetID)
	}
	return ds, nil
}

// ReadAllDeposits returns the full deposit state map for a planet.
// Used by the API to display current resource levels in the frontend.
func ReadAllDeposits(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID) (map[string]depositState, error) {
	var raw []byte
	if err := db.QueryRow(ctx,
		`SELECT state FROM planet_deposits WHERE planet_id = $1`,
		planetID,
	).Scan(&raw); err != nil {
		if err == pgx.ErrNoRows {
			return map[string]depositState{}, nil
		}
		return nil, fmt.Errorf("economy2: read deposits planet %s: %w", planetID, err)
	}
	var state map[string]depositState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("economy2: parse deposit state: %w", err)
	}
	return state, nil
}

// depleteDeposit subtracts qty from the remaining amount and persists the result.
// Returns the new remaining value and whether the deposit is now empty.
// Uses read-modify-write (safe: single server instance per player per spec).
func depleteDeposit(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID, goodID string, qty float64) (remaining float64, depleted bool, err error) {
	var raw []byte
	if err := db.QueryRow(ctx,
		`SELECT state FROM planet_deposits WHERE planet_id = $1`,
		planetID,
	).Scan(&raw); err != nil {
		return 0, false, fmt.Errorf("economy2: read deposit for deplete: %w", err)
	}

	// Use RawMessage to preserve unknown fields (slots, survey_quality, …).
	var state map[string]json.RawMessage
	if err := json.Unmarshal(raw, &state); err != nil {
		return 0, false, fmt.Errorf("economy2: parse deposit state: %w", err)
	}

	var ds depositState
	if err := json.Unmarshal(state[goodID], &ds); err != nil {
		return 0, false, fmt.Errorf("economy2: parse deposit entry %q: %w", goodID, err)
	}

	ds.Remaining -= qty
	if ds.Remaining < 0 {
		ds.Remaining = 0
	}
	depleted = ds.Remaining <= 0

	updated, _ := json.Marshal(ds)
	state[goodID] = updated

	finalRaw, _ := json.Marshal(state)
	_, err = db.Exec(ctx,
		`UPDATE planet_deposits SET state = $1, updated_at = now() WHERE planet_id = $2`,
		finalRaw, planetID,
	)
	return ds.Remaining, depleted, err
}

// countActiveMines counts mine facilities currently active on a given deposit.
// Used to enforce the max_mines slot limit before assigning a new mine order.
func countActiveMines(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID, goodID string) (int, error) {
	var count int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM econ2_facilities
		WHERE planet_id = $1
		  AND factory_type = 'mine'
		  AND config->>'deposit_good_id' = $2
		  AND status NOT IN ('idle','destroyed')
	`, planetID, goodID).Scan(&count)
	return count, err
}

// FindHomePlanet returns the id of the first (lowest orbit_index) planet for the given star.
// Mine facilities are placed on this planet when no explicit planet_id is provided.
func FindHomePlanet(ctx context.Context, db *pgxpool.Pool, starID uuid.UUID) (*uuid.UUID, error) {
	var planetID uuid.UUID
	err := db.QueryRow(ctx,
		`SELECT id FROM planets WHERE star_id = $1 ORDER BY orbit_index ASC LIMIT 1`,
		starID,
	).Scan(&planetID)
	if err != nil {
		return nil, fmt.Errorf("economy2: find home planet for star %s: %w", starID, err)
	}
	return &planetID, nil
}

// EnsureDeposits lazily initialises planet_deposits for planetID if it does not yet exist.
// It reads the planet's resource_deposits quality map (0–1 per good) and scales to
// initial amounts:  remaining = quality × 50 000,  max_rate = max(1, quality × 5).
// The scaling constants match game-params_v1.8.yaml:
//
//	deposits.common_deposit_units = 50_000
//	mine.base_rate × level_multiplier[Lv1] = 5.0 × 0.5 → max_rate ≤ 5 for quality=1
func EnsureDeposits(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID) error {
	// Fast path: row already exists.
	var exists bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM planet_deposits WHERE planet_id=$1)`, planetID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("economy2: ensure deposits check: %w", err)
	}
	if exists {
		return nil
	}

	// Read quality map from planet.
	var raw []byte
	if err := db.QueryRow(ctx,
		`SELECT resource_deposits FROM planets WHERE id=$1`, planetID,
	).Scan(&raw); err != nil {
		return fmt.Errorf("economy2: read planet resource_deposits %s: %w", planetID, err)
	}

	var qualities map[string]float64
	if err := json.Unmarshal(raw, &qualities); err != nil {
		return fmt.Errorf("economy2: parse resource_deposits: %w", err)
	}

	state := make(map[string]depositState, len(qualities))
	for goodID, q := range qualities {
		if q <= 0 {
			continue
		}
		maxRate := q * 5.0
		if maxRate < 1.0 {
			maxRate = 1.0
		}
		state[goodID] = depositState{
			Remaining: q * 50_000.0,
			MaxRate:   maxRate,
		}
	}

	stateRaw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx,
		`INSERT INTO planet_deposits (planet_id, state)
		 VALUES ($1, $2)
		 ON CONFLICT (planet_id) DO NOTHING`,
		planetID, stateRaw,
	)
	return err
}
