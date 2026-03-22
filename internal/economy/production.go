package economy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FacilityConfig is the JSONB config blob stored in facilities.config.
type FacilityConfig struct {
	Level          int     `json:"level"`
	RecipeID       string  `json:"recipe_id,omitempty"`
	TicksRemaining int     `json:"ticks_remaining"`
	EfficiencyAcc  float64 `json:"efficiency_acc"`
	// DepositID is set for mine facilities — the good_id being extracted.
	DepositID string `json:"deposit_id,omitempty"`
}

// Facility is the in-memory representation of a facilities row.
type Facility struct {
	ID           uuid.UUID
	PlayerID     uuid.UUID
	StarID       uuid.UUID
	PlanetID     *uuid.UUID // nil for orbital facilities
	FacilityType string
	Status       string
	Config       FacilityConfig
}

// tickEvent is a single entry in production_log.events.
type tickEvent struct {
	Type       string  `json:"type"`
	FacilityID string  `json:"facility_id"`
	Good       string  `json:"good,omitempty"`
	Qty        float64 `json:"qty,omitempty"`
	Missing    string  `json:"missing,omitempty"`
	AccBefore  float64 `json:"acc_before,omitempty"`
	AccAfter   float64 `json:"acc_after,omitempty"`
}

// ProductionHandler returns a tick.Handler that advances the production system
// by one tick for all running facilities.
func ProductionHandler(db *pgxpool.Pool, reg *Registries) func(ctx context.Context, tickN int64) {
	return func(ctx context.Context, tickN int64) {
		if err := runProductionTick(ctx, db, reg, tickN); err != nil {
			log.Printf("production tick %d error: %v", tickN, err)
		}
	}
}

func runProductionTick(ctx context.Context, db *pgxpool.Pool, reg *Registries, tickN int64) error {
	facilities, err := loadRunningFacilities(ctx, db)
	if err != nil {
		return fmt.Errorf("production: load facilities: %w", err)
	}

	// Group log events by (player_id, star_id) to write one log row per group.
	type groupKey struct {
		PlayerID uuid.UUID
		StarID   uuid.UUID
	}
	logEvents := make(map[groupKey][]tickEvent)

	for _, f := range facilities {
		events, err := processFacility(ctx, db, reg, f, tickN)
		if err != nil {
			log.Printf("production: facility %s: %v", f.ID, err)
			continue
		}
		if len(events) > 0 {
			key := groupKey{f.PlayerID, f.StarID}
			logEvents[key] = append(logEvents[key], events...)
		}
	}

	// Persist log events.
	for key, events := range logEvents {
		if err := appendLog(ctx, db, key.PlayerID, key.StarID, tickN, events); err != nil {
			log.Printf("production: log write %s/%s tick %d: %v",
				key.PlayerID, key.StarID, tickN, err)
		}
	}

	// Prune old log entries (rolling window: last 100 ticks).
	if err := pruneOldLogs(ctx, db, tickN); err != nil {
		log.Printf("production: prune logs: %v", err)
	}

	return nil
}

// processFacility advances a single facility by one tick and returns log events.
func processFacility(
	ctx context.Context,
	db *pgxpool.Pool,
	reg *Registries,
	f *Facility,
	tickN int64,
) ([]tickEvent, error) {
	// Mine facilities are handled separately (deposit extraction, no recipe).
	if f.FacilityType == "mine" {
		return processMine(ctx, db, reg, f, tickN)
	}
	return processRecipeFacility(ctx, db, reg, f, tickN)
}

// processMine extracts resources from a planet deposit into system storage.
func processMine(
	ctx context.Context,
	db *pgxpool.Pool,
	reg *Registries,
	f *Facility,
	tickN int64,
) ([]tickEvent, error) {
	if f.PlanetID == nil || f.Config.DepositID == "" {
		return nil, nil
	}

	pd, err := GetDeposits(ctx, db, *f.PlanetID)
	if err != nil {
		return nil, fmt.Errorf("mine: read deposits: %w", err)
	}

	ds, ok := pd.State[f.Config.DepositID]
	if !ok || ds.Remaining <= 0 {
		// Deposit exhausted → pause.
		if err := setStatus(ctx, db, f.ID, "paused_depleted"); err != nil {
			return nil, err
		}
		return []tickEvent{{Type: "deposit_depleted", FacilityID: f.ID.String(), Good: f.Config.DepositID}}, nil
	}

	// Output rate = min(facility_level_rate, deposit.max_rate).
	facilityRate := mineOutputRate(reg, f.Config.Level)
	rate := math.Min(facilityRate, ds.MaxRate)
	rate = math.Min(rate, ds.Remaining)

	// Deplete deposit.
	depleted, err := Deplete(ctx, db, *f.PlanetID, f.Config.DepositID, rate)
	if err != nil {
		return nil, fmt.Errorf("mine: deplete: %w", err)
	}

	// Produce into storage.
	if _, err := Produce(ctx, db, f.PlayerID, f.StarID, map[string]float64{f.Config.DepositID: rate}); err != nil {
		return nil, fmt.Errorf("mine: produce to storage: %w", err)
	}

	events := []tickEvent{{
		Type:       "mined",
		FacilityID: f.ID.String(),
		Good:       f.Config.DepositID,
		Qty:        rate,
	}}

	if depleted {
		events = append(events, tickEvent{Type: "deposit_depleted", FacilityID: f.ID.String(), Good: f.Config.DepositID})
	} else {
		// Update mining player's survey snapshot.
		if err := UpdateOwnMiningSnapshot(ctx, db, f.PlayerID, *f.PlanetID, tickN, reg); err != nil {
			log.Printf("mine: update survey snapshot: %v", err)
		}
		// Deposit warning checks.
		events = append(events, depositWarningEvents(f, ds.Remaining-rate, reg)...)
	}

	return events, nil
}

