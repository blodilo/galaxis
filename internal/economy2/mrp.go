package economy2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ResolveDemand recursively resolves all base material requirements (dry run, no DB writes).
// totals accumulates {itemID → total required amount}.
// visiting prevents infinite recursion on cyclic recipes.
func ResolveDemand(
	productID string,
	amount float64,
	factoryType string,
	recipes RecipeBook,
	totals map[string]float64,
	visiting map[string]bool,
) error {
	if visiting[productID] {
		return fmt.Errorf("economy2: mrp cycle at %s", productID)
	}
	visiting[productID] = true
	defer delete(visiting, productID)

	key := RecipeKey{productID, factoryType}
	recipe, ok := recipes[key]
	if !ok {
		// Base raw material — no recipe, must be in stock.
		totals[productID] += amount
		return nil
	}

	runs := amount / recipe.BaseYield
	for _, input := range recipe.Inputs {
		if err := ResolveDemand(input.ItemID, input.Amount*runs, factoryType, recipes, totals, visiting); err != nil {
			return err
		}
	}
	return nil
}

// AllocateOrder atomically checks and reserves all materials for an order.
// Sets order.Status = OrderStatusReady on success, OrderStatusWaiting if stock insufficient.
// Uses a DB transaction with row-level locking to prevent double-allocation.
func AllocateOrder(ctx context.Context, db *pgxpool.Pool, nodeID uuid.UUID, order *ProductionOrder, totals map[string]float64) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Phase 1: check availability (lock rows)
	for itemID, needed := range totals {
		var total, allocated float64
		err := tx.QueryRow(ctx, `
			SELECT COALESCE(total, 0), COALESCE(allocated, 0)
			FROM econ2_item_stock
			WHERE node_id=$1 AND item_id=$2
			FOR UPDATE
		`, nodeID, itemID).Scan(&total, &allocated)
		if err != nil || (total-allocated) < needed {
			_ = tx.Rollback(ctx)
			order.Status = OrderStatusWaiting
			_, _ = db.Exec(ctx,
				`UPDATE econ2_orders SET status='waiting', updated_at=now() WHERE id=$1`,
				order.ID,
			)
			return nil
		}
	}

	// Phase 2: commit allocations
	for itemID, needed := range totals {
		_, err := tx.Exec(ctx, `
			UPDATE econ2_item_stock SET allocated=allocated+$3, updated_at=now()
			WHERE node_id=$1 AND item_id=$2
		`, nodeID, itemID, needed)
		if err != nil {
			return err
		}
		order.AllocatedInputs[itemID] += needed
	}

	allocJSON, err := json.Marshal(order.AllocatedInputs)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`UPDATE econ2_orders SET status='ready', allocated_inputs=$1, updated_at=now() WHERE id=$2`,
		allocJSON, order.ID,
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	order.Status = OrderStatusReady
	return nil
}
