package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"galaxis/internal/economy"
	"galaxis/internal/tick"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// registerEconomyRoutes mounts all economy-related REST and SSE endpoints.
// tickEngine is needed only for the admin/tick/advance route.
func registerEconomyRoutes(
	r chi.Router,
	db *pgxpool.Pool,
	reg *economy.Registries,
	bus *economy.Broadcaster,
	eng *tick.Engine,
) {
	// Player overview
	r.Get("/economy/my-systems", getMySystemsHandler(db))

	// Recipes
	r.Get("/economy/recipes", getRecipes(reg))

	// System-level
	r.Get("/economy/system/{starId}", getEconomySystem(db, reg))
	r.Post("/economy/system/{starId}/build", buildFacility(db, reg))
	r.Post("/economy/system/{starId}/facilities/{facilityId}/recipe", assignRecipe(db, reg))
	r.Get("/economy/system/{starId}/log", getProductionLog(db))
	r.Get("/economy/system/{starId}/events", streamTickEvents(bus))
	r.Get("/economy/system/{starId}/surveys", getSystemSurveys(db))

	// Production orders
	r.Post("/economy/system/{starId}/orders", createOrder(db, reg))
	r.Patch("/economy/system/{starId}/orders/{orderId}", updateOrder(db))
	r.Delete("/economy/system/{starId}/orders/{orderId}", cancelOrder(db))

	// Planet-level
	r.Post("/economy/planets/{planetId}/survey", executeSurvey(db, reg))
	r.Get("/economy/planets/{planetId}/survey", getSurvey(db))

	// Admin
	r.Post("/admin/tick/advance", advanceTick(eng))
	r.Post("/admin/home-planet", setupHomePlanet(db, reg))
}

// --- Player ID helper -------------------------------------------------------

// playerID extracts the player UUID from the X-Player-ID header.
// Falls back to PlayerZeroID when the header is absent (MVP mode).
func playerIDFromRequest(r *http.Request) uuid.UUID {
	if h := r.Header.Get("X-Player-ID"); h != "" {
		if id, err := uuid.Parse(h); err == nil {
			return id
		}
	}
	id, _ := uuid.Parse(economy.PlayerZeroID)
	return id
}

// --- GET /economy/my-systems ------------------------------------------------

type mySystemDTO struct {
	StarID        string `json:"star_id"`
	FacilityCount int    `json:"facility_count"`
	PlanetCount   int    `json:"planet_count"`  // distinct planets with facilities
	RunningCount  int    `json:"running_count"` // facilities currently running
}

func getMySystemsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := playerIDFromRequest(r)

		rows, err := db.Query(r.Context(), `
			SELECT
			  star_id,
			  COUNT(*)                                      AS facility_count,
			  COUNT(DISTINCT planet_id)                    AS planet_count,
			  COUNT(*) FILTER (WHERE status = 'running')   AS running_count
			FROM facilities
			WHERE player_id = $1
			GROUP BY star_id
			ORDER BY star_id`,
			playerID,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		result := make([]mySystemDTO, 0)
		for rows.Next() {
			var dto mySystemDTO
			if err := rows.Scan(&dto.StarID, &dto.FacilityCount, &dto.PlanetCount, &dto.RunningCount); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			result = append(result, dto)
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// --- GET /economy/recipes ---------------------------------------------------

type recipeDTO struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	FacilityType string             `json:"facility_type"`
	OutputGood   string             `json:"output_good"`
	Tier         int                `json:"tier"`
	Ticks        int                `json:"ticks"`
	Inputs       map[string]float64 `json:"inputs"`
	Outputs      map[string]float64 `json:"outputs"`
}

func getRecipes(reg *economy.Registries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := make([]recipeDTO, 0, len(reg.Recipes))
		for _, rec := range reg.Recipes {
			result = append(result, recipeDTO{
				ID:           rec.ID,
				Name:         rec.Name,
				FacilityType: rec.FacilityType,
				OutputGood:   rec.OutputGood,
				Tier:         rec.Tier,
				Ticks:        rec.Ticks,
				Inputs:       rec.Inputs,
				Outputs:      rec.Outputs,
			})
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// --- GET /economy/system/:starId --------------------------------------------

type storageNodeResponse struct {
	ID       string                  `json:"id"`
	Level    string                  `json:"level"`
	PlanetID *string                 `json:"planet_id,omitempty"`
	Capacity *float64                `json:"capacity,omitempty"`
	Storage  economy.StorageContents `json:"storage"`
}

type orderDTO struct {
	ID             string  `json:"id"`
	FacilityType   string  `json:"facility_type"`
	RecipeID       string  `json:"recipe_id"`
	Mode           string  `json:"mode"`
	BatchRemaining *int    `json:"batch_remaining,omitempty"`
	GoodID         *string `json:"good_id,omitempty"`
	MinStock       *float64 `json:"min_stock,omitempty"`
	TargetStock    *float64 `json:"target_stock,omitempty"`
	Priority       int     `json:"priority"`
	Active         bool    `json:"active"`
}

type systemResponse struct {
	StarID           string                  `json:"star_id"`
	LastTickN        int64                   `json:"last_tick_n"`
	StorageNodes     []storageNodeResponse   `json:"storage_nodes"`
	Facilities       []facilityResponse      `json:"facilities"`
	Orders           []orderDTO              `json:"orders"`
	OrbitalSlotsUsed int                     `json:"orbital_slots_used"`
	OrbitalSlotsMax  int                     `json:"orbital_slots_max"`
	Surveys          []*economy.PlayerSurvey `json:"surveys"`
}

type facilityResponse struct {
	ID             string                 `json:"id"`
	FacilityType   string                 `json:"facility_type"`
	PlanetID       *string                `json:"planet_id,omitempty"`
	Status         string                 `json:"status"`
	Config         economy.FacilityConfig `json:"config"`
	CurrentOrderID *string                `json:"current_order_id,omitempty"`
}

func getEconomySystem(db *pgxpool.Pool, reg *economy.Registries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		starID, err := uuid.Parse(chi.URLParam(r, "starId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starId")
			return
		}
		playerID := playerIDFromRequest(r)

		nodes, err := economy.GetSystemNodes(r.Context(), db, playerID, starID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage read failed")
			return
		}

		facilities, err := loadFacilitiesForSystem(r.Context(), db, playerID, starID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "facilities read failed")
			return
		}

		orbitalUsed := countOrbital(facilities)

		surveys, err := economy.GetSystemSurveys(r.Context(), db, playerID, starID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "surveys read failed")
			return
		}

		nodeResps := make([]storageNodeResponse, len(nodes))
		for i, n := range nodes {
			nr := storageNodeResponse{
				ID:       n.ID.String(),
				Level:    n.Level,
				Capacity: n.Capacity,
				Storage:  n.Storage,
			}
			if n.PlanetID != nil {
				s := n.PlanetID.String()
				nr.PlanetID = &s
			}
			nodeResps[i] = nr
		}

		orders, err := loadOrdersForSystem(r.Context(), db, playerID, starID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "orders read failed")
			return
		}

		resp := systemResponse{
			StarID:           starID.String(),
			StorageNodes:     nodeResps,
			Facilities:       toFacilityResponses(facilities),
			Orders:           orders,
			OrbitalSlotsUsed: orbitalUsed,
			OrbitalSlotsMax:  8,
			Surveys:          surveys,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// --- POST /economy/system/:starId/build ------------------------------------

type buildRequest struct {
	FacilityType string  `json:"facility_type"`
	PlanetID     *string `json:"planet_id"`
	Level        int     `json:"level"`
	DepositID    string  `json:"deposit_id,omitempty"` // required for mine
}

func buildTicks(reg *economy.Registries, facilityType string) int {
	if t, ok := reg.Facilities.BuildTicks[facilityType]; ok && t > 0 {
		return t
	}
	return 3 // fallback
}

func buildFacility(db *pgxpool.Pool, reg *economy.Registries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		starID, err := uuid.Parse(chi.URLParam(r, "starId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starId")
			return
		}
		playerID := playerIDFromRequest(r)

		var req buildRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if req.FacilityType == "" {
			writeError(w, http.StatusUnprocessableEntity, "facility_type required")
			return
		}
		if req.Level < 1 {
			req.Level = 1
		}

		var planetID *uuid.UUID
		if req.PlanetID != nil && *req.PlanetID != "" {
			pid, err := uuid.Parse(*req.PlanetID)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid planet_id")
				return
			}
			planetID = &pid
		}

		cfg := economy.FacilityConfig{
			Level:          req.Level,
			TicksRemaining: buildTicks(reg, req.FacilityType),
			DepositID:      req.DepositID,
		}
		cfgRaw, _ := json.Marshal(cfg)

		// Ensure storage node exists for this location.
		nodeID, err := economy.GetOrCreateNode(r.Context(), db, playerID, starID, planetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage node failed")
			return
		}

		var facilityID uuid.UUID
		err = db.QueryRow(r.Context(),
			`INSERT INTO facilities
			   (player_id, star_id, planet_id, facility_type, status, config, storage_node_id)
			 VALUES ($1, $2, $3, $4, 'building', $5, $6)
			 RETURNING id`,
			playerID, starID, planetID, req.FacilityType, cfgRaw, nodeID,
		).Scan(&facilityID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "facility creation failed")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"id": facilityID.String()})
	}
}