// processRecipeFacility runs one tick of the recipe-based production algorithm.
func processRecipeFacility(
	ctx context.Context,
	db *pgxpool.Pool,
	reg *Registries,
	f *Facility,
	tickN int64,
) ([]tickEvent, error) {
	if f.Config.RecipeID == "" {
		return nil, nil // not configured
	}

	recipe, ok := reg.Recipes[f.Config.RecipeID]
	if !ok {
		return nil, fmt.Errorf("unknown recipe %q for facility %s", f.Config.RecipeID, f.ID)
	}

	// Mid-batch: just decrement counter.
	if f.Config.TicksRemaining > 1 {
		f.Config.TicksRemaining--
		return nil, saveFacilityConfig(ctx, db, f)
	}

	// Batch complete: try to consume inputs.
	storage, err := GetStorage(ctx, db, f.PlayerID, f.StarID)
	if err != nil {
		return nil, err
	}

	for goodID, qty := range recipe.Inputs {
		if !Has(storage, goodID, qty) {
			if err := setStatus(ctx, db, f.ID, "paused_input"); err != nil {
				return nil, err
			}
			return []tickEvent{{Type: "paused_input", FacilityID: f.ID.String(), Missing: goodID}}, nil
		}
	}

	// Deduct inputs.
	for goodID, qty := range recipe.Inputs {
		storage[goodID] -= qty
	}

	// Produce outputs with efficiency accumulator.
	eta := reg.Facilities.Eta(f.FacilityType, f.Config.Level)
	var events []tickEvent
	pausedOutput := false

	for goodID, baseQty := range recipe.Outputs {
		accBefore := f.Config.EfficiencyAcc
		f.Config.EfficiencyAcc += baseQty * eta
		produced := math.Floor(f.Config.EfficiencyAcc)
		f.Config.EfficiencyAcc -= produced

		if produced <= 0 {
			continue
		}

		storage[goodID] += produced
		events = append(events, tickEvent{
			Type:       "produced",
			FacilityID: f.ID.String(),
			Good:       goodID,
			Qty:        produced,
			AccBefore:  accBefore,
			AccAfter:   f.Config.EfficiencyAcc,
		})

		// Deposit warning check (only relevant for mine outputs — skip here).
		_ = pausedOutput // capacity checks are Post-MVP (storage is unbounded in MVP)
	}

	// Save updated storage.
	if err := SetStorage(ctx, db, f.PlayerID, f.StarID, storage); err != nil {
		return nil, err
	}

	// Reset batch counter and save facility config.
	f.Config.TicksRemaining = recipe.Ticks
	if err := saveFacilityConfig(ctx, db, f); err != nil {
		return nil, err
	}

	return events, nil
}

// depositWarningEvents returns warning log events when deposit crosses thresholds.
func depositWarningEvents(f *Facility, remaining float64, reg *Registries) []tickEvent {
	spec, ok := reg.Deposits[f.Config.DepositID]
	if !ok {
		return nil
	}
	initial := spec.BaseUnits // approximate: full-quality initial value used as reference
	pct := remaining / initial

	var events []tickEvent
	switch {
	case pct <= reg.DepositWarnings.CriticalPercent:
		events = append(events, tickEvent{
			Type:       "deposit_critical",
			FacilityID: f.ID.String(),
			Good:       f.Config.DepositID,
			Qty:        remaining,
		})
	case pct <= reg.DepositWarnings.WarningPercent:
		events = append(events, tickEvent{
			Type:       "deposit_warning",
			FacilityID: f.ID.String(),
			Good:       f.Config.DepositID,
			Qty:        remaining,
		})
	}
	return events
}

// mineOutputRate returns the facility_output_per_tick for the given mine level.
func mineOutputRate(reg *Registries, level int) float64 {
	rates, ok := reg.Facilities.OutputPerTick["mine"]
	if !ok || level < 1 || level > len(rates) {
		return 0
	}
	return float64(rates[level-1])
}

// --- DB helpers -------------------------------------------------------------

func loadRunningFacilities(ctx context.Context, db *pgxpool.Pool) ([]*Facility, error) {
	rows, err := db.Query(ctx,
		`SELECT id, player_id, star_id, planet_id, facility_type, status, config
		 FROM facilities
		 WHERE status = 'running'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Facility
	for rows.Next() {
		var (
			f       Facility
			rawConf []byte
		)
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.StarID, &f.PlanetID,
			&f.FacilityType, &f.Status, &rawConf); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawConf, &f.Config); err != nil {
			return nil, fmt.Errorf("facility %s: unmarshal config: %w", f.ID, err)
		}
		result = append(result, &f)
	}
	return result, rows.Err()
}

func saveFacilityConfig(ctx context.Context, db *pgxpool.Pool, f *Facility) error {
	raw, err := json.Marshal(f.Config)
	if err != nil {
		return fmt.Errorf("facility: marshal config: %w", err)
	}
	_, err = db.Exec(ctx,
		`UPDATE facilities SET config = $1, updated_at = now() WHERE id = $2`,
		raw, f.ID,
	)
	return err
}

func setStatus(ctx context.Context, db *pgxpool.Pool, facilityID uuid.UUID, status string) error {
	_, err := db.Exec(ctx,
		`UPDATE facilities SET status = $1, updated_at = now() WHERE id = $2`,
		status, facilityID,
	)
	return err
}
