package economy2

import (
	"context"
	"fmt"
	"log"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
)

// processExtractor handles one production tick for an extractor facility.
//
// Extraction formula: extracted_per_tick = facility.config.max_rate × deposit.quality
//
// Flow:
//  1. Resolve planet (fallback for orbit nodes with no planet_id).
//  2. Read deposit state.
//  3. If depleted → pause order + facility.
//  4. Compute rate, deplete planets.resource_deposits.
//  5. Apply efficiency accumulator → floor → produce into goods stock.
//  6. Reset batch counter.
func processExtractor(
	ctx context.Context,
	db *pgxpool.Pool,
	f *Facility,
	order *ProductionOrder,
	recipe *Recipe,
) error {
	if f.PlanetID == nil {
		pid, err := FindHomePlanet(ctx, db, f.StarID)
		if err != nil {
			return fmt.Errorf("economy2: extractor facility missing planet_id: %w", err)
		}
		f.PlanetID = pid
	}

	goodID := recipe.GeologicalInput

	ds, err := readDeposit(ctx, db, *f.PlanetID, goodID)
	if err != nil {
		return fmt.Errorf("economy2: extractor read deposit: %w", err)
	}

	if ds.Remaining <= 0 {
		if _, err := db.Exec(ctx,
			`UPDATE econ2_orders SET status='paused_depleted', updated_at=now() WHERE id=$1`,
			order.ID,
		); err != nil {
			return err
		}
		_, err = db.Exec(ctx,
			`UPDATE econ2_facilities SET status='paused_depleted', updated_at=now() WHERE id=$1`,
			f.ID,
		)
		return err
	}

	rate := f.Config.MaxRate * ds.Quality
	if rate <= 0 {
		rate = 1.0
	}
	actual := math.Min(rate, ds.Remaining)

	_, depleted, err := depleteDeposit(ctx, db, *f.PlanetID, goodID, actual)
	if err != nil {
		return fmt.Errorf("economy2: extractor deplete: %w", err)
	}

	eff := recipe.Efficiency
	if eff <= 0 {
		eff = 1.0
	}
	f.Config.EfficiencyAcc += actual * eff
	produced := math.Floor(f.Config.EfficiencyAcc)
	f.Config.EfficiencyAcc -= produced

	if produced > 0 {
		if err := AddToStock(ctx, db, order.NodeID, recipe.ProductID, produced); err != nil {
			log.Printf("economy2: extractor add stock %s: %v", recipe.ProductID, err)
		}
		if _, err := db.Exec(ctx,
			`UPDATE econ2_orders SET produced_qty = produced_qty + $1, updated_at=now() WHERE id=$2`,
			produced, order.ID,
		); err != nil {
			return err
		}
	}

	if depleted {
		if _, err := db.Exec(ctx,
			`UPDATE econ2_orders SET status='paused_depleted', updated_at=now() WHERE id=$1`,
			order.ID,
		); err != nil {
			return err
		}
		_, err = db.Exec(ctx,
			`UPDATE econ2_facilities SET status='paused_depleted', updated_at=now() WHERE id=$1`,
			f.ID,
		)
		return err
	}

	f.Config.TicksRemaining = recipe.Ticks
	return saveFacilityConfig(ctx, db, f)
}
