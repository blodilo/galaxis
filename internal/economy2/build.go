package economy2

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BuildTickHandler processes construction orders each tick.
// Construction orders (factory_type='construction') do not require a physical facility.
// Each tick, produced_qty is incremented as a progress counter.
// When produced_qty reaches recipe_ticks, the facility is created and the order completed.
func BuildTickHandler(db *pgxpool.Pool) func(ctx context.Context, tickN int64) {
	return func(ctx context.Context, tickN int64) {
		if err := runBuildTick(ctx, db); err != nil {
			log.Printf("economy2: build tick %d: %v", tickN, err)
		}
	}
}

func runBuildTick(ctx context.Context, db *pgxpool.Pool) error {
	// Move ready construction orders to running (no facility assignment step needed).
	if _, err := db.Exec(ctx, `
		UPDATE econ2_orders
		SET status = 'running', updated_at = now()
		WHERE factory_type = 'construction' AND status = 'ready'
	`); err != nil {
		return err
	}

	// Load all running construction orders with their node's planet_id.
	rows, err := db.Query(ctx, `
		SELECT o.id, o.player_id, o.star_id, o.node_id,
		       o.product_id, o.recipe_ticks, o.produced_qty,
		       o.inputs, o.allocated_inputs,
		       n.planet_id
		FROM econ2_orders o
		JOIN econ2_nodes n ON n.id = o.node_id
		WHERE o.factory_type = 'construction' AND o.status = 'running'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type constructionOrder struct {
		id              uuid.UUID
		playerID        uuid.UUID
		starID          uuid.UUID
		nodeID          uuid.UUID
		productID       string
		recipeTicks     int
		producedQty     float64
		inputs          []RecipeInput
		allocatedInputs map[string]float64
		planetID        *uuid.UUID
	}

	var pending []constructionOrder
	for rows.Next() {
		var co constructionOrder
		var inputsRaw, allocRaw []byte
		if err := rows.Scan(
			&co.id, &co.playerID, &co.starID, &co.nodeID,
			&co.productID, &co.recipeTicks, &co.producedQty,
			&inputsRaw, &allocRaw,
			&co.planetID,
		); err != nil {
			return err
		}
		_ = json.Unmarshal(inputsRaw, &co.inputs)
		co.allocatedInputs = map[string]float64{}
		_ = json.Unmarshal(allocRaw, &co.allocatedInputs)
		pending = append(pending, co)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, co := range pending {
		newQty := co.producedQty + 1
		if int(newQty) >= co.recipeTicks {
			if err := finishBuildOrder(ctx, db, co.id, co.playerID, co.starID, co.nodeID, co.planetID,
				co.productID, co.inputs, co.allocatedInputs); err != nil {
				log.Printf("economy2: finish build order %s: %v", co.id, err)
			}
		} else {
			if _, err := db.Exec(ctx,
				`UPDATE econ2_orders SET produced_qty=$1, updated_at=now() WHERE id=$2`,
				newQty, co.id,
			); err != nil {
				log.Printf("economy2: advance build order %s: %v", co.id, err)
			}
		}
	}
	return nil
}

func finishBuildOrder(
	ctx context.Context, db *pgxpool.Pool,
	orderID, playerID, starID, nodeID uuid.UUID, planetID *uuid.UUID,
	productID string, inputs []RecipeInput, allocated map[string]float64,
) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Consume allocated inputs from stock.
	for _, inp := range inputs {
		qty := allocated[inp.ItemID]
		if qty <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE econ2_item_stock
			SET total = total - $3, allocated = allocated - $3, updated_at = now()
			WHERE node_id = $1 AND item_id = $2
		`, nodeID, inp.ItemID, qty); err != nil {
			return err
		}
	}

	// Derive factory_type and deposit_good_id from product_id.
	factoryType, depositGoodID := parseBuildProductID(productID)

	// Mine facilities need a planet. If the build node has no planet_id (star-level node),
	// fall back to the star's first planet and initialise deposits if needed.
	if factoryType == "mine" && planetID == nil {
		pid, err := FindHomePlanet(ctx, db, starID)
		if err == nil {
			planetID = pid
		}
	}
	if factoryType == "mine" && planetID != nil {
		_ = EnsureDeposits(ctx, db, *planetID)
	}

	// Create the facility.
	cfg := FacilityConfig{Level: 1, DepositGoodID: depositGoodID}
	cfgRaw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	var facilityID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO econ2_facilities (player_id, star_id, planet_id, node_id, factory_type, status, config)
		VALUES ($1,$2,$3,$4,$5,'idle',$6)
		RETURNING id
	`, playerID, starID, planetID, nodeID, factoryType, cfgRaw).Scan(&facilityID); err != nil {
		return err
	}

	// Mark order complete.
	if _, err := tx.Exec(ctx, `
		UPDATE econ2_orders
		SET status='completed', produced_qty=recipe_ticks, updated_at=now()
		WHERE id=$1
	`, orderID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// parseBuildProductID extracts factory_type and deposit_good_id from a construction order product_id.
//
//	"facility_mine_iron_ore" → ("mine", "iron_ore")
//	"facility_smelter"       → ("smelter", "")
func parseBuildProductID(productID string) (factoryType, depositGoodID string) {
	s := strings.TrimPrefix(productID, "facility_")
	const minePrefix = "mine_"
	if strings.HasPrefix(s, minePrefix) {
		return "mine", strings.TrimPrefix(s, minePrefix)
	}
	return s, ""
}