// --- POST /economy/system/:starId/facilities/:facilityId/recipe ------------

type assignRecipeRequest struct {
	RecipeID string `json:"recipe_id"`
}

func assignRecipe(db *pgxpool.Pool, reg *economy.Registries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		facilityID, err := uuid.Parse(chi.URLParam(r, "facilityId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid facilityId")
			return
		}
		playerID := playerIDFromRequest(r)

		var req assignRecipeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}

		if _, ok := reg.Recipes[req.RecipeID]; !ok {
			writeError(w, http.StatusUnprocessableEntity, "unknown recipe_id")
			return
		}

		// Read current config, update recipe + reset batch counter.
		var rawCfg []byte
		err = db.QueryRow(r.Context(),
			`SELECT config FROM facilities WHERE id = $1 AND player_id = $2`,
			facilityID, playerID,
		).Scan(&rawCfg)
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "facility not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		var cfg economy.FacilityConfig
		if err := json.Unmarshal(rawCfg, &cfg); err != nil {
			writeError(w, http.StatusInternalServerError, "config parse failed")
			return
		}
		cfg.RecipeID = req.RecipeID
		cfg.TicksRemaining = reg.Recipes[req.RecipeID].Ticks
		cfg.EfficiencyAcc = 0

		newCfgRaw, _ := json.Marshal(cfg)
		_, err = db.Exec(r.Context(),
			`UPDATE facilities
			 SET config = $1, status = 'running', updated_at = now()
			 WHERE id = $2 AND player_id = $3`,
			newCfgRaw, facilityID, playerID,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "running"})
	}
}

// --- GET /economy/system/:starId/log ---------------------------------------

