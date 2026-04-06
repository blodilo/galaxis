package economy2

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// playerIDFromRequest extracts the player UUID from the X-Player-ID header.
func playerIDFromRequest(r *http.Request) uuid.UUID {
	if h := r.Header.Get("X-Player-ID"); h != "" {
		if id, err := uuid.Parse(h); err == nil {
			return id
		}
	}
	return uuid.Nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Goal is the in-memory representation of an econ2_goals row.
type Goal struct {
	ID                 uuid.UUID         `json:"id"`
	PlayerID           uuid.UUID         `json:"player_id"`
	StarID             uuid.UUID         `json:"star_id"`
	ProductID          string            `json:"product_id"`
	TargetQty          float64           `json:"target_qty"`
	Priority           int               `json:"priority"`
	Status             string            `json:"status"`
	TransportOverrides map[string]string `json:"transport_overrides"`
	CreatedAt          time.Time         `json:"created_at"`
}

// RegisterRoutes mounts all economy2 REST endpoints on the given router.
func RegisterRoutes(r chi.Router, db *pgxpool.Pool, recipes RecipeBook, bootstrapCfg BootstrapConfig, catalog ItemCatalog) {
	r.Post("/econ2/items/deploy", deployItemHandler(db, catalog, recipes))
	r.Get("/econ2/facilities", listFacilitiesHandler(db))
	r.Delete("/econ2/facilities/{id}", destroyFacilityHandler(db))

	r.Post("/econ2/facilities/{id}/start", startFacilityHandler(db, recipes))
	r.Post("/econ2/facilities/{id}/stop", stopFacilityHandler(db))

	r.Post("/econ2/orders", createOrderHandler(db, recipes))
	r.Get("/econ2/orders", listOrdersHandler(db))
	r.Delete("/econ2/orders/{id}", cancelOrderHandler(db))

	r.Post("/econ2/routes", createRouteHandler(db))
	r.Get("/econ2/routes", listRoutesHandler(db))

	r.Get("/econ2/stock", getStockHandler(db))
	r.Post("/econ2/nodes", getOrCreateNodeHandler(db))
	r.Get("/econ2/my-nodes", listMyNodesHandler(db))

	r.Post("/econ2/bootstrap", bootstrapHandler(db, bootstrapCfg, recipes))

	r.Get("/econ2/recipes", listRecipesHandler(recipes))
	r.Get("/econ2/deposits", depositsHandler(db))

	// Goals
	r.Post("/econ2/goals", createGoalHandler(db, recipes))
	r.Get("/econ2/goals", listGoalsHandler(db))
	r.Delete("/econ2/goals/{id}", deleteGoalHandler(db))
	r.Patch("/econ2/goals/reorder", reorderGoalsHandler(db))

	// Player-wide aggregates
	r.Get("/econ2/stock-all", stockAllHandler(db))
	r.Get("/econ2/facilities-all", facilitiesAllHandler(db))
	r.Get("/econ2/orders-all", ordersAllHandler(db))
}

// --- GET /econ2/recipes ---

func listRecipesHandler(recipes RecipeBook) http.HandlerFunc {
	all := recipes.All()
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"recipes": all})
	}
}

// --- POST /econ2/items/deploy ---

type deployItemRequest struct {
	StarID string `json:"star_id"`
	NodeID string `json:"node_id"`
	ItemID string `json:"item_id"`
}

func deployItemHandler(db *pgxpool.Pool, catalog ItemCatalog, recipes RecipeBook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		var req deployItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		starID, err := uuid.Parse(req.StarID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid star_id")
			return
		}
		nodeID, err := uuid.Parse(req.NodeID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid node_id")
			return
		}

		f, err := DeployItem(r.Context(), db, playerID, starID, nodeID, nil, req.ItemID, catalog, recipes)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, f)
	}
}

// --- GET /econ2/facilities?star_id=... ---

func listFacilitiesHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		starIDStr := r.URL.Query().Get("star_id")
		if starIDStr == "" {
			writeError(w, http.StatusBadRequest, "star_id query param required")
			return
		}
		starID, err := uuid.Parse(starIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid star_id")
			return
		}

		// JOIN with nodes to expose planet_id in the response.
		rows, err := db.Query(r.Context(), `
			SELECT f.id, f.player_id, f.star_id, f.node_id, f.factory_type, f.status, f.config, f.current_order_id,
			       n.planet_id
			FROM econ2_facilities f
			JOIN econ2_nodes n ON n.id = f.node_id
			WHERE f.player_id=$1 AND f.star_id=$2
			ORDER BY f.created_at ASC
		`, playerID, starID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var facilities []*Facility
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
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			_ = json.Unmarshal(cfgRaw, &f.Config)
			facilities = append(facilities, &f)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if facilities == nil {
			facilities = []*Facility{}
		}

		writeJSON(w, http.StatusOK, map[string]any{"facilities": facilities})
	}
}

// --- POST /econ2/facilities/{id}/start ---

func startFacilityHandler(db *pgxpool.Pool, recipes RecipeBook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		f, err := LoadFacilityByID(r.Context(), db, id)
		if err != nil {
			writeError(w, http.StatusNotFound, "facility not found")
			return
		}
		if f.PlayerID != playerID {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		if f.Status != "idle" {
			writeError(w, http.StatusConflict, "facility is not idle")
			return
		}

		// For extractors: auto-create a continuous order for the deposit good.
		if f.FactoryType == FactoryTypeExtractor && f.Config.DepositGoodID != "" {
			key := RecipeKey{ProductID: f.Config.DepositGoodID, FactoryType: FactoryTypeExtractor}
			recipe, ok := recipes[key]
			if !ok {
				writeError(w, http.StatusBadRequest, "no extractor recipe for "+f.Config.DepositGoodID)
				return
			}
			order := &ProductionOrder{
				PlayerID:        playerID,
				StarID:          f.StarID,
				NodeID:          f.NodeID,
				OrderType:       OrderTypeContinuous,
				Status:          OrderStatusReady, // extractors don't need material allocation
				RecipeID:        recipe.RecipeID,
				ProductID:       recipe.ProductID,
				FactoryType:     recipe.FactoryType,
				Inputs:          recipe.Inputs,
				BaseYield:       recipe.BaseYield,
				RecipeTicks:     recipe.Ticks,
				Efficiency:      recipe.Efficiency,
				TargetQty:       0, // continuous = unlimited
				AllocatedInputs: map[string]float64{},
				Priority:        5,
			}
			if err := CreateOrder(r.Context(), db, order); err != nil {
				writeError(w, http.StatusInternalServerError, "create order: "+err.Error())
				return
			}
			// Assign immediately.
			if err := assignFacility(r.Context(), db, f.ID, order.ID, recipe.Ticks); err != nil {
				writeError(w, http.StatusInternalServerError, "assign: "+err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"status": "started", "order_id": order.ID})
			return
		}

		// Non-extractor: try to find a ready order matching this factory_type at the same star.
		// If none ready, try to allocate a pending/waiting one first.
		var orderID uuid.UUID
		err = db.QueryRow(r.Context(), `
			SELECT id FROM econ2_orders
			WHERE star_id=$1 AND factory_type=$2 AND status='ready'
			ORDER BY priority ASC, created_at ASC
			LIMIT 1
		`, f.StarID, f.FactoryType).Scan(&orderID)
		if err != nil {
			// No ready order — try to allocate a pending/waiting one.
			var pendingID uuid.UUID
			err2 := db.QueryRow(r.Context(), `
				SELECT id FROM econ2_orders
				WHERE star_id=$1 AND factory_type=$2 AND status IN ('pending','waiting')
				ORDER BY priority ASC, created_at ASC
				LIMIT 1
			`, f.StarID, f.FactoryType).Scan(&pendingID)
			if err2 != nil {
				writeError(w, http.StatusConflict, "no order available for "+f.FactoryType+"; create one in the PLAN tab first")
				return
			}
			// Try MRP allocation using the facility's node for stock lookup.
			pendingOrder, err3 := LoadOrderByID(r.Context(), db, pendingID)
			if err3 != nil {
				writeError(w, http.StatusInternalServerError, err3.Error())
				return
			}
			totals := map[string]float64{}
			for _, inp := range pendingOrder.Inputs {
				totals[inp.ItemID] = inp.Amount * (pendingOrder.TargetQty / pendingOrder.BaseYield)
			}
			// Try allocation against ALL nodes at this star, not just one.
			// Find the node that has the most stock.
			nodeRows, _ := db.Query(r.Context(), `
				SELECT id FROM econ2_nodes WHERE star_id=$1 AND player_id=$2
			`, f.StarID, playerID)
			allocOk := false
			if nodeRows != nil {
				var nodeIDs []uuid.UUID
				for nodeRows.Next() {
					var nid uuid.UUID
					_ = nodeRows.Scan(&nid)
					nodeIDs = append(nodeIDs, nid)
				}
				nodeRows.Close()
				for _, nid := range nodeIDs {
					_ = AllocateOrder(r.Context(), db, nid, pendingOrder, totals)
					if pendingOrder.Status == OrderStatusReady {
						allocOk = true
						break
					}
				}
			}
			if !allocOk {
				writeError(w, http.StatusConflict, "order for "+f.FactoryType+" exists but inputs not available (status: "+string(pendingOrder.Status)+")")
				return
			}
			orderID = pendingID
		}

		order, err := LoadOrderByID(r.Context(), db, orderID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := assignFacility(r.Context(), db, f.ID, order.ID, order.RecipeTicks); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"status": "started", "order_id": order.ID})
	}
}

