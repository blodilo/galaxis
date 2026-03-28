package economy2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SchedulerHandler returns a tick.Handler that:
//  1. Attempts MRP allocation for pending/waiting orders.
//  2. Assigns ready orders to idle facilities.
//
// Must run before ProductionHandler so newly assigned facilities produce in the same tick.
func SchedulerHandler(db *pgxpool.Pool, recipes RecipeBook) func(ctx context.Context, tickN int64) {
	return func(ctx context.Context, tickN int64) {
		if err := runScheduler(ctx, db, recipes); err != nil {
			log.Printf("economy2: scheduler tick %d: %v", tickN, err)
		}
	}
}

func runScheduler(ctx context.Context, db *pgxpool.Pool, recipes RecipeBook) error {
	if err := tryAllocatePending(ctx, db, recipes); err != nil {
		log.Printf("economy2: mrp phase: %v", err)
	}
	return assignReadyOrders(ctx, db)
}

// tryAllocatePending runs MRP for all pending/waiting orders and commits allocations.
func tryAllocatePending(ctx context.Context, db *pgxpool.Pool, recipes RecipeBook) error {
	rows, err := db.Query(ctx, `
		SELECT id, node_id, product_id, factory_type, inputs, base_yield, target_qty, allocated_inputs
		FROM econ2_orders
		WHERE status IN ('pending','waiting')
		ORDER BY priority DESC, created_at ASC
		LIMIT 100
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		id          uuid.UUID
		nodeID      uuid.UUID
		productID   string
		factoryType string
		inputs      []RecipeInput
		baseYield   float64
		targetQty   float64
		allocated   map[string]float64
	}

	var pending []row
	for rows.Next() {
		var (
			r         row
			inputsRaw []byte
			allocRaw  []byte
		)
		if err := rows.Scan(
			&r.id, &r.nodeID, &r.productID, &r.factoryType,
			&inputsRaw, &r.baseYield, &r.targetQty, &allocRaw,
		); err != nil {
			return err
		}
		_ = json.Unmarshal(inputsRaw, &r.inputs)
		r.allocated = map[string]float64{}
		_ = json.Unmarshal(allocRaw, &r.allocated)
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, r := range pending {
		totals := map[string]float64{}
		visiting := map[string]bool{}
		if err := ResolveDemand(r.productID, r.targetQty, r.factoryType, recipes, totals, visiting); err != nil {
			log.Printf("economy2: mrp resolve order %s: %v", r.id, err)
			continue
		}

		order := &ProductionOrder{
			ID:              r.id,
			NodeID:          r.nodeID,
			AllocatedInputs: r.allocated,
		}
		if err := AllocateOrder(ctx, db, r.nodeID, order, totals); err != nil {
			log.Printf("economy2: allocate order %s: %v", r.id, err)
		}
	}
	return nil
}

// assignReadyOrders assigns ready orders to idle matching facilities.
// For mine facilities, enforces the deposit slot limit before assigning.
func assignReadyOrders(ctx context.Context, db *pgxpool.Pool) error {
	// JOIN with econ2_nodes to get the node's planet_id (mine slot check needs it).
	rows, err := db.Query(ctx, `
		SELECT
		    f.id, f.factory_type, f.star_id, n.planet_id,
		    o.id, o.recipe_id, o.recipe_ticks, o.product_id
		FROM econ2_facilities f
		JOIN econ2_nodes n ON n.id = f.node_id
		CROSS JOIN LATERAL (
		    SELECT id, recipe_id, recipe_ticks, product_id
		    FROM econ2_orders
		    WHERE node_id      = f.node_id
		      AND factory_type = f.factory_type
		      AND status       = 'ready'
		    ORDER BY priority DESC, created_at ASC
		    LIMIT 1
		) o
		WHERE f.status = 'idle'
		  AND f.current_order_id IS NULL
	`)
	if err != nil {
		return fmt.Errorf("economy2: assign query: %w", err)
	}
	defer rows.Close()

	type assignment struct {
		facilityID  uuid.UUID
		factoryType string
		starID      uuid.UUID
		planetID    *uuid.UUID
		orderID     uuid.UUID
		recipeID    string
		recipeTicks int
		productID   string
	}

	var assignments []assignment
	for rows.Next() {
		var a assignment
		if err := rows.Scan(
			&a.facilityID, &a.factoryType, &a.starID, &a.planetID,
			&a.orderID, &a.recipeID, &a.recipeTicks, &a.productID,
		); err != nil {
			return fmt.Errorf("economy2: assign scan: %w", err)
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, a := range assignments {
		// Mine slot enforcement: check deposit.max_rate as max_mines.
		if a.factoryType == "mine" {
			if ok, err := mineSlotAvailable(ctx, db, a.starID, a.planetID, a.productID); err != nil {
				log.Printf("economy2: mine slot check facility %s: %v", a.facilityID, err)
				continue
			} else if !ok {
				continue // deposit slots exhausted
			}
		}

		if err := assignFacility(ctx, db, a.facilityID, a.orderID, a.recipeTicks); err != nil {
			log.Printf("economy2: assign facility %s to order %s: %v", a.facilityID, a.orderID, err)
		}
	}
	return nil
}

// mineSlotAvailable returns true when the deposit still has free mine slots.
// max_mines is read from planet_deposits.state[goodID].max_rate.
// When planetID is nil (orbit node), the home planet is resolved via starID.
func mineSlotAvailable(ctx context.Context, db *pgxpool.Pool, starID uuid.UUID, planetID *uuid.UUID, goodID string) (bool, error) {
	if planetID == nil {
		pid, err := FindHomePlanet(ctx, db, starID)
		if err != nil {
			return false, fmt.Errorf("economy2: mine facility missing planet_id: %w", err)
		}
		planetID = pid
	}

	if err := EnsureDeposits(ctx, db, *planetID); err != nil {
		return false, fmt.Errorf("economy2: mine ensure deposits: %w", err)
	}

	ds, err := readDeposit(ctx, db, *planetID, goodID)
	if err != nil {
		return false, err
	}

	active, err := countActiveMines(ctx, db, *planetID, goodID)
	if err != nil {
		return false, err
	}

	return float64(active) < ds.MaxRate, nil
}

func assignFacility(ctx context.Context, db *pgxpool.Pool, facilityID, orderID uuid.UUID, ticks int) error {
	_, err := db.Exec(ctx, `
		UPDATE econ2_facilities
		SET status           = 'running',
		    current_order_id = $1,
		    config           = config || jsonb_build_object(
		                           'ticks_remaining', $2::int,
		                           'efficiency_acc',  0.0
		                       ),
		    updated_at       = now()
		WHERE id = $3
	`, orderID, ticks, facilityID)
	return err
}
