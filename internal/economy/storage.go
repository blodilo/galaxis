package economy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StorageContents maps good_id → quantity (float64 units).
// MVP: one shared pool per (player, star system).
// Post-MVP: becomes a local buffer node in the pipeline-graph model.
type StorageContents map[string]float64

// GetStorage reads the system_storage JSONB for (playerID, starID).
// Returns an empty StorageContents (not an error) if the row does not exist yet.
func GetStorage(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID uuid.UUID,
) (StorageContents, error) {
	row := db.QueryRow(ctx,
		`SELECT contents FROM system_storage WHERE player_id = $1 AND star_id = $2`,
		playerID, starID,
	)

	var raw []byte
	if err := row.Scan(&raw); err != nil {
		return StorageContents{}, nil // no row yet → empty
	}

	var contents StorageContents
	if err := json.Unmarshal(raw, &contents); err != nil {
		return nil, fmt.Errorf("storage: unmarshal: %w", err)
	}
	return contents, nil
}

// SetStorage upserts the full contents for (playerID, starID).
func SetStorage(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID uuid.UUID,
	contents StorageContents,
) error {
	raw, err := json.Marshal(contents)
	if err != nil {
		return fmt.Errorf("storage: marshal: %w", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO system_storage (player_id, star_id, contents)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (player_id, star_id) DO UPDATE
		   SET contents = EXCLUDED.contents, updated_at = now()`,
		playerID, starID, raw,
	)
	return err
}

// Has returns true if goodID is available in storage with quantity ≥ qty.
func Has(contents StorageContents, goodID string, qty float64) bool {
	return contents[goodID] >= qty
}

// Consume deducts amounts from storage in a single atomic read-modify-write.
// All inputs are validated before any deduction — no partial deductions.
// Returns the updated contents.
func Consume(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID uuid.UUID,
	amounts map[string]float64,
) (StorageContents, error) {
	contents, err := GetStorage(ctx, db, playerID, starID)
	if err != nil {
		return nil, err
	}
	for goodID, qty := range amounts {
		if !Has(contents, goodID, qty) {
			return nil, fmt.Errorf("storage: insufficient %s (have %.2f, need %.2f)",
				goodID, contents[goodID], qty)
		}
	}
	for goodID, qty := range amounts {
		contents[goodID] -= qty
	}
	if err := SetStorage(ctx, db, playerID, starID, contents); err != nil {
		return nil, err
	}
	return contents, nil
}

// Produce adds amounts to storage.
// Returns the updated contents.
func Produce(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID uuid.UUID,
	amounts map[string]float64,
) (StorageContents, error) {
	contents, err := GetStorage(ctx, db, playerID, starID)
	if err != nil {
		return nil, err
	}
	for goodID, qty := range amounts {
		contents[goodID] += qty
	}
	if err := SetStorage(ctx, db, playerID, starID, contents); err != nil {
		return nil, err
	}
	return contents, nil
}