func getProductionLog(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		starID, err := uuid.Parse(chi.URLParam(r, "starId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starId")
			return
		}
		playerID := playerIDFromRequest(r)

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		rows, err := economy.GetLog(r.Context(), db, playerID, starID, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

// --- GET /economy/system/:starId/events (SSE) -------------------------------

func streamTickEvents(bus *economy.Broadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		starID := chi.URLParam(r, "starId")

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		sseCtx, sseCancel := context.WithCancel(context.Background())
		defer sseCancel()
		go func() {
			select {
			case <-r.Context().Done():
				sseCancel()
			case <-sseCtx.Done():
			}
		}()

		ch := bus.Subscribe()
		defer bus.Unsubscribe(ch)

		for {
			select {
			case ev, open := <-ch:
				if !open {
					return
				}
				if ev.StarID != starID {
					continue // filter to this system
				}
				data, _ := json.Marshal(ev)
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-sseCtx.Done():
				return
			}
		}
	}
}

// --- POST /economy/planets/:planetId/survey --------------------------------

type surveyRequest struct {
	Quality float64 `json:"quality"`
}

func executeSurvey(db *pgxpool.Pool, reg *economy.Registries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		planetID, err := uuid.Parse(chi.URLParam(r, "planetId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid planetId")
			return
		}
		playerID := playerIDFromRequest(r)

		var req surveyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if req.Quality <= 0 || req.Quality > 1 {
			writeError(w, http.StatusUnprocessableEntity, "quality must be 0.0–1.0")
			return
		}

		// Load resource qualities from planets table.
		resourceQualities, err := loadPlanetResourceQualities(r.Context(), db, planetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "planet read failed")
			return
		}

		survey, err := economy.ExecuteSurvey(
			r.Context(), db, playerID, planetID,
			req.Quality, 0, // tickN=0 for MVP (no tick counter in handler context)
			resourceQualities, reg,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, survey)
	}
}

// --- GET /economy/planets/:planetId/survey ---------------------------------

func getSurvey(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		planetID, err := uuid.Parse(chi.URLParam(r, "planetId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid planetId")
			return
		}
		playerID := playerIDFromRequest(r)

		survey, err := economy.GetSurvey(r.Context(), db, playerID, planetID)
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "no survey found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, survey)
	}
}

// --- GET /economy/system/:starId/surveys -----------------------------------

func getSystemSurveys(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		starID, err := uuid.Parse(chi.URLParam(r, "starId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starId")
			return
		}
		playerID := playerIDFromRequest(r)

		surveys, err := economy.GetSystemSurveys(r.Context(), db, playerID, starID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, surveys)
	}
}

// --- POST /admin/tick/advance -----------------------------------------------

func advanceTick(eng *tick.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eng.Advance(r.Context())
		writeJSON(w, http.StatusOK, map[string]string{"status": "advanced"})
	}
}

// --- Internal helpers -------------------------------------------------------

func loadFacilitiesForSystem(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID uuid.UUID,
) ([]*economy.Facility, error) {
	rows, err := db.Query(ctx,
		`SELECT id, player_id, star_id, planet_id, facility_type, status, config, storage_node_id, current_order_id
		 FROM facilities
		 WHERE player_id = $1 AND star_id = $2`,
		playerID, starID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*economy.Facility
	for rows.Next() {
		var (
			f       economy.Facility
			rawConf []byte
		)
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.StarID, &f.PlanetID,
			&f.FacilityType, &f.Status, &rawConf, &f.StorageNodeID, &f.CurrentOrderID); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawConf, &f.Config); err != nil {
			return nil, err
		}
		result = append(result, &f)
	}
	return result, rows.Err()
}

func toFacilityResponses(facilities []*economy.Facility) []facilityResponse {
	resp := make([]facilityResponse, len(facilities))
	for i, f := range facilities {
		r := facilityResponse{
			ID:           f.ID.String(),
			FacilityType: f.FacilityType,
			Status:       f.Status,
			Config:       f.Config,
		}
		if f.PlanetID != nil {
			s := f.PlanetID.String()
			r.PlanetID = &s
		}
		if f.CurrentOrderID != nil {
			s := f.CurrentOrderID.String()
			r.CurrentOrderID = &s
		}
		resp[i] = r
	}
	return resp
}

func countOrbital(facilities []*economy.Facility) int {
	n := 0
	for _, f := range facilities {
		if f.PlanetID == nil {
			n++
		}
	}
	return n
}

