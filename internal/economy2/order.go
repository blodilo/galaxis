package economy2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderType distinguishes batch (finite) from continuous (ongoing) orders.
type OrderType string

const (
	OrderTypeBatch      OrderType = "batch"
	OrderTypeContinuous OrderType = "continuous"
	OrderTypeBuild      OrderType = "build"
)

// OrderStatus is the lifecycle state of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusWaiting   OrderStatus = "waiting"
	OrderStatusReady     OrderStatus = "ready"
	OrderStatusRunning   OrderStatus = "running"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// ProductionOrder is the in-memory representation of an econ2_orders row.
type ProductionOrder struct {
	ID         uuid.UUID  `json:"id"`
	PlayerID   uuid.UUID  `json:"player_id"`
	StarID     uuid.UUID  `json:"star_id"`
	NodeID     uuid.UUID  `json:"node_id"`
	FacilityID *uuid.UUID `json:"facility_id"`
	OrderType  OrderType  `json:"order_type"`
	Status     OrderStatus `json:"status"`

	// Snapshot — copied from recipe at creation, never modified.
	RecipeID    string        `json:"recipe_id"`
	ProductID   string        `json:"product_id"`
	FactoryType string        `json:"factory_type"`
	Inputs      []RecipeInput `json:"inputs"`
	BaseYield   float64       `json:"base_yield"`
	RecipeTicks int           `json:"recipe_ticks"`
	Efficiency  float64       `json:"efficiency"`

	TargetQty       float64            `json:"target_qty"`
	AllocatedInputs map[string]float64 `json:"allocated_inputs"`
	ProducedQty     float64            `json:"produced_qty"`
	Priority        int                `json:"priority"`
}

// TransportNeedPerTick returns the total input units needed per tick to sustain production.
func (o *ProductionOrder) TransportNeedPerTick() float64 {
	if o.RecipeTicks <= 0 {
		return 0
	}
	total := 0.0
	for _, input := range o.Inputs {
		total += input.Amount
	}
	return total / float64(o.RecipeTicks)
}

// CreateOrder inserts a new order and sets o.ID from the returning clause.
func CreateOrder(ctx context.Context, db *pgxpool.Pool, o *ProductionOrder) error {
	inputsJSON, err := json.Marshal(o.Inputs)
	if err != nil {
		return fmt.Errorf("economy2: marshal inputs: %w", err)
	}
	if o.AllocatedInputs == nil {
		o.AllocatedInputs = map[string]float64{}
	}
	allocJSON, err := json.Marshal(o.AllocatedInputs)
	if err != nil {
		return fmt.Errorf("economy2: marshal allocated: %w", err)
	}

	return db.QueryRow(ctx, `
		INSERT INTO econ2_orders
		    (player_id, star_id, node_id, order_type, status,
		     recipe_id, product_id, factory_type, inputs, base_yield, recipe_ticks, efficiency,
		     target_qty, allocated_inputs, priority)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id
	`,
		o.PlayerID, o.StarID, o.NodeID, string(o.OrderType), string(o.Status),
		o.RecipeID, o.ProductID, o.FactoryType, inputsJSON, o.BaseYield, o.RecipeTicks, o.Efficiency,
		o.TargetQty, allocJSON, o.Priority,
	).Scan(&o.ID)
}

// LoadOrderByID loads a single order by primary key.
func LoadOrderByID(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*ProductionOrder, error) {
	var (
		o         ProductionOrder
		inputsRaw []byte
		allocRaw  []byte
		orderType string
		status    string
	)
	err := db.QueryRow(ctx, `
		SELECT id, player_id, star_id, node_id, facility_id, order_type, status,
		       recipe_id, product_id, factory_type, inputs, base_yield, recipe_ticks, efficiency,
		       target_qty, allocated_inputs, produced_qty, priority
		FROM econ2_orders WHERE id = $1
	`, id).Scan(
		&o.ID, &o.PlayerID, &o.StarID, &o.NodeID, &o.FacilityID, &orderType, &status,
		&o.RecipeID, &o.ProductID, &o.FactoryType, &inputsRaw, &o.BaseYield, &o.RecipeTicks, &o.Efficiency,
		&o.TargetQty, &allocRaw, &o.ProducedQty, &o.Priority,
	)
	if err != nil {
		return nil, fmt.Errorf("economy2: load order %s: %w", id, err)
	}
	o.OrderType = OrderType(orderType)
	o.Status = OrderStatus(status)
	_ = json.Unmarshal(inputsRaw, &o.Inputs)
	o.AllocatedInputs = map[string]float64{}
	_ = json.Unmarshal(allocRaw, &o.AllocatedInputs)
	return &o, nil
}

// ListOrdersByNode returns all orders for a node.
func ListOrdersByNode(ctx context.Context, db *pgxpool.Pool, nodeID uuid.UUID) ([]*ProductionOrder, error) {
	rows, err := db.Query(ctx, `
		SELECT id, player_id, star_id, node_id, facility_id, order_type, status,
		       recipe_id, product_id, factory_type, inputs, base_yield, recipe_ticks, efficiency,
		       target_qty, allocated_inputs, produced_qty, priority
		FROM econ2_orders WHERE node_id = $1
		ORDER BY priority DESC, created_at ASC
	`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

func scanOrders(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]*ProductionOrder, error) {
	var result []*ProductionOrder
	for rows.Next() {
		var (
			o         ProductionOrder
			inputsRaw []byte
			allocRaw  []byte
			orderType string
			status    string
		)
		if err := rows.Scan(
			&o.ID, &o.PlayerID, &o.StarID, &o.NodeID, &o.FacilityID, &orderType, &status,
			&o.RecipeID, &o.ProductID, &o.FactoryType, &inputsRaw, &o.BaseYield, &o.RecipeTicks, &o.Efficiency,
			&o.TargetQty, &allocRaw, &o.ProducedQty, &o.Priority,
		); err != nil {
			return nil, err
		}
		o.OrderType = OrderType(orderType)
		o.Status = OrderStatus(status)
		_ = json.Unmarshal(inputsRaw, &o.Inputs)
		o.AllocatedInputs = map[string]float64{}
		_ = json.Unmarshal(allocRaw, &o.AllocatedInputs)
		result = append(result, &o)
	}
	return result, rows.Err()
}
