package api

import (
	"net/http"
	"os"

	"galaxis/internal/config"

	"gopkg.in/yaml.v3"
)

// morphologyTemplate mirrors the relevant fields of galaxy_morphology_catalog_v1.0.yaml.
type morphologyTemplate struct {
	ID           string `yaml:"id"                  json:"id"`
	Enabled      bool   `yaml:"enabled"             json:"enabled"`
	Name         string `yaml:"name"                json:"name"`
	Designation  string `yaml:"designation"         json:"designation"`
	HubbleType   string `yaml:"hubble_type"         json:"hubble_type"`
	HubbleDesc   string `yaml:"hubble_description"  json:"hubble_description"`
	File         string `yaml:"file"                json:"file"`
	Orientation  string `yaml:"orientation"         json:"orientation"`
	Credit       string `yaml:"credit"              json:"credit"`
	ResolutionPx []int  `yaml:"resolution_px"       json:"resolution_px"`
}

type morphologyCatalog struct {
	Templates []morphologyTemplate `yaml:"templates"`
}

// listMorphologies serves GET /api/v1/catalog/morphologies.
// Returns only enabled templates; adds thumbnail_url for frontend convenience.
func listMorphologies(catalogPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(catalogPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "morphology catalog not found")
			return
		}
		var cat morphologyCatalog
		if err := yaml.Unmarshal(data, &cat); err != nil {
			writeError(w, http.StatusInternalServerError, "morphology catalog parse error")
			return
		}
		type templateWithURL struct {
			morphologyTemplate
			ThumbnailURL string `json:"thumbnail_url"`
		}
		result := make([]templateWithURL, 0, len(cat.Templates))
		for _, t := range cat.Templates {
			if !t.Enabled {
				continue
			}
			result = append(result, templateWithURL{
				morphologyTemplate: t,
				ThumbnailURL:       "/assets/morphology/" + t.File,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"morphologies": result})
	}
}

// getDefaultParams serves GET /api/v1/params/defaults.
// Returns the current server config (loaded from game-params YAML) as JSON.
func getDefaultParams(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"galaxy":            cfg.Galaxy,
			"ftlw":              cfg.FTLW,
			"sensors":           cfg.Sensors,
			"time":              cfg.Time,
			"economy":           cfg.Economy,
			"planet_generation": cfg.PlanetGen,
			"research":          cfg.Research,
			"combat":            cfg.Combat,
			"server":            cfg.Server,
		})
	}
}
