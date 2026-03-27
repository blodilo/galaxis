package economy2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
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
