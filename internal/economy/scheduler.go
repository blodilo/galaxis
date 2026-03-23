package economy

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductionOrder is the in-memory representation of a production_orders row.
type ProductionOrder struct {
	ID             uuid.UUID
	PlayerID       uuid.UUID
	StarID         uuid.UUID
	FacilityType   string
	RecipeID       string
	Mode           string // continuous_full | continuous_demand | batch
	BatchRemaining *int
	GoodID         *string
	MinStock       *float64
	TargetStock    *float64
	Priority       int
	Active         bool
}

// SchedulerHandler returns a tick.Handler that assigns idle facilities to pending orders.
// Must run BEFORE ProductionHandler so newly assigned facilities produce in the same tick.
func SchedulerHandler(db *pgxpool.Pool, reg *Registries) func(ctx context.Context, tickN int64) {
	return func(ctx context.Context, tickN int64) {
		if err := runScheduler(ctx, db, reg); err != nil {
			log.Printf("scheduler tick %d error: %v", tickN, err)
		}
	}
}

// runScheduler finds idle facilities and assigns them to the highest-priority active order.
func runScheduler(ctx context.Context, db *pgxpool.Pool, reg *Registries) error {
	// Find all idle facilities (no current order) that have at least one active order.
	rows, err := db.Query(ctx, `
		SELECT
		  f.id, f.player_id, f.star_id, f.facility_type,
		  o.id, o.recipe_id, o.mode, o.batch_remaining,
		  o.good_id, o.min_stock, o.target_stock,
		  f.storage_node_id
		FROM facilities f
		CROSS JOIN LATERAL (
		  SELECT id, recipe_id, mode, batch_remaining, good_id, min_stock, target_stock
		  FROM production_orders
		  WHERE player_id    = f.player_id
		    AND star_id       = f.star_id
		    AND facility_type = f.facility_type
		    AND active        = true
		  ORDER BY
		    CASE WHEN mode = 'batch' THEN 0 ELSE 1 END ASC,
		    priority DESC,
		    created_at ASC
		  LIMIT 1
		) o
		WHERE f.status          = 'idle'
		  AND f.current_order_id IS NULL
	`)
	if err != nil {
		return fmt.Errorf("scheduler: query: %w", err)
	}
	defer rows.Close()

	type assignment struct {
		facilityID    uuid.UUID
		playerID      uuid.UUID
		starID        uuid.UUID
		facilityType  string
		orderID       uuid.UUID
		recipeID      string
		mode          string
		batchRemain   *int
		goodID        *string
		minStock      *float64
		targetStock   *float64
		storageNodeID *uuid.UUID
	}

	var assignments []assignment
	for rows.Next() {
		var a assignment
		if err := rows.Scan(
			&a.facilityID, &a.playerID, &a.starID, &a.facilityType,
			&a.orderID, &a.recipeID, &a.mode, &a.batchRemain,
			&a.goodID, &a.minStock, &a.targetStock,
			&a.storageNodeID,
		); err != nil {
			return fmt.Errorf("scheduler: scan: %w", err)
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, a := range assignments {
		// continuous_demand: skip assignment if stock >= target already.
		if a.mode == "continuous_demand" && a.goodID != nil && a.targetStock != nil {
			nodeID, err := resolveStorageNode(ctx, db, a.storageNodeID, a.playerID, a.starID, nil)
			if err == nil {
				storage, err := GetNodeStorage(ctx, db, nodeID)
				if err == nil && storage[*a.goodID] >= *a.targetStock {
					continue // already enough stock — don't assign yet
				}
			}
		}

		recipe, ok := reg.Recipes[a.recipeID]
		if !ok {
			log.Printf("scheduler: unknown recipe %q for order %s", a.recipeID, a.orderID)
			continue
		}

		if err := assignFacilityToOrder(ctx, db, a.facilityID, a.orderID, a.recipeID, recipe.Ticks); err != nil {
			log.Printf("scheduler: assign facility %s to order %s: %v", a.facilityID, a.orderID, err)
		}
	}
	return nil
}

// assignFacilityToOrder sets current_order_id, updates config with recipe, sets status=running.
func assignFacilityToOrder(
	ctx context.Context,
	db *pgxpool.Pool,
	facilityID, orderID uuid.UUID,
	recipeID string,
	ticks int,
) error {
	_, err := db.Exec(ctx, `
		UPDATE facilities
		SET
		  current_order_id = $1,
		  status           = 'running',
		  config           = config
		              || jsonb_build_object(
		                   'recipe_id',       $2::text,
		                   'ticks_remaining', $3::int,
		                   'efficiency_acc',  0.0
		                 ),
		  updated_at       = now()
		WHERE id = $4
	`, orderID, recipeID, ticks, facilityID)
	return err
}

// resolveStorageNode returns a node UUID, creating one if storageNodeID is nil.
func resolveStorageNode(
	ctx context.Context,
	db *pgxpool.Pool,
	storageNodeID *uuid.UUID,
	playerID, starID uuid.UUID,
	planetID *uuid.UUID,
) (uuid.UUID, error) {
	if storageNodeID != nil {
		return *storageNodeID, nil
	}
	return GetOrCreateNode(ctx, db, playerID, starID, planetID)
}

// --- Order completion (called from production.go) ---------------------------

// HandleOrderBatchComplete is called after a recipe facility finishes one batch
// while assigned to an order. It handles batch countdown, demand pausing, etc.
// Returns true if the facility should be unassigned (set back to idle).
func HandleOrderBatchComplete(
	ctx context.Context,
	db *pgxpool.Pool,
	f *Facility,
	storage StorageContents,
) (unassign bool, err error) {
	if f.CurrentOrderID == nil {
		return false, nil
	}

	// Load current order state.
	var mode string
	var batchRemaining *int
	var goodID *string
	var targetStock *float64
	err = db.QueryRow(ctx, `
		SELECT mode, batch_remaining, good_id, target_stock
		FROM production_orders
		WHERE id = $1
	`, *f.CurrentOrderID).Scan(&mode, &batchRemaining, &goodID, &targetStock)
	if err != nil {
		return false, fmt.Errorf("order complete: load order: %w", err)
	}

	switch mode {
	case "batch":
		if batchRemaining == nil {
			return false, nil
		}
		next := *batchRemaining - 1
		if next <= 0 {
			// Order fulfilled — deactivate and unassign.
			if _, err := db.Exec(ctx, `
				UPDATE production_orders SET active = false, updated_at = now() WHERE id = $1
			`, *f.CurrentOrderID); err != nil {
				return false, err
			}
			return true, nil
		}
		// Decrement batch counter.
		_, err = db.Exec(ctx, `
			UPDATE production_orders SET batch_remaining = $1, updated_at = now() WHERE id = $2
		`, next, *f.CurrentOrderID)
		return false, err

	case "continuous_demand":
		if goodID == nil || targetStock == nil {
			return false, nil
		}
		// Pause when stock has reached target.
		if storage[*goodID] >= *targetStock {
			return true, nil
		}
		return false, nil

	default: // continuous_full — never unassign
		return false, nil
	}
}
