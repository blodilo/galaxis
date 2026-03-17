package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"galaxis/internal/config"
	"galaxis/internal/db"
	"galaxis/internal/galaxy"
	"galaxis/internal/jobs"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
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
			ConfigDir:   runningCfg.ConfigDir,
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

// lookupMorphologyImagePath resolves the filesystem path for a morphology template.
// It reads the catalog YAML, finds the template by ID, verifies it is enabled,
// and returns the absolute path to the image file.
func lookupMorphologyImagePath(catalogPath, assetsDir, morphologyID string) (string, error) {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return "", fmt.Errorf("lookupMorphologyImagePath: read catalog %s: %w", catalogPath, err)
	}
	var cat morphologyCatalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return "", fmt.Errorf("lookupMorphologyImagePath: parse catalog: %w", err)
	}
	for _, t := range cat.Templates {
		if t.ID != morphologyID {
			continue
		}
		if !t.Enabled {
			return "", fmt.Errorf("lookupMorphologyImagePath: morphology %q is disabled", morphologyID)
		}
		return filepath.Join(assetsDir, "morphology", t.File), nil
	}
	return "", fmt.Errorf("lookupMorphologyImagePath: morphology %q not found in catalog", morphologyID)
}

// triggerStep1 handles POST /api/v1/generate/step1.
// Creates a new galaxy record and runs Step1Morphology asynchronously.
func triggerStep1(pool *pgxpool.Pool, runningCfg *config.Config, store *jobs.Store, assetsDir, catalogPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			Galaxy: req.Galaxy, FTLW: req.FTLW, Sensors: req.Sensors,
			Time: req.Time, Economy: req.Economy, PlanetGen: req.PlanetGen,
			Research: req.Research, Combat: req.Combat, Server: req.Server,
			DatabaseURL: runningCfg.DatabaseURL, RedisURL: runningCfg.RedisURL,
			ConfigDir: runningCfg.ConfigDir,
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
			cfgJSON, _ := json.Marshal(struct {
				MorphologyID string              `json:"morphology_id"`
				Galaxy       config.GalaxyConfig `json:"galaxy"`
			}{req.MorphologyID, genCfg.Galaxy})
			galaxyID, err := db.CreateGalaxy(ctx, pool, name, genCfg.Galaxy.Seed, cfgJSON)
			if err != nil {
				store.SetError(job.ID, "db: create galaxy: "+err.Error())
				return
			}
			jobID := job.ID
			emitFn := func(step string, done, total int, msg string) {
				store.Emit(jobID, step, done, total, msg)
			}
			imagePath, err := lookupMorphologyImagePath(catalogPath, assetsDir, req.MorphologyID)
			if err != nil {
				store.SetError(job.ID, "morphology image: "+err.Error())
				return
			}
			gen := galaxy.NewGenerator(genCfg, pool)
			if err := gen.Step1Morphology(ctx, galaxyID, imagePath, emitFn); err != nil {
				_ = db.SetGalaxyStatus(ctx, pool, galaxyID, "error")
				store.SetError(job.ID, err.Error())
				return
			}
			store.SetDone(job.ID, galaxyID)
		}()
		writeJSON(w, http.StatusAccepted, job)
	}
}

// triggerGalaxyStep handles POST /api/v1/galaxy/{galaxyID}/steps/{step}.
// Runs the specified step (spectral|objects|planets) on an existing galaxy.
func triggerGalaxyStep(pool *pgxpool.Pool, runningCfg *config.Config, store *jobs.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		galaxyID, err := uuid.Parse(chi.URLParam(r, "galaxyID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid galaxy id")
			return
		}
		step := chi.URLParam(r, "step")

		// Load stored config for this galaxy
		genCfg := &config.Config{
			Galaxy: runningCfg.Galaxy, FTLW: runningCfg.FTLW, Sensors: runningCfg.Sensors,
			Time: runningCfg.Time, Economy: runningCfg.Economy, PlanetGen: runningCfg.PlanetGen,
			Research: runningCfg.Research, Combat: runningCfg.Combat, Server: runningCfg.Server,
			DatabaseURL: runningCfg.DatabaseURL, RedisURL: runningCfg.RedisURL,
			ConfigDir: runningCfg.ConfigDir,
		}

		job := store.Create()
		go func() {
			ctx := context.Background()
			store.SetRunning(job.ID)
			jobID := job.ID
			emitFn := func(step string, done, total int, msg string) {
				store.Emit(jobID, step, done, total, msg)
			}
			gen := galaxy.NewGenerator(genCfg, pool)
			var stepErr error
			switch step {
			case "spectral":
				stepErr = gen.Step2Spectral(ctx, galaxyID, emitFn)
			case "objects":
				stepErr = gen.Step3Objects(ctx, galaxyID, emitFn)
			case "planets":
				stepErr = gen.Step4Planets(ctx, galaxyID, emitFn)
			default:
				store.SetError(job.ID, "unknown step: "+step)
				return
			}
			if stepErr != nil {
				_ = db.SetGalaxyStatus(ctx, pool, galaxyID, "error")
				store.SetError(job.ID, stepErr.Error())
				return
			}
			store.SetDone(job.ID, galaxyID)
		}()
		writeJSON(w, http.StatusAccepted, job)
	}
}

// handleDeleteGalaxy handles DELETE /api/v1/galaxy/{galaxyID}.
func handleDeleteGalaxy(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		galaxyID, err := uuid.Parse(chi.URLParam(r, "galaxyID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid galaxy id")
			return
		}
		if err := db.DeleteGalaxy(r.Context(), pool, galaxyID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
