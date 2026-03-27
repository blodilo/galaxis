package economy2

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ShipState is the current phase of a ship's round-trip cycle.
type ShipState string

const (
	ShipStateLoading     ShipState = "loading"
	ShipStateTransitTo   ShipState = "transit_to"
	ShipStateUnloading   ShipState = "unloading"
	ShipStateTransitBack ShipState = "transit_back"
)

// transitTicks is the number of ticks a ship spends in transit (each direction).
// TODO: derive from route distance / ship speed when those params exist.
const transitTicks = 5

// Ship is the in-memory representation of an econ2_ships row.
type Ship struct {
	ID       uuid.UUID
	RouteID  uuid.UUID
	State    ShipState
	Cargo    map[string]float64
	CargoMax float64
	ETATick  int64
}

// ShipTickHandler returns a tick.Handler that advances all ship state machines.
func ShipTickHandler(db *pgxpool.Pool) func(ctx context.Context, tickN int64) {
	return func(ctx context.Context, tickN int64) {
		if err := runShipTick(ctx, db, tickN); err != nil {
			log.Printf("economy2: ship tick %d: %v", tickN, err)
		}
	}
}

func runShipTick(ctx context.Context, db *pgxpool.Pool, tickN int64) error {
	rows, err := db.Query(ctx, `
		SELECT s.id, s.route_id, s.state, s.cargo, s.cargo_max, s.eta_tick,
		       r.from_node_id, r.to_node_id
		FROM econ2_ships s
		JOIN econ2_routes r ON r.id = s.route_id
		WHERE s.eta_tick <= $1 AND r.status = 'active'
	`, tickN)
	if err != nil {
		return err
	}
	defer rows.Close()

	type entry struct {
		ship       Ship
		fromNodeID uuid.UUID
		toNodeID   uuid.UUID
	}

	var entries []entry
	for rows.Next() {
		var (
			e        entry
			cargoRaw []byte
			state    string
		)
		if err := rows.Scan(
			&e.ship.ID, &e.ship.RouteID, &state, &cargoRaw, &e.ship.CargoMax, &e.ship.ETATick,
			&e.fromNodeID, &e.toNodeID,
		); err != nil {
			return err
		}
		e.ship.State = ShipState(state)
		e.ship.Cargo = map[string]float64{}
		_ = json.Unmarshal(cargoRaw, &e.ship.Cargo)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, e := range entries {
		if err := advanceShip(ctx, db, e.ship, e.fromNodeID, e.toNodeID, tickN); err != nil {
			log.Printf("economy2: advance ship %s: %v", e.ship.ID, err)
		}
	}
	return nil
}

func advanceShip(ctx context.Context, db *pgxpool.Pool, s Ship, fromNodeID, toNodeID uuid.UUID, tickN int64) error {
	var nextState ShipState
	var nextETA int64

	switch s.State {
	case ShipStateLoading:
		nextState = ShipStateTransitTo
		nextETA = tickN + transitTicks

	case ShipStateTransitTo:
		// Deposit cargo into destination node.
		for itemID, qty := range s.Cargo {
			if err := AddToStock(ctx, db, toNodeID, itemID, qty); err != nil {
				log.Printf("economy2: ship %s unload %s: %v", s.ID, itemID, err)
			}
		}
		s.Cargo = map[string]float64{}
		nextState = ShipStateUnloading
		nextETA = tickN + 1

	case ShipStateUnloading:
		nextState = ShipStateTransitBack
		nextETA = tickN + transitTicks

	case ShipStateTransitBack:
		nextState = ShipStateLoading
		nextETA = tickN + 1
	}

	cargoJSON, _ := json.Marshal(s.Cargo)
	_, err := db.Exec(ctx, `
		UPDATE econ2_ships SET state=$1, cargo=$2, eta_tick=$3, updated_at=now() WHERE id=$4
	`, string(nextState), cargoJSON, nextETA, s.ID)
	return err
}