// --- POST /econ2/facilities/{id}/stop ---

func stopFacilityHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		f, err := LoadFacilityByID(r.Context(), db, id)
		if err != nil {
			writeError(w, http.StatusNotFound, "facility not found")
			return
		}
		if f.PlayerID != playerID {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		// Cancel only THIS facility's current order (not all orders with same ID).
		if f.CurrentOrderID != nil {
			_, _ = db.Exec(r.Context(),
				`UPDATE econ2_orders SET status='cancelled', updated_at=now() WHERE id=$1`,
				*f.CurrentOrderID,
			)
		}

		// Set this facility to idle.
		_, err = db.Exec(r.Context(), `
			UPDATE econ2_facilities SET status='idle', current_order_id=NULL, updated_at=now()
			WHERE id=$1
		`, f.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
	}
}

// --- DELETE /econ2/facilities/{id} ---

func destroyFacilityHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		f, err := LoadFacilityByID(r.Context(), db, id)
		if err != nil {
			writeError(w, http.StatusNotFound, "facility not found")
			return
		}
		if f.PlayerID != playerID {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		if err := f.Destroy(r.Context(), db); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "destroyed"})
	}
}

// --- POST /econ2/orders ---

type createOrderRequest struct {
	StarID      string  `json:"star_id"`
	NodeID      string  `json:"node_id"`
	FactoryType string  `json:"factory_type"`
	ProductID   string  `json:"product_id"`
	OrderType   string  `json:"order_type"`
	TargetQty   float64 `json:"target_qty"`
	Priority    int     `json:"priority"`
}

