package economy2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// depositState mirrors model.DepositEntry — the per-good entry in planets.resource_deposits.
// Stored as JSONB directly on the planets row (no separate planet_deposits table).
type depositState struct {
	Amount   float64 `json:"amount"`   // current extractable stock
	Quality  float64 `json:"quality"`  // geological modifier 0–1 (static)
	MaxMines int     `json:"max_mines"` // max simultaneous mine facilities
}

// readDeposit loads the deposit state for a single good on a planet.
// Reads directly from planets.resource_deposits JSONB.
func readDeposit(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID, goodID string) (depositState, error) {
	var raw []byte
	if err := db.QueryRow(ctx,
		`SELECT resource_deposits->$2 FROM planets WHERE id=$1`,
		planetID, goodID,
	).Scan(&raw); err != nil {
		return depositState{}, fmt.Errorf("economy2: read deposit planet %s good %s: %w", planetID, goodID, err)
	}
	if raw == nil {
		return depositState{}, fmt.Errorf("economy2: deposit %q not found on planet %s", goodID, planetID)
	}
	var ds depositState
	if err := json.Unmarshal(raw, &ds); err != nil {
		return depositState{}, fmt.Errorf("economy2: parse deposit state: %w", err)
	}
	return ds, nil
}

// ReadAllDeposits returns the full deposit state map for a planet.
// Used by the API to display current resource levels in the frontend.
func ReadAllDeposits(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID) (map[string]depositState, error) {
	var raw []byte
	if err := db.QueryRow(ctx,
		`SELECT resource_deposits FROM planets WHERE id=$1`, planetID,
	).Scan(&raw); err != nil {
		if err == pgx.ErrNoRows {
			return map[string]depositState{}, nil
		}
		return nil, fmt.Errorf("economy2: read deposits planet %s: %w", planetID, err)
	}
	var state map[string]depositState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("economy2: parse deposits: %w", err)
	}
	if state == nil {
		state = map[string]depositState{}
	}
	return state, nil
}

// depleteDeposit subtracts qty from the deposit amount and persists the result.
// Returns the new amount and whether the deposit is now empty.
// Uses read-modify-write (safe: single server instance per spec).
func depleteDeposit(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID, goodID string, qty float64) (remaining float64, depleted bool, err error) {
	var raw []byte
	if err := db.QueryRow(ctx,
		`SELECT resource_deposits FROM planets WHERE id=$1`, planetID,
	).Scan(&raw); err != nil {
		return 0, false, fmt.Errorf("economy2: read deposits for deplete: %w", err)
	}

	// Use RawMessage to preserve all fields (quality, max_mines, etc.) during partial update.
	var state map[string]json.RawMessage
	if err := json.Unmarshal(raw, &state); err != nil {
		return 0, false, fmt.Errorf("economy2: parse deposits: %w", err)
	}

	var ds depositState
	if err := json.Unmarshal(state[goodID], &ds); err != nil {
		return 0, false, fmt.Errorf("economy2: parse deposit entry %q: %w", goodID, err)
	}

	ds.Amount -= qty
	if ds.Amount < 0 {
		ds.Amount = 0
	}
	depleted = ds.Amount <= 0

	updated, _ := json.Marshal(ds)
	state[goodID] = updated

	finalRaw, _ := json.Marshal(state)
	_, err = db.Exec(ctx,
		`UPDATE planets SET resource_deposits=$1 WHERE id=$2`,
		finalRaw, planetID,
	)
	return ds.Amount, depleted, err
}

// countActiveMines counts mine facilities currently active on a given deposit.
// Used to enforce the max_mines slot limit before assigning a new mine order.
func countActiveMines(ctx context.Context, db *pgxpool.Pool, planetID uuid.UUID, goodID string) (int, error) {
	var count int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM econ2_facilities f
		JOIN econ2_nodes n ON n.id = f.node_id
		WHERE n.planet_id = $1
		  AND f.factory_type = 'mine'
		  AND f.config->>'deposit_good_id' = $2
		  AND f.status NOT IN ('idle','destroyed')
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

// EnsureDeposits is a no-op since migration 014.
// Deposits are initialised by the planet generator in planets.resource_deposits.
// Kept for call-site compatibility during the transition period.
func EnsureDeposits(_ context.Context, _ *pgxpool.Pool, _ uuid.UUID) error {
	return nil
}
