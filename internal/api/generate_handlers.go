package api

import (
	"context"
	"encoding/json"
	"net/http"

	"galaxis/internal/config"
	"galaxis/internal/db"
	"galaxis/internal/galaxy"
	"galaxis/internal/jobs"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// generateRequest is the POST /api/v1/generate body.
// All config sections are pre-filled by the frontend from GET /params/defaults,
// so every field is expected to be present.
type generateRequest struct {
	Name         string              `json:"name"`
	MorphologyID string              `json:"morphology_id"`
	Galaxy       config.GalaxyConfig `json:"galaxy"`
	FTLW         config.FTLWConfig   `json:"ftlw"`
	Sensors      config.SensorsConfig `json:"sensors"`
	Time         config.TimeConfig   `json:"time"`
	Economy      config.EconomyConfig `json:"economy"`
	PlanetGen    config.PlanetGenConfig `json:"planet_generation"`
	Research     config.ResearchConfig `json:"research"`
	Combat       config.CombatConfig  `json:"combat"`
	Server       config.ServerConfig  `json:"server"`
}

// triggerGenerate handles POST /api/v1/generate.
// It validates the request, creates a job, and starts generation in a goroutine.
func triggerGenerate(pool *pgxpool.Pool, runningCfg *config.Config, store *jobs.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Pre-fill with server defaults so partial bodies still work.
		req := generateRequest{
			Galaxy:    runningCfg.Galaxy,
			FTLW:      runningCfg.FTLW,
			Sensors:   runningCfg.Sensors,
			Time:      runningCfg.Time,
			Economy:   runningCfg.Economy,
			PlanetGen: runningCfg.PlanetGen,
			Research:  runningCfg.Research,
			Combat:    runningCfg.Combat,
			Server:    runningCfg.Server,
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}

		genCfg := &config.Config{
			Galaxy:      req.Galaxy,
			FTLW:        req.FTLW,
			Sensors:     req.Sensors,
			Time:        req.Time,
			Economy:     req.Economy,
			PlanetGen:   req.PlanetGen,
			Research:    req.Research,
			Combat:      req.Combat,
			Server:      req.Server,
			DatabaseURL: runningCfg.DatabaseURL,
			RedisURL:    runningCfg.RedisURL,
		}
		if err := genCfg.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		name := req.Name
		if name == "" {
			name = req.Server.InstanceName
		}
		if name == "" {
			name = "Unnamed Galaxy"
		}

		job := store.Create()

		go func() {
			ctx := context.Background()
			store.SetRunning(job.ID)

			// Store morphology_id alongside galaxy config for BL-09.
			cfgJSON, _ := json.Marshal(struct {
				MorphologyID string              `json:"morphology_id"`
				Galaxy       config.GalaxyConfig `json:"galaxy"`
			}{req.MorphologyID, genCfg.Galaxy})

			galaxyID, err := db.CreateGalaxy(ctx, pool, name, genCfg.Galaxy.Seed, cfgJSON)
			if err != nil {
				store.SetError(job.ID, "db: create galaxy: "+err.Error())
				return
			}

			gen := galaxy.NewGenerator(genCfg, pool)
			if err := gen.Run(ctx, galaxyID); err != nil {
				_ = db.SetGalaxyStatus(ctx, pool, galaxyID, "error")
				store.SetError(job.ID, err.Error())
				return
			}

			store.SetDone(job.ID, galaxyID)
		}()

		writeJSON(w, http.StatusAccepted, job)
	}
}

// getGenerateStatus handles GET /api/v1/generate/{jobID}/status.
func getGenerateStatus(store *jobs.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobID")
		job, ok := store.Get(jobID)
		if !ok {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		writeJSON(w, http.StatusOK, job)
	}
}
