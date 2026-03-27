package economy2

import (
	"encoding/json"
	"net/http"

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

// RegisterRoutes mounts all economy2 REST endpoints on the given router.
func RegisterRoutes(r chi.Router, db *pgxpool.Pool, recipes RecipeBook, bootstrapCfg BootstrapConfig) {
	r.Post("/econ2/facilities", createFacilityHandler(db))
	r.Get("/econ2/facilities", listFacilitiesHandler(db))
	r.Delete("/econ2/facilities/{id}", destroyFacilityHandler(db))

	r.Post("/econ2/orders", createOrderHandler(db, recipes))
	r.Get("/econ2/orders", listOrdersHandler(db))
	r.Delete("/econ2/orders/{id}", cancelOrderHandler(db))

	r.Post("/econ2/routes", createRouteHandler(db))
	r.Get("/econ2/routes", listRoutesHandler(db))

	r.Get("/econ2/stock", getStockHandler(db))
	r.Post("/econ2/nodes", getOrCreateNodeHandler(db))

	r.Post("/econ2/bootstrap", bootstrapHandler(db, bootstrapCfg))
}

// --- POST /econ2/facilities ---

type createFacilityRequest struct {
	StarID        string  `json:"star_id"`
	PlanetID      *string `json:"planet_id"`
	FactoryType   string  `json:"factory_type"`
	Level         int     `json:"level"`
	DepositGoodID string  `json:"deposit_good_id"`
}

func createFacilityHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)
		if playerID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "missing player id")
			return
		}

		var req createFacilityRequest
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

		f := &Facility{
			PlayerID:    playerID,
			StarID:      starID,
			PlanetID:    planetID,
			NodeID:      nodeID,
			FactoryType: req.FactoryType,
			Status:      "idle",
			Config:      FacilityConfig{Level: req.Level, DepositGoodID: req.DepositGoodID},
		}

		if err := CreateFacility(r.Context(), db, f); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
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

		rows, err := db.Query(r.Context(), `
			SELECT id, player_id, star_id, planet_id, node_id, factory_type, status, config, current_order_id
			FROM econ2_facilities
			WHERE player_id=$1 AND star_id=$2
			ORDER BY created_at ASC
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
				&f.ID, &f.PlayerID, &f.StarID, &f.PlanetID, &f.NodeID,
				&f.FactoryType, &f.Status, &cfgRaw, &f.CurrentOrderID,
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

func bootstrapHandler(db *pgxpool.Pool, cfg BootstrapConfig) http.HandlerFunc {
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

		result, err := RunBootstrap(r.Context(), db, playerID, starID, cfg)
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