func loadPlanetResourceQualities(
	ctx context.Context,
	db *pgxpool.Pool,
	planetID uuid.UUID,
) (map[string]float64, error) {
	var raw []byte
	err := db.QueryRow(ctx,
		`SELECT resource_deposits FROM planets WHERE id = $1`, planetID,
	).Scan(&raw)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("planet %s not found", planetID)
	}
	if err != nil {
		return nil, err
	}

	// Support both formats:
	//   v1 (old): {"iron": 0.8, ...}
	//   v2 (migration 014): {"iron": {"amount":40000,"quality":0.8,"max_mines":4}, ...}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return nil, fmt.Errorf("planet resource_deposits parse: %w", err)
	}
	qualities := make(map[string]float64, len(rawMap))
	for k, v := range rawMap {
		var entry struct {
			Quality float64 `json:"quality"`
		}
		if json.Unmarshal(v, &entry) == nil && entry.Quality > 0 {
			qualities[k] = entry.Quality
			continue
		}
		var q float64
		if json.Unmarshal(v, &q) == nil {
			qualities[k] = q
		}
	}
	return qualities, nil
}

// --- POST /admin/home-planet ------------------------------------------------

type homePlanetRequest struct {
	PlanetID string `json:"planet_id"`
	StarID   string `json:"star_id"`
}

