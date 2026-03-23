package economy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StorageContents maps good_id → quantity (float64 units).
type StorageContents map[string]float64

// StorageNode is the in-memory representation of a storage_nodes row.
type StorageNode struct {
	ID       uuid.UUID
	PlayerID uuid.UUID
	StarID   uuid.UUID
	PlanetID *uuid.UUID
	Level    string   // "planetary" | "orbital" | "intersystem"
	Capacity *float64 // nil = unlimited (planetary)
	Storage  StorageContents
}

// GetNodeStorage reads the JSONB storage for a given node ID.
// Returns empty StorageContents (not an error) if the node is not found.
func GetNodeStorage(ctx context.Context, db *pgxpool.Pool, nodeID uuid.UUID) (StorageContents, error) {
	var raw []byte
	err := db.QueryRow(ctx, `SELECT storage FROM storage_nodes WHERE id = $1`, nodeID).Scan(&raw)
	if err != nil {
		return StorageContents{}, nil
	}
	var c StorageContents
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("storage: unmarshal node %s: %w", nodeID, err)
	}
	return c, nil
}

// SetNodeStorage persists storage contents for an existing node.
func SetNodeStorage(ctx context.Context, db *pgxpool.Pool, nodeID uuid.UUID, c StorageContents) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("storage: marshal: %w", err)
	}
	_, err = db.Exec(ctx,
		`UPDATE storage_nodes SET storage = $1, updated_at = now() WHERE id = $2`,
		raw, nodeID,
	)
	return err
}

// GetOrCreateNode returns the node ID for (playerID, starID, planetID),
// creating the row if it does not yet exist.
// Pass nil planetID for an orbital node, non-nil for a planetary node.
func GetOrCreateNode(
	ctx context.Context, db *pgxpool.Pool,
	playerID, starID uuid.UUID,
	planetID *uuid.UUID,
) (uuid.UUID, error) {
	id, err := findNode(ctx, db, playerID, starID, planetID)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return uuid.Nil, fmt.Errorf("storage: find node: %w", err)
	}

	level := "orbital"
	if planetID != nil {
		level = "planetary"
	}
	err = db.QueryRow(ctx,
		`INSERT INTO storage_nodes (player_id, star_id, planet_id, level)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING
		 RETURNING id`,
		playerID, starID, planetID, level,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	// Race condition: another goroutine won the INSERT; re-fetch.
	id, err = findNode(ctx, db, playerID, starID, planetID)
	return id, err
}

func findNode(ctx context.Context, db *pgxpool.Pool, playerID, starID uuid.UUID, planetID *uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	var err error
	if planetID != nil {
		err = db.QueryRow(ctx,
			`SELECT id FROM storage_nodes
			 WHERE player_id=$1 AND star_id=$2 AND planet_id=$3`,
			playerID, starID, planetID,
		).Scan(&id)
	} else {
		err = db.QueryRow(ctx,
			`SELECT id FROM storage_nodes
			 WHERE player_id=$1 AND star_id=$2 AND planet_id IS NULL AND level='orbital'`,
			playerID, starID,
		).Scan(&id)
	}
	return id, err
}

// GetSystemNodes returns all storage nodes for a (player, star) pair,
// ordered: orbital first, then planetary by planet_id.
func GetSystemNodes(ctx context.Context, db *pgxpool.Pool, playerID, starID uuid.UUID) ([]StorageNode, error) {
	rows, err := db.Query(ctx,
		`SELECT id, planet_id, level, capacity, storage
		 FROM storage_nodes
		 WHERE player_id=$1 AND star_id=$2
		 ORDER BY level DESC, planet_id NULLS FIRST`,
		playerID, starID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []StorageNode
	for rows.Next() {
		var (
			n          StorageNode
			rawStorage []byte
		)
		n.PlayerID = playerID
		n.StarID = starID
		if err := rows.Scan(&n.ID, &n.PlanetID, &n.Level, &n.Capacity, &rawStorage); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawStorage, &n.Storage); err != nil {
			n.Storage = StorageContents{}
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

// Has returns true if goodID is present in c with quantity ≥ qty.
func Has(c StorageContents, goodID string, qty float64) bool {
	return c[goodID] >= qty
}

// ProduceToNode adds amounts to a storage node and persists the result.
func ProduceToNode(
	ctx context.Context, db *pgxpool.Pool,
	nodeID uuid.UUID, amounts map[string]float64,
) (StorageContents, error) {
	c, err := GetNodeStorage(ctx, db, nodeID)
	if err != nil {
		return nil, err
	}
	for g, q := range amounts {
		c[g] += q
	}
	return c, SetNodeStorage(ctx, db, nodeID, c)
}

// ConsumeFromNode deducts amounts from a storage node.
// Returns an error if any good is insufficient; no partial deductions occur.
func ConsumeFromNode(
	ctx context.Context, db *pgxpool.Pool,
	nodeID uuid.UUID, amounts map[string]float64,
) (StorageContents, error) {
	c, err := GetNodeStorage(ctx, db, nodeID)
	if err != nil {
		return nil, err
	}
	for g, q := range amounts {
		if !Has(c, g, q) {
			return nil, fmt.Errorf("storage: insufficient %s (have %.2f, need %.2f)", g, c[g], q)
		}
	}
	for g, q := range amounts {
		c[g] -= q
	}
	return c, SetNodeStorage(ctx, db, nodeID, c)
}
