package economy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const rollingLogWindow = 100

// appendLog inserts a new row into production_log for (playerID, starID, tickN).
func appendLog(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID uuid.UUID,
	tickN int64,
	events []tickEvent,
) error {
	raw, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("log: marshal events: %w", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO production_log (player_id, star_id, tick_n, events)
		 VALUES ($1, $2, $3, $4)`,
		playerID, starID, tickN, raw,
	)
	return err
}

// pruneOldLogs deletes production_log rows older than the rolling window.
// Called once per tick after all events are written.
func pruneOldLogs(ctx context.Context, db *pgxpool.Pool, currentTick int64) error {
	cutoff := currentTick - rollingLogWindow
	_, err := db.Exec(ctx,
		`DELETE FROM production_log WHERE tick_n < $1`, cutoff,
	)
	return err
}

// GetLog returns the last n log rows for (playerID, starID), newest first.
func GetLog(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID uuid.UUID,
	limit int,
) ([]LogRow, error) {
	rows, err := db.Query(ctx,
		`SELECT id, tick_n, events, created_at
		 FROM production_log
		 WHERE player_id = $1 AND star_id = $2
		 ORDER BY tick_n DESC
		 LIMIT $3`,
		playerID, starID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("log: query: %w", err)
	}
	defer rows.Close()

	var result []LogRow
	for rows.Next() {
		var (
			row    LogRow
			rawEvt []byte
		)
		if err := rows.Scan(&row.ID, &row.TickN, &rawEvt, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("log: scan: %w", err)
		}
		if err := json.Unmarshal(rawEvt, &row.Events); err != nil {
			return nil, fmt.Errorf("log: unmarshal events: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LogRow is the API-facing representation of a production_log row.
type LogRow struct {
	ID        uuid.UUID
	TickN     int64
	Events    []tickEvent
	CreatedAt interface{} // time.Time, scanned via pgx
}
