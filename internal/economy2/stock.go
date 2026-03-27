package economy2

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ItemStock tracks physical and allocated quantity for one item in one node.
type ItemStock struct {
	Total     float64 `json:"total"`
	Allocated float64 `json:"allocated"`
}

// Available returns quantity free to allocate.
func (s ItemStock) Available() float64 {
	return s.Total - s.Allocated
}

// NodeStock returns all ItemStock entries for a node, keyed by item_id.
func NodeStock(ctx context.Context, db *pgxpool.Pool, nodeID uuid.UUID) (map[string]ItemStock, error) {
	rows, err := db.Query(ctx,
		`SELECT item_id, total, allocated FROM econ2_item_stock WHERE node_id = $1`,
		nodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]ItemStock{}
	for rows.Next() {
		var itemID string
		var s ItemStock
		if err := rows.Scan(&itemID, &s.Total, &s.Allocated); err != nil {
			return nil, err
		}
		result[itemID] = s
	}
	return result, rows.Err()
}

// AddToStock adds amount to total (production output, incoming shipment).
func AddToStock(ctx context.Context, db *pgxpool.Pool, nodeID uuid.UUID, itemID string, amount float64) error {
	_, err := db.Exec(ctx, `
		INSERT INTO econ2_item_stock (node_id, item_id, total)
		VALUES ($1, $2, $3)
		ON CONFLICT (node_id, item_id) DO UPDATE
		SET total = econ2_item_stock.total + EXCLUDED.total, updated_at = now()
	`, nodeID, itemID, amount)
	return err
}

// ConsumeAllocated removes amount from both total and allocated (material consumed in production).
func ConsumeAllocated(ctx context.Context, db *pgxpool.Pool, nodeID uuid.UUID, itemID string, amount float64) error {
	_, err := db.Exec(ctx, `
		UPDATE econ2_item_stock
		SET total = total - $3, allocated = allocated - $3, updated_at = now()
		WHERE node_id = $1 AND item_id = $2
	`, nodeID, itemID, amount)
	return err
}

// GetOrCreateNode returns the node ID for the given scope, creating it if not found.
func GetOrCreateNode(ctx context.Context, db *pgxpool.Pool, playerID, starID uuid.UUID, planetID *uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	var err error

	if planetID != nil {
		err = db.QueryRow(ctx,
			`SELECT id FROM econ2_nodes WHERE player_id=$1 AND star_id=$2 AND planet_id=$3`,
			playerID, starID, planetID,
		).Scan(&id)
	} else {
		err = db.QueryRow(ctx,
			`SELECT id FROM econ2_nodes WHERE player_id=$1 AND star_id=$2 AND planet_id IS NULL`,
			playerID, starID,
		).Scan(&id)
	}

	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("economy2: find node: %w", err)
	}

	level := "orbital"
	if planetID != nil {
		level = "planetary"
	}
	err = db.QueryRow(ctx,
		`INSERT INTO econ2_nodes (player_id, star_id, planet_id, level) VALUES ($1,$2,$3,$4) RETURNING id`,
		playerID, starID, planetID, level,
	).Scan(&id)
	return id, err
}
