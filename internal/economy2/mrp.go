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

// WalkNode is a single node in the recipe walk tree.
type WalkNode struct {
	ProductID   string
	FactoryType string
	Qty         float64
	Recipe      *Recipe
	Children    []*WalkNode
}

// walkRecipeTree recursively builds a production tree for goal BOM creation.
// Each node represents one production step; leaves are raw materials (no recipe).
// fn is called for each non-leaf node (factory step that needs an order).
// visiting prevents infinite recursion on cyclic recipes.
func walkRecipeTree(
	productID string,
	qty float64,
	recipes RecipeBook,
	visiting map[string]bool,
	fn func(node *WalkNode),
) (*WalkNode, error) {
	if visiting[productID] {
		return nil, fmt.Errorf("economy2: mrp cycle at %s", productID)
	}

	// Find recipe — try all factory types for this product.
	var recipe *Recipe
	for key, r := range recipes {
		if key.ProductID == productID {
			recipe = r
			break
		}
	}

	node := &WalkNode{
		ProductID: productID,
		Qty:       qty,
		Recipe:    recipe,
	}

	if recipe == nil {
		// Raw material leaf — no further resolution.
		return node, nil
	}

	node.FactoryType = recipe.FactoryType

	visiting[productID] = true
	defer delete(visiting, productID)

	runs := qty / recipe.BaseYield
	for _, input := range recipe.Inputs {
		child, err := walkRecipeTree(input.ItemID, input.Amount*runs, recipes, visiting, fn)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)
	}

	fn(node)
	return node, nil
}

// AllocateOrder atomically checks and reserves all materials for an order.
// Sets order.Status = OrderStatusReady on success, OrderStatusWaiting if stock insufficient.
// Uses a DB transaction with row-level locking to prevent double-allocation.
func AllocateOrder(ctx context.Context, db *pgxpool.Pool, nodeID uuid.UUID, order *ProductionOrder, totals map[string]float64) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

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