func createOrderHandler(db *pgxpool.Pool, recipes RecipeBook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		var req createOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		starID, err := uuid.Parse(req.StarID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid star_id")
			return
		}
		nodeID, err := uuid.Parse(req.NodeID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid node_id")
			return
		}

		key := RecipeKey{ProductID: req.ProductID, FactoryType: req.FactoryType}
		recipe, ok := recipes[key]
		if !ok {
			writeError(w, http.StatusBadRequest, "unknown recipe for product/factory combination")
			return
		}

		priority := req.Priority
		if priority == 0 {
			priority = 5
		}

		order := &ProductionOrder{
			PlayerID:        playerID,
			StarID:          starID,
			NodeID:          nodeID,
			OrderType:       OrderType(req.OrderType),
			Status:          OrderStatusPending,
			RecipeID:        recipe.RecipeID,
			ProductID:       recipe.ProductID,
			FactoryType:     recipe.FactoryType,
			Inputs:          recipe.Inputs,
			BaseYield:       recipe.BaseYield,
			RecipeTicks:     recipe.Ticks,
			Efficiency:      recipe.Efficiency,
			TargetQty:       req.TargetQty,
			AllocatedInputs: map[string]float64{},
			Priority:        priority,
		}

		if err := CreateOrder(r.Context(), db, order); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Immediately attempt MRP + allocation.
		totals := map[string]float64{}
		visiting := map[string]bool{}
		if err := ResolveDemand(order.ProductID, order.TargetQty, order.FactoryType, recipes, totals, visiting); err == nil {
			_ = AllocateOrder(r.Context(), db, nodeID, order, totals)
		}

		writeJSON(w, http.StatusCreated, order)
	}
}

// --- GET /econ2/orders?node_id=... ---

func listOrdersHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		nodeIDStr := r.URL.Query().Get("node_id")
		if nodeIDStr == "" {
			writeError(w, http.StatusBadRequest, "node_id query param required")
			return
		}
		nodeID, err := uuid.Parse(nodeIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid node_id")
			return
		}

		orders, err := ListOrdersByNode(r.Context(), db, nodeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if orders == nil {
			orders = []*ProductionOrder{}
		}

		writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
	}
}

// --- DELETE /econ2/orders/{id} ---

func cancelOrderHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		order, err := LoadOrderByID(r.Context(), db, id)
		if err != nil {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		if order.PlayerID != playerID {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		_, err = db.Exec(r.Context(),
			`UPDATE econ2_orders SET status='cancelled', updated_at=now() WHERE id=$1`,
			id,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Release any facility that was running this order.
		_, _ = db.Exec(r.Context(), `
			UPDATE econ2_facilities SET status='idle', current_order_id=NULL, updated_at=now()
			WHERE current_order_id=$1
		`, id)

		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
	}
}

// --- POST /econ2/routes ---

type createRouteRequest struct {
	FromNodeID         string  `json:"from_node_id"`
	ToNodeID           string  `json:"to_node_id"`
	CapacityPerTick    float64 `json:"capacity_per_tick"`
	MinContinuousShare float64 `json:"min_continuous_share"`
}

func createRouteHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		var req createRouteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		fromNodeID, err := uuid.Parse(req.FromNodeID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid from_node_id")
			return
		}
		toNodeID, err := uuid.Parse(req.ToNodeID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid to_node_id")
			return
		}

		share := req.MinContinuousShare
		if share <= 0 {
			share = 0.20
		}

		route := &Route{
			PlayerID:           playerID,
			FromNodeID:         fromNodeID,
			ToNodeID:           toNodeID,
			CapacityPerTick:    req.CapacityPerTick,
			MinContinuousShare: share,
			Status:             RouteStatusActive,
		}

		if err := CreateRoute(r.Context(), db, route); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, route)
	}
}

// --- GET /econ2/routes ---

func listRoutesHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		routes, err := ListRoutesByPlayer(r.Context(), db, playerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if routes == nil {
			routes = []Route{}
		}

		writeJSON(w, http.StatusOK, map[string]any{"routes": routes})
	}
}

// --- GET /econ2/stock?node_id=... ---

func getStockHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		nodeIDStr := r.URL.Query().Get("node_id")
		if nodeIDStr == "" {
			writeError(w, http.StatusBadRequest, "node_id query param required")
			return
		}
		nodeID, err := uuid.Parse(nodeIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid node_id")
			return
		}

		stock, err := NodeStock(r.Context(), db, nodeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"stock": stock})
	}
}

// --- POST /econ2/bootstrap ---

func bootstrapHandler(db *pgxpool.Pool, cfg BootstrapConfig, recipes RecipeBook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		var req struct {
			StarID string `json:"star_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		starID, err := uuid.Parse(req.StarID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid star_id")
			return
		}

		result, err := RunBootstrap(r.Context(), db, playerID, starID, cfg, recipes)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, result)
	}
}

// --- POST /econ2/nodes ---

type getOrCreateNodeRequest struct {
	StarID   string  `json:"star_id"`
	PlanetID *string `json:"planet_id"`
}

func getOrCreateNodeHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		var req getOrCreateNodeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		starID, err := uuid.Parse(req.StarID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid star_id")
			return
		}

		var planetID *uuid.UUID
		if req.PlanetID != nil {
			pid, err := uuid.Parse(*req.PlanetID)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid planet_id")
				return
			}
			planetID = &pid
		}

		nodeID, err := GetOrCreateNode(r.Context(), db, playerID, starID, planetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"node_id": nodeID})
	}
}

// --- GET /econ2/my-nodes ---

type myNodeEntry struct {
	NodeID        uuid.UUID  `json:"node_id"`
	StarID        uuid.UUID  `json:"star_id"`
	PlanetID      *uuid.UUID `json:"planet_id"`
	Level         string     `json:"level"`
	StarType      string     `json:"star_type"`
	X             float64    `json:"x"`
	Y             float64    `json:"y"`
	FacilityCount int        `json:"facility_count"`
}

func listMyNodesHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		rows, err := db.Query(r.Context(), `
			SELECT
				n.id, n.star_id, n.planet_id, n.level,
				s.star_type, s.x, s.y,
				COUNT(f.id) AS facility_count
			FROM econ2_nodes n
			JOIN stars s ON s.id = n.star_id
			LEFT JOIN econ2_facilities f ON f.node_id = n.id AND f.status != 'destroyed'
			WHERE n.player_id = $1
			GROUP BY n.id, s.star_type, s.x, s.y
			ORDER BY n.created_at ASC
		`, playerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var nodes []myNodeEntry
		for rows.Next() {
			var e myNodeEntry
			if err := rows.Scan(&e.NodeID, &e.StarID, &e.PlanetID, &e.Level,
				&e.StarType, &e.X, &e.Y, &e.FacilityCount); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			nodes = append(nodes, e)
		}
		if nodes == nil {
			nodes = []myNodeEntry{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes})
	}
}

// --- POST /econ2/goals ---

type createGoalRequest struct {
	StarID    string  `json:"star_id"`
	ProductID string  `json:"product_id"`
	TargetQty float64 `json:"target_qty"`
}

func createGoalHandler(db *pgxpool.Pool, recipes RecipeBook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		var req createGoalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		starID, err := uuid.Parse(req.StarID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid star_id")
			return
		}
		if req.ProductID == "" || req.TargetQty <= 0 {
			writeError(w, http.StatusBadRequest, "product_id and target_qty required")
			return
		}

		// Determine next priority (max + 1).
		var maxPriority int
		_ = db.QueryRow(r.Context(),
			`SELECT COALESCE(MAX(priority), 0) FROM econ2_goals WHERE player_id=$1 AND status='active'`,
			playerID,
		).Scan(&maxPriority)

		goal := &Goal{
			ID:                 uuid.New(),
			PlayerID:           playerID,
			StarID:             starID,
			ProductID:          req.ProductID,
			TargetQty:          req.TargetQty,
			Priority:           maxPriority + 1,
			Status:             "active",
			TransportOverrides: map[string]string{},
			CreatedAt:          time.Now().UTC(),
		}

		overridesJSON, _ := json.Marshal(goal.TransportOverrides)
		_, err = db.Exec(r.Context(), `
			INSERT INTO econ2_goals (id, player_id, star_id, product_id, target_qty, priority, status, transport_overrides, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, goal.ID, goal.PlayerID, goal.StarID, goal.ProductID, goal.TargetQty,
			goal.Priority, goal.Status, overridesJSON, goal.CreatedAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Get or create node for this star.
		nodeID, err := GetOrCreateNode(r.Context(), db, playerID, starID, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not resolve node: "+err.Error())
			return
		}

		// Walk recipe tree and create batch orders for each production step.
		var createdOrders []*ProductionOrder
		visiting := map[string]bool{}
		_, _ = walkRecipeTree(req.ProductID, req.TargetQty, recipes, visiting, func(node *WalkNode) {
			if node.Recipe == nil || node.Recipe.IsExtractor() {
				return // extractors are handled by continuous orders separately
			}

			goalID := goal.ID
			order := &ProductionOrder{
				PlayerID:        playerID,
				StarID:          starID,
				NodeID:          nodeID,
				OrderType:       OrderTypeBatch,
				Status:          OrderStatusPending,
				RecipeID:        node.Recipe.RecipeID,
				ProductID:       node.Recipe.ProductID,
				FactoryType:     node.Recipe.FactoryType,
				Inputs:          node.Recipe.Inputs,
				BaseYield:       node.Recipe.BaseYield,
				RecipeTicks:     node.Recipe.Ticks,
				Efficiency:      node.Recipe.Efficiency,
				TargetQty:       node.Qty,
				AllocatedInputs: map[string]float64{},
				Priority:        goal.Priority,
				GoalID:          &goalID,
			}
			if err := CreateOrder(r.Context(), db, order); err == nil {
				createdOrders = append(createdOrders, order)
			}
		})

		writeJSON(w, http.StatusCreated, map[string]any{
			"goal":   goal,
			"orders": createdOrders,
		})
	}
}

// --- GET /econ2/goals ---

func listGoalsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		rows, err := db.Query(r.Context(), `
			SELECT id, player_id, star_id, product_id, target_qty, priority, status, transport_overrides, created_at
			FROM econ2_goals
			WHERE player_id=$1 AND status='active'
			ORDER BY priority ASC, created_at ASC
		`, playerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var goals []*Goal
		for rows.Next() {
			var g Goal
			var overridesRaw []byte
			if err := rows.Scan(&g.ID, &g.PlayerID, &g.StarID, &g.ProductID, &g.TargetQty,
				&g.Priority, &g.Status, &overridesRaw, &g.CreatedAt); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			g.TransportOverrides = map[string]string{}
			_ = json.Unmarshal(overridesRaw, &g.TransportOverrides)
			goals = append(goals, &g)
		}
		if goals == nil {
			goals = []*Goal{}
		}

		writeJSON(w, http.StatusOK, map[string]any{"goals": goals})
	}
}