// setupHomePlanet sets up a planet as the player's home world:
//  1. Sets all known deposit resources to quality=1.0 on the planet
//  2. Initialises/overwrites planet_deposits with full-quality deposits
//  3. Creates a quality=1.0 survey for the player
//  4. Creates one facility of each type (mines running, others idle)
//  5. Seeds system_storage with starter materials
func setupHomePlanet(db *pgxpool.Pool, reg *economy.Registries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req homePlanetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		planetID, err := uuid.Parse(req.PlanetID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid planet_id")
			return
		}
		starID, err := uuid.Parse(req.StarID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid star_id")
			return
		}
		playerID := playerIDFromRequest(r)
		ctx := r.Context()

		// Guard: reject if facilities already exist for this player+star.
		var existingCount int
		if err := db.QueryRow(ctx,
			`SELECT COUNT(*) FROM facilities WHERE player_id = $1 AND star_id = $2`,
			playerID, starID,
		).Scan(&existingCount); err != nil {
			writeError(w, http.StatusInternalServerError, "facility check failed: "+err.Error())
			return
		}
		if existingCount > 0 {
			writeError(w, http.StatusConflict, "Heimatplanet für dieses System bereits eingerichtet")
			return
		}

		// 1. Build v2 deposit map: {good_id: {amount, quality, max_mines}} for all known deposit resources.
		type depositV2 struct {
			Amount   float64 `json:"amount"`
			Quality  float64 `json:"quality"`
			MaxMines int     `json:"max_mines"`
		}
		depositsV2 := make(map[string]depositV2, len(reg.Deposits))
		for goodID, spec := range reg.Deposits {
			depositsV2[goodID] = depositV2{
				Amount:   spec.BaseUnits,
				Quality:  1.0,
				MaxMines: spec.BaseSlots,
			}
		}

		// 2. Overwrite planets.resource_deposits with v2 format.
		rawQ, _ := json.Marshal(depositsV2)
		if _, err := db.Exec(ctx,
			`UPDATE planets SET resource_deposits = $1 WHERE id = $2`,
			rawQ, planetID,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "planet update failed: "+err.Error())
			return
		}

		// 3. planet_deposits table was dropped in migration 014 — deposits live in planets.resource_deposits.

		// 4. Upsert quality=1.0 survey directly (bypasses defunct planet_deposits table).
		//    At quality=1.0 all fields are revealed: exact amount, max_rate, slots.
		type resourceSnap struct {
			Present        bool     `json:"present"`
			RemainingExact *float64 `json:"remaining_exact,omitempty"`
			MaxRate        *float64 `json:"max_rate,omitempty"`
			Slots          *int     `json:"slots,omitempty"`
		}
		snapshot := make(map[string]resourceSnap, len(reg.Deposits))
		for goodID, spec := range reg.Deposits {
			remaining := spec.BaseUnits
			maxRate := spec.BaseMaxRate
			slots := spec.BaseSlots
			snapshot[goodID] = resourceSnap{
				Present:        true,
				RemainingExact: &remaining,
				MaxRate:        &maxRate,
				Slots:          &slots,
			}
		}
		rawSnap, _ := json.Marshal(snapshot)
		if _, err := db.Exec(ctx,
			`INSERT INTO player_surveys (player_id, planet_id, tick_n, quality, snapshot, surveyed_at)
			 VALUES ($1, $2, $3, $4, $5, now())
			 ON CONFLICT (player_id, planet_id) DO UPDATE
			   SET tick_n = EXCLUDED.tick_n, quality = EXCLUDED.quality,
			       snapshot = EXCLUDED.snapshot, surveyed_at = now()`,
			playerID, planetID, int64(0), float64(1.0), rawSnap,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "survey failed: "+err.Error())
			return
		}

		// 5. Remove existing facilities for this player+star to avoid duplicates.
		if _, err := db.Exec(ctx,
			`DELETE FROM facilities WHERE player_id = $1 AND star_id = $2`,
			playerID, starID,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "facility cleanup failed: "+err.Error())
			return
		}

		// 6. Ensure storage nodes exist for planet (planetary) and system (orbital).
		planetNodeID, err := economy.GetOrCreateNode(ctx, db, playerID, starID, &planetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "planet node failed: "+err.Error())
			return
		}
		orbitalNodeID, err := economy.GetOrCreateNode(ctx, db, playerID, starID, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "orbital node failed: "+err.Error())
			return
		}

		insertFacility := func(ftype, status string, pid *uuid.UUID, nodeID uuid.UUID, cfg economy.FacilityConfig) error {
			rawCfg, _ := json.Marshal(cfg)
			_, err := db.Exec(ctx,
				`INSERT INTO facilities (player_id, star_id, planet_id, facility_type, status, config, storage_node_id)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				playerID, starID, pid, ftype, status, rawCfg, nodeID,
			)
			return err
		}

		// 7. Create mines (one per deposit resource, running, planet node).
		for goodID := range reg.Deposits {
			cfg := economy.FacilityConfig{Level: 1, TicksRemaining: 1, DepositID: goodID}
			if err := insertFacility("mine", "running", &planetID, planetNodeID, cfg); err != nil {
				writeError(w, http.StatusInternalServerError, "mine creation failed: "+err.Error())
				return
			}
		}

		// 8. Create processing facilities on planet (idle, planet node).
		onPlanet := []string{"elevator", "steel_mill", "semiconductor_plant", "biosynth_lab", "precision_factory", "assembler"}
		for _, ftype := range onPlanet {
			cfg := economy.FacilityConfig{Level: 1, TicksRemaining: 1}
			if err := insertFacility(ftype, "idle", &planetID, planetNodeID, cfg); err != nil {
				writeError(w, http.StatusInternalServerError, ftype+" creation failed: "+err.Error())
				return
			}
		}

		// 9. Create shipyard orbital (orbital node).
		{
			cfg := economy.FacilityConfig{Level: 1, TicksRemaining: 1}
			if err := insertFacility("shipyard", "idle", nil, orbitalNodeID, cfg); err != nil {
				writeError(w, http.StatusInternalServerError, "shipyard creation failed: "+err.Error())
				return
			}
		}

		// 10. Seed planet node with starter materials.
		starter := economy.StorageContents{"steel": 200, "titansteel": 50}
		if err := economy.SetNodeStorage(ctx, db, planetNodeID, starter); err != nil {
			writeError(w, http.StatusInternalServerError, "storage seed failed: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"deposits": len(reg.Deposits),
			"survey":   planetID,
		})
	}
}

// --- Production orders CRUD -------------------------------------------------

type createOrderRequest struct {
	FacilityType   string   `json:"facility_type"`
	RecipeID       string   `json:"recipe_id"`
	Mode           string   `json:"mode"`
	BatchRemaining *int     `json:"batch_remaining,omitempty"`
	GoodID         *string  `json:"good_id,omitempty"`
	MinStock       *float64 `json:"min_stock,omitempty"`
	TargetStock    *float64 `json:"target_stock,omitempty"`
	Priority       int      `json:"priority"`
}

func createOrder(db *pgxpool.Pool, reg *economy.Registries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		starID, err := uuid.Parse(chi.URLParam(r, "starId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starId")
			return
		}
		playerID := playerIDFromRequest(r)

		var req createOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}

		if req.FacilityType == "" || req.RecipeID == "" {
			writeError(w, http.StatusUnprocessableEntity, "facility_type and recipe_id required")
			return
		}
		validModes := map[string]bool{"continuous_full": true, "continuous_demand": true, "batch": true}
		if !validModes[req.Mode] {
			writeError(w, http.StatusUnprocessableEntity, "mode must be continuous_full, continuous_demand, or batch")
			return
		}
		if _, ok := reg.Recipes[req.RecipeID]; !ok {
			writeError(w, http.StatusUnprocessableEntity, "unknown recipe_id")
			return
		}

		var orderID uuid.UUID
		err = db.QueryRow(r.Context(), `
			INSERT INTO production_orders
			  (player_id, star_id, facility_type, recipe_id, mode, batch_remaining,
			   good_id, min_stock, target_stock, priority)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id`,
			playerID, starID, req.FacilityType, req.RecipeID, req.Mode,
			req.BatchRemaining, req.GoodID, req.MinStock, req.TargetStock, req.Priority,
		).Scan(&orderID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "order creation failed: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": orderID.String()})
	}
}

type updateOrderRequest struct {
	Priority *int  `json:"priority,omitempty"`
	Active   *bool `json:"active,omitempty"`
}

func updateOrder(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID, err := uuid.Parse(chi.URLParam(r, "orderId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid orderId")
			return
		}
		playerID := playerIDFromRequest(r)

		var req updateOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}

		if req.Priority != nil {
			if _, err := db.Exec(r.Context(), `
				UPDATE production_orders SET priority = $1, updated_at = now()
				WHERE id = $2 AND player_id = $3`,
				*req.Priority, orderID, playerID,
			); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if req.Active != nil {
			if _, err := db.Exec(r.Context(), `
				UPDATE production_orders SET active = $1, updated_at = now()
				WHERE id = $2 AND player_id = $3`,
				*req.Active, orderID, playerID,
			); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func cancelOrder(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID, err := uuid.Parse(chi.URLParam(r, "orderId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid orderId")
			return
		}
		playerID := playerIDFromRequest(r)

		// Deactivate the order and unassign all facilities currently executing it.
		tx, err := db.Begin(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer tx.Rollback(r.Context()) //nolint:errcheck

		if _, err := tx.Exec(r.Context(), `
			UPDATE production_orders SET active = false, updated_at = now()
			WHERE id = $1 AND player_id = $2`,
			orderID, playerID,
		); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := tx.Exec(r.Context(), `
			UPDATE facilities SET status = 'idle', current_order_id = NULL, updated_at = now()
			WHERE current_order_id = $1`,
			orderID,
		); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
	}
}

func loadOrdersForSystem(
	ctx context.Context,
	db *pgxpool.Pool,
	playerID, starID uuid.UUID,
) ([]orderDTO, error) {
	rows, err := db.Query(ctx, `
		SELECT id, facility_type, recipe_id, mode, batch_remaining,
		       good_id, min_stock, target_stock, priority, active
		FROM production_orders
		WHERE player_id = $1 AND star_id = $2
		ORDER BY priority DESC, created_at ASC`,
		playerID, starID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []orderDTO
	for rows.Next() {
		var o orderDTO
		if err := rows.Scan(&o.ID, &o.FacilityType, &o.RecipeID, &o.Mode,
			&o.BatchRemaining, &o.GoodID, &o.MinStock, &o.TargetStock,
			&o.Priority, &o.Active); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	if result == nil {
		result = []orderDTO{}
	}
	return result, rows.Err()
}
