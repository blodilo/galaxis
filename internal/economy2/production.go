package economy2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductionHandler returns a tick.Handler that advances all running economy2 facilities.
func ProductionHandler(db *pgxpool.Pool, recipes RecipeBook) func(ctx context.Context, tickN int64) {
	return func(ctx context.Context, tickN int64) {
		if err := runProductionTick(ctx, db, recipes); err != nil {
			log.Printf("economy2: production tick %d: %v", tickN, err)
		}
	}
}

func runProductionTick(ctx context.Context, db *pgxpool.Pool, recipes RecipeBook) error {
	facilities, err := loadRunningFacilities(ctx, db)
	if err != nil {
		return fmt.Errorf("economy2: load facilities: %w", err)
	}

	for _, f := range facilities {
		if err := processFacility(ctx, db, recipes, f); err != nil {
			log.Printf("economy2: facility %s: %v", f.ID, err)
		}
	}
	return nil
}

func processFacility(ctx context.Context, db *pgxpool.Pool, recipes RecipeBook, f *Facility) error {
	// Mid-batch: just decrement counter.
	if f.Config.TicksRemaining > 1 {
		f.Config.TicksRemaining--
		return saveFacilityConfig(ctx, db, f)
	}

	if f.CurrentOrderID == nil {
		return nil
	}

	order, err := LoadOrderByID(ctx, db, *f.CurrentOrderID)
	if err != nil {
		return err
	}

	key := RecipeKey{order.ProductID, order.FactoryType}
	recipe, ok := recipes[key]
	if !ok {
		return fmt.Errorf("economy2: unknown recipe (%s, %s)", order.ProductID, order.FactoryType)
	}

	// Extractor facilities draw from planets.resource_deposits, not goods storage.
	if recipe.IsExtractor() {
		return processExtractor(ctx, db, f, order, recipe)
	}

	// Consume allocated inputs.
	for itemID, needed := range order.AllocatedInputs {
		if err := ConsumeAllocated(ctx, db, order.NodeID, itemID, needed); err != nil {
			log.Printf("economy2: consume %s order %s: %v", itemID, order.ID, err)
		}
	}
	if _, err := db.Exec(ctx,
		`UPDATE econ2_orders SET allocated_inputs='{}', updated_at=now() WHERE id=$1`,
		order.ID,
	); err != nil {
		return err
	}

	// Produce output with efficiency accumulator.
	eff := recipe.Efficiency
	if eff <= 0 {
		eff = 1.0
	}
	f.Config.EfficiencyAcc += recipe.BaseYield * eff
	produced := math.Floor(f.Config.EfficiencyAcc)
	f.Config.EfficiencyAcc -= produced

	if produced > 0 {
		if err := AddToStock(ctx, db, order.NodeID, recipe.ProductID, produced); err != nil {
			log.Printf("economy2: add stock %s: %v", recipe.ProductID, err)
		}
		if _, err := db.Exec(ctx,
			`UPDATE econ2_orders SET produced_qty = produced_qty + $1, updated_at=now() WHERE id=$2`,
			produced, order.ID,
		); err != nil {
			return err
		}
	}

	// Batch order complete?
	if order.OrderType == OrderTypeBatch && (order.ProducedQty+produced) >= order.TargetQty {
		if _, err := db.Exec(ctx,
			`UPDATE econ2_orders SET status='completed', updated_at=now() WHERE id=$1`,
			order.ID,
		); err != nil {
			return err
		}
		if _, err := db.Exec(ctx,
			`UPDATE econ2_facilities SET status='idle', current_order_id=NULL, updated_at=now() WHERE id=$1`,
			f.ID,
		); err != nil {
			return err
		}
		return nil
	}

	// Reset batch counter for next run.
	f.Config.TicksRemaining = recipe.Ticks
	if err := saveFacilityConfig(ctx, db, f); err != nil {
		return err
	}

	// Continuous: re-allocate inputs for next batch cycle.
	if order.OrderType == OrderTypeContinuous {
		totals := map[string]float64{}
		for _, input := range recipe.Inputs {
			totals[input.ItemID] = input.Amount
		}
		order.AllocatedInputs = map[string]float64{}
		if err := AllocateOrder(ctx, db, order.NodeID, order, totals); err != nil {
			log.Printf("economy2: re-allocate continuous order %s: %v", order.ID, err)
		}
		if order.Status == OrderStatusWaiting {
			if _, err := db.Exec(ctx,
				`UPDATE econ2_facilities SET status='paused_input', updated_at=now() WHERE id=$1`,
				f.ID,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func loadRunningFacilities(ctx context.Context, db *pgxpool.Pool) ([]*Facility, error) {
	// JOIN with econ2_nodes to populate PlanetID from the node's location.
	rows, err := db.Query(ctx, `
		SELECT f.id, f.player_id, f.star_id, f.node_id, f.factory_type, f.status, f.config, f.current_order_id,
		       n.planet_id
		FROM econ2_facilities f
		JOIN econ2_nodes n ON n.id = f.node_id
		WHERE f.status IN ('running','building')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Facility
	for rows.Next() {
		var (
			f      Facility
			cfgRaw []byte
		)
		if err := rows.Scan(
			&f.ID, &f.PlayerID, &f.StarID, &f.NodeID,
			&f.FactoryType, &f.Status, &cfgRaw, &f.CurrentOrderID,
			&f.PlanetID,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(cfgRaw, &f.Config); err != nil {
			return nil, fmt.Errorf("economy2: facility %s config: %w", f.ID, err)
		}
		result = append(result, &f)
	}
	return result, rows.Err()
}