// --- DELETE /econ2/goals/{id} ---

func deleteGoalHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		// Verify ownership.
		var ownerID uuid.UUID
		if err := db.QueryRow(r.Context(), `SELECT player_id FROM econ2_goals WHERE id=$1`, id).Scan(&ownerID); err != nil {
			writeError(w, http.StatusNotFound, "goal not found")
			return
		}
		if ownerID != playerID {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		// Cancel open orders belonging to this goal.
		_, _ = db.Exec(r.Context(), `
			UPDATE econ2_orders SET status='cancelled', updated_at=now()
			WHERE goal_id=$1 AND status NOT IN ('completed', 'cancelled')
		`, id)

		// Mark goal cancelled.
		_, err = db.Exec(r.Context(), `UPDATE econ2_goals SET status='cancelled' WHERE id=$1`, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
	}
}

// --- PATCH /econ2/goals/reorder ---

func reorderGoalsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		var req struct {
			IDs []string `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		tx, err := db.Begin(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer func() { _ = tx.Rollback(r.Context()) }()

		for i, idStr := range req.IDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id: "+idStr)
				return
			}
			if _, err := tx.Exec(r.Context(),
				`UPDATE econ2_goals SET priority=$1 WHERE id=$2 AND player_id=$3`,
				i+1, id, playerID,
			); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// --- GET /econ2/stock-all ---

func stockAllHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		rows, err := db.Query(r.Context(), `
			SELECT item_id, SUM(total) AS total, SUM(allocated) AS allocated
			FROM econ2_item_stock
			WHERE node_id IN (SELECT id FROM econ2_nodes WHERE player_id=$1)
			GROUP BY item_id
			ORDER BY item_id
		`, playerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type stockEntry struct {
			ItemID    string  `json:"item_id"`
			Total     float64 `json:"total"`
			Allocated float64 `json:"allocated"`
			Available float64 `json:"available"`
		}
		var stock []stockEntry
		for rows.Next() {
			var s stockEntry
			if err := rows.Scan(&s.ItemID, &s.Total, &s.Allocated); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			s.Available = s.Total - s.Allocated
			stock = append(stock, s)
		}
		if stock == nil {
			stock = []stockEntry{}
		}

		writeJSON(w, http.StatusOK, map[string]any{"stock": stock})
	}
}

// --- GET /econ2/facilities-all ---

func facilitiesAllHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		rows, err := db.Query(r.Context(), `
			SELECT f.id, f.player_id, f.star_id, f.node_id, f.factory_type, f.status, f.config, f.current_order_id,
			       n.planet_id
			FROM econ2_facilities f
			JOIN econ2_nodes n ON n.id = f.node_id
			WHERE f.player_id=$1 AND f.status != 'destroyed'
			ORDER BY f.star_id, f.created_at ASC
		`, playerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var facilities []*Facility
		for rows.Next() {
			var f Facility
			var cfgRaw []byte
			if err := rows.Scan(
				&f.ID, &f.PlayerID, &f.StarID, &f.NodeID,
				&f.FactoryType, &f.Status, &cfgRaw, &f.CurrentOrderID,
				&f.PlanetID,
			); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			_ = json.Unmarshal(cfgRaw, &f.Config)
			facilities = append(facilities, &f)
		}
		if facilities == nil {
			facilities = []*Facility{}
		}

		writeJSON(w, http.StatusOK, map[string]any{"facilities": facilities})
	}
}

// --- GET /econ2/orders-all ---

func ordersAllHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		rows, err := db.Query(r.Context(), `
			SELECT id, player_id, star_id, node_id, facility_id,
			       order_type, status, recipe_id, product_id, factory_type,
			       inputs, base_yield, recipe_ticks, efficiency,
			       target_qty, allocated_inputs, produced_qty, priority, goal_id
			FROM econ2_orders
			WHERE player_id=$1 AND status NOT IN ('completed', 'cancelled')
			ORDER BY priority ASC, created_at ASC
		`, playerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		orders, err := scanOrders(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if orders == nil {
			orders = []*ProductionOrder{}
		}

		writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
	}
}

// --- GET /econ2/deposits?star_id=... ---
// Returns planets.resource_deposits for the star's home planet (first by orbit_index).

func depositsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		starIDStr := r.URL.Query().Get("star_id")
		if starIDStr == "" {
			writeError(w, http.StatusBadRequest, "star_id query param required")
			return
		}
		starID, err := uuid.Parse(starIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid star_id")
			return
		}

		planetID, err := FindHomePlanet(r.Context(), db, starID)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"deposits": map[string]any{}})
			return
		}

		deposits, err := ReadAllDeposits(r.Context(), db, *planetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"planet_id": planetID,
			"deposits":  deposits,
		})
	}
}
