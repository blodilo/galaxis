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
	// System-level
	r.Get("/economy/system/{starId}", getEconomySystem(db, reg))
	r.Post("/economy/system/{starId}/build", buildFacility(db, reg))
	r.Post("/economy/system/{starId}/facilities/{facilityId}/recipe", assignRecipe(db, reg))
	r.Get("/economy/system/{starId}/log", getProductionLog(db))
	r.Get("/economy/system/{starId}/events", streamTickEvents(bus))
	r.Get("/economy/system/{starId}/surveys", getSystemSurveys(db))

	// Planet-level
	r.Post("/economy/planets/{planetId}/survey", executeSurvey(db, reg))
	r.Get("/economy/planets/{planetId}/survey", getSurvey(db))

	// Admin
	r.Post("/admin/tick/advance", advanceTick(eng))
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

// --- GET /economy/system/:starId --------------------------------------------

type systemResponse struct {
	StarID           string                        `json:"star_id"`
	LastTickN        int64                         `json:"last_tick_n"`
	Storage          economy.StorageContents       `json:"storage"`
	Facilities       []facilityResponse            `json:"facilities"`
	OrbitalSlotsUsed int                           `json:"orbital_slots_used"`
	OrbitalSlotsMax  int                           `json:"orbital_slots_max"`
	Surveys          []*economy.PlayerSurvey       `json:"surveys"`
}

type facilityResponse struct {
	ID           string                  `json:"id"`
	FacilityType string                  `json:"facility_type"`
	PlanetID     *string                 `json:"planet_id,omitempty"`
	Status       string                  `json:"status"`
	Config       economy.FacilityConfig  `json:"config"`
}

func getEconomySystem(db *pgxpool.Pool, reg *economy.Registries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		starID, err := uuid.Parse(chi.URLParam(r, "starId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starId")
			return
		}
		playerID := playerIDFromRequest(r)

		storage, err := economy.GetStorage(r.Context(), db, playerID, starID)
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

		resp := systemResponse{
			StarID:           starID.String(),
			Storage:          storage,
			Facilities:       toFacilityResponses(facilities),
			OrbitalSlotsUsed: orbitalUsed,
			OrbitalSlotsMax:  8, // game-params: orbital_slots — TODO wire from cfg
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
			TicksRemaining: 1,
			DepositID:      req.DepositID,
		}
		cfgRaw, _ := json.Marshal(cfg)

		var facilityID uuid.UUID
		err = db.QueryRow(r.Context(),
			`INSERT INTO facilities
			   (player_id, star_id, planet_id, facility_type, status, config)
			 VALUES ($1, $2, $3, $4, 'idle', $5)
			 RETURNING id`,
			playerID, starID, planetID, req.FacilityType, cfgRaw,
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
				fmt.Fprintf(w, "data: %s\n\n", data)
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
		`SELECT id, player_id, star_id, planet_id, facility_type, status, config
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
			&f.FacilityType, &f.Status, &rawConf); err != nil {
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
	var qualities map[string]float64
	if err := json.Unmarshal(raw, &qualities); err != nil {
		return nil, fmt.Errorf("planet resource_deposits parse: %w", err)
	}
	return qualities, nil
}

// sseTimeout replaces the router's global 60s timeout for SSE handlers.
// It returns a deadline-free context that cancels when the client disconnects.
func sseTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-r.Context().Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
