package economy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SurveySnapshot is the per-resource view that the player receives.
// Which fields are non-nil depends on the survey quality.
type SurveySnapshot map[string]ResourceSnapshot

// ResourceSnapshot holds the information revealed about a single resource.
// Nil pointer fields are omitted in JSON (unknown to the player).
type ResourceSnapshot struct {
	Present         bool     `json:"present"`
	RemainingApprox *string  `json:"remaining_approx,omitempty"` // e.g. "40000–60000"
	RemainingExact  *float64 `json:"remaining_exact,omitempty"`
	MaxRate         *float64 `json:"max_rate,omitempty"`
	Slots           *int     `json:"slots,omitempty"`
}

// PlayerSurvey is the full player_surveys row.
type PlayerSurvey struct {
	PlayerID   uuid.UUID
	PlanetID   uuid.UUID
	SurveyedAt time.Time
	TickN      int64
	Quality    float64
	Snapshot   SurveySnapshot
	Stale      bool // true when planet_deposits was updated after this survey
}

// ExecuteSurvey runs a survey for (playerID, planetID) at the given quality.
// It reads the ground truth from planet_deposits (initialising it if needed),
// builds a filtered snapshot, and upserts player_surveys.
// Returns the player's view (filtered snapshot + staleness flag).
func ExecuteSurvey(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, planetID uuid.UUID,
	quality float64,
	tickN int64,
	resourceQualities map[string]float64, // from planets.resource_deposits (for lazy init)
	reg *Registries,
) (*PlayerSurvey, error) {
	// Ensure planet_deposits exists (lazy init).
	pd, err := InitDeposits(ctx, db, planetID, resourceQualities, reg.Deposits)
	if err != nil {
		return nil, fmt.Errorf("survey: init deposits: %w", err)
	}

	snapshot := filterSnapshot(pd.State, quality, reg.SurveyThresholds)

	rawSnap, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("survey: marshal snapshot: %w", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO player_surveys
		   (player_id, planet_id, tick_n, quality, snapshot, surveyed_at)
		 VALUES ($1, $2, $3, $4, $5, now())
		 ON CONFLICT (player_id, planet_id) DO UPDATE
		   SET tick_n      = EXCLUDED.tick_n,
		       quality     = EXCLUDED.quality,
		       snapshot    = EXCLUDED.snapshot,
		       surveyed_at = now()`,
		playerID, planetID, tickN, quality, rawSnap,
	)
	if err != nil {
		return nil, fmt.Errorf("survey: upsert player_surveys: %w", err)
	}

	return &PlayerSurvey{
		PlayerID:   playerID,
		PlanetID:   planetID,
		SurveyedAt: time.Now(),
		TickN:      tickN,
		Quality:    quality,
		Snapshot:   snapshot,
		Stale:      false,
	}, nil
}

// GetSurvey reads the current player_surveys row for (playerID, planetID).
// The Stale field is set to true when planet_deposits.updated_at is newer
// than the survey (another player mined after our last scan).
// Returns pgx.ErrNoRows if no survey exists yet.
func GetSurvey(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, planetID uuid.UUID,
) (*PlayerSurvey, error) {
	row := db.QueryRow(ctx, `
		SELECT
		  ps.tick_n,
		  ps.quality,
		  ps.snapshot,
		  ps.surveyed_at,
		  (pd.updated_at > ps.surveyed_at) AS stale
		FROM player_surveys ps
		LEFT JOIN planet_deposits pd ON pd.planet_id = ps.planet_id
		WHERE ps.player_id = $1 AND ps.planet_id = $2`,
		playerID, planetID,
	)

	var (
		tickN      int64
		quality    float64
		rawSnap    []byte
		surveyedAt time.Time
		stale      bool
	)
	if err := row.Scan(&tickN, &quality, &rawSnap, &surveyedAt, &stale); err != nil {
		return nil, err
	}

	var snap SurveySnapshot
	if err := json.Unmarshal(rawSnap, &snap); err != nil {
		return nil, fmt.Errorf("survey: unmarshal snapshot: %w", err)
	}

	return &PlayerSurvey{
		PlayerID:   playerID,
		PlanetID:   planetID,
		SurveyedAt: surveyedAt,
		TickN:      tickN,
		Quality:    quality,
		Snapshot:   snap,
		Stale:      stale,
	}, nil
}

// GetSystemSurveys returns all player_surveys rows for a player within a system.
func GetSystemSurveys(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID uuid.UUID,
) ([]*PlayerSurvey, error) {
	rows, err := db.Query(ctx, `
		SELECT
		  ps.planet_id,
		  ps.tick_n,
		  ps.quality,
		  ps.snapshot,
		  ps.surveyed_at,
		  (pd.updated_at > ps.surveyed_at) AS stale
		FROM player_surveys ps
		JOIN planets p ON p.id = ps.planet_id
		LEFT JOIN planet_deposits pd ON pd.planet_id = ps.planet_id
		WHERE ps.player_id = $1 AND p.star_id = $2`,
		playerID, starID,
	)
	if err != nil {
		return nil, fmt.Errorf("survey: system query: %w", err)
	}
	defer rows.Close()

	var result []*PlayerSurvey
	for rows.Next() {
		var (
			pID        uuid.UUID
			tickN      int64
			quality    float64
			rawSnap    []byte
			surveyedAt time.Time
			stale      bool
		)
		if err := rows.Scan(&pID, &tickN, &quality, &rawSnap, &surveyedAt, &stale); err != nil {
			return nil, fmt.Errorf("survey: scan row: %w", err)
		}
		var snap SurveySnapshot
		if err := json.Unmarshal(rawSnap, &snap); err != nil {
			return nil, fmt.Errorf("survey: unmarshal snapshot: %w", err)
		}
		result = append(result, &PlayerSurvey{
			PlayerID:   playerID,
			PlanetID:   pID,
			SurveyedAt: surveyedAt,
			TickN:      tickN,
			Quality:    quality,
			Snapshot:   snap,
			Stale:      stale,
		})
	}
	return result, rows.Err()
}

// UpdateOwnMiningSnapshot refreshes the player's survey snapshot after mining
// (called each tick for resources where player_id is the active miner).
// Only updates remaining counts — rate and slot data stay as surveyed.
func UpdateOwnMiningSnapshot(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, planetID uuid.UUID,
	tickN int64,
	reg *Registries,
) error {
	// Read current survey.
	ps, err := GetSurvey(ctx, db, playerID, planetID)
	if err == pgx.ErrNoRows {
		return nil // no survey → nothing to update
	}
	if err != nil {
		return fmt.Errorf("survey: update mining snapshot read: %w", err)
	}

	// Read current ground truth.
	pd, err := GetDeposits(ctx, db, planetID)
	if err != nil {
		return fmt.Errorf("survey: update mining snapshot deposits: %w", err)
	}

	// Refresh snapshot with current quality (only fields visible at that quality).
	updated := filterSnapshot(pd.State, ps.Quality, reg.SurveyThresholds)

	rawSnap, err := json.Marshal(updated)
	if err != nil {
		return fmt.Errorf("survey: marshal updated snapshot: %w", err)
	}

	_, err = db.Exec(ctx,
		`UPDATE player_surveys
		 SET snapshot = $1, tick_n = $2, surveyed_at = now()
		 WHERE player_id = $3 AND planet_id = $4`,
		rawSnap, tickN, playerID, planetID,
	)
	return err
}

// filterSnapshot builds a ResourceSnapshot map filtered to what the given
// quality level allows the player to see.
func filterSnapshot(state map[string]DepositState, quality float64, t SurveyThresholds) SurveySnapshot {
	snap := make(SurveySnapshot, len(state))
	for goodID, ds := range state {
		rs := ResourceSnapshot{Present: true}

		switch {
		case quality >= t.Exact:
			exact := ds.Remaining
			rate := ds.MaxRate
			slots := ds.Slots
			rs.RemainingExact = &exact
			rs.MaxRate = &rate
			rs.Slots = &slots

		case quality >= t.RangeNarrow:
			// ±25 % range
			approxStr := approxRange(ds.Remaining, 0.25)
			rs.RemainingApprox = &approxStr

		case quality >= t.RangeApprox:
			// ±50 % range
			approxStr := approxRange(ds.Remaining, 0.50)
			rs.RemainingApprox = &approxStr

		// quality >= t.TypeOnly (always true given survey is created): presence only
		}

		snap[goodID] = rs
	}
	return snap
}

// approxRange formats a range string like "40000–60000" around value ± pct.
func approxRange(value, pct float64) string {
	lo := value * (1 - pct)
	hi := value * (1 + pct)
	return fmt.Sprintf("%.0f–%.0f", lo, hi)
}
