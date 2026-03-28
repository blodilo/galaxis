package economy2

import (
	"context"
	"fmt"
	"log"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MineParams holds mining calibration values loaded from game-params.
type MineParams struct {
	// BaseRate is the output per tick for a level-1 mine before the level multiplier.
	BaseRate float64 `yaml:"base_rate"`
	// LevelMultiplier[i] is the multiplier for mine level i+1.
	// Higher levels are unlocked by research; for MVP Lv1 is the only available level.
	LevelMultiplier []float64 `yaml:"level_multiplier"`
}

// RateForLevel returns the output per tick for the given 1-based mine level.
// Falls back to BaseRate × 1.0 for out-of-range levels.
func (p MineParams) RateForLevel(level int) float64 {
	if level < 1 || level > len(p.LevelMultiplier) {
		return p.BaseRate
	}
	return p.BaseRate * p.LevelMultiplier[level-1]
}

// processMine handles one production tick for a mine facility.
//
// Flow:
//  1. Read deposit state.
//  2. If depleted → pause order + facility with status paused_depleted.
//  3. Compute rate, deplete planet_deposits.
//  4. Apply efficiency accumulator → floor → produce into goods stock.
//  5. Reset batch counter.
func processMine(
	ctx context.Context,
	db *pgxpool.Pool,
	f *Facility,
	order *ProductionOrder,
	recipe *Recipe,
	params MineParams,
) error {
	// Fallback: orbit nodes have no planet_id — resolve via home planet.
	if f.PlanetID == nil {
		pid, err := FindHomePlanet(ctx, db, f.StarID)
		if err != nil {
			return fmt.Errorf("economy2: mine facility missing planet_id: %w", err)
		}
		f.PlanetID = pid
	}

	goodID := recipe.GeologicalInput

	// Lazily initialise planet_deposits if the planet was just colonised.
	if err := EnsureDeposits(ctx, db, *f.PlanetID); err != nil {
		return fmt.Errorf("economy2: mine ensure deposits: %w", err)
	}

	ds, err := readDeposit(ctx, db, *f.PlanetID, goodID)
	if err != nil {
		return fmt.Errorf("economy2: mine read deposit: %w", err)
	}

	// Deposit exhausted → pause.
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

	// Compute actual extraction rate.
	rate := params.RateForLevel(f.Config.Level)
	actual := math.Min(rate, ds.Remaining)

	// Deplete planet_deposits.
	_, depleted, err := depleteDeposit(ctx, db, *f.PlanetID, goodID, actual)
	if err != nil {
		return fmt.Errorf("economy2: mine deplete: %w", err)
	}

	// Efficiency accumulator → floor → produce.
	eff := recipe.Efficiency
	if eff <= 0 {
		eff = 1.0
	}
	f.Config.EfficiencyAcc += actual * eff
	produced := math.Floor(f.Config.EfficiencyAcc)
	f.Config.EfficiencyAcc -= produced

	if produced > 0 {
		if err := AddToStock(ctx, db, order.NodeID, recipe.ProductID, produced); err != nil {
			log.Printf("economy2: mine add stock %s: %v", recipe.ProductID, err)
		}
		if _, err := db.Exec(ctx,
			`UPDATE econ2_orders SET produced_qty = produced_qty + $1, updated_at=now() WHERE id=$2`,
			produced, order.ID,
		); err != nil {
			return err
		}
	}

	// Deposit just became empty after this tick → pause next tick.
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

	// Reset batch counter.
	f.Config.TicksRemaining = recipe.Ticks
	return saveFacilityConfig(ctx, db, f)
}
