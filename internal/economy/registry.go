// Package economy implements the production and resource economy system.
package economy

import (
	"fmt"
	"os"
	"path/filepath"

	"galaxis/internal/config"

	"gopkg.in/yaml.v3"
)

// --- Recipe types -----------------------------------------------------------

// Recipe describes a single production recipe loaded from recipes_v1.1.yaml.
type Recipe struct {
	ID             string             `yaml:"id"`
	Name           string             `yaml:"name"`
	FacilityType   string             `yaml:"facility_type"`
	OutputGood     string             `yaml:"output_good"`
	Tier           int                `yaml:"tier"`
	Ticks          int                `yaml:"ticks"`
	BaseEfficiency float64            `yaml:"base_efficiency"`
	Inputs         map[string]float64 `yaml:"inputs"`
	Outputs        map[string]float64 `yaml:"outputs"`
}

type recipesFile struct {
	Recipes []Recipe `yaml:"recipes"`
}

// RecipeRegistry is an in-memory lookup of all recipes by ID.
type RecipeRegistry map[string]*Recipe

// --- Deposit init types -----------------------------------------------------

// DepositSpec holds the initialisation parameters for one resource type.
type DepositSpec struct {
	BaseUnits   float64
	BaseMaxRate float64
	BaseSlots   int
}

// DepositRegistry maps good_id → DepositSpec.
type DepositRegistry map[string]DepositSpec

// --- Survey quality thresholds ----------------------------------------------

// SurveyThresholds holds the quality cut-offs that determine information depth.
type SurveyThresholds struct {
	TypeOnly    float64
	RangeApprox float64
	RangeNarrow float64
	Exact       float64
}

// --- Facility efficiency & output -------------------------------------------

// FacilityRegistry maps facility_type → per-level efficiency and output slices.
type FacilityRegistry struct {
	// Efficiency[facilityType][level-1] → η (0–1)
	Efficiency map[string][]float64
	// OutputPerTick[facilityType][level-1] → units/tick (mine only)
	OutputPerTick map[string][]int
	// BuildTicks[facilityType] → ticks to complete construction
	BuildTicks map[string]int
	// OutputGood[facilityType] → the good_id this facility type produces.
	// Derived from recipe output_good fields during LoadRegistries.
	// Empty for mine (deposit-specific) and assembler/elevator (special).
	OutputGood map[string]string
}

// Eta returns the efficiency for the given facility type and 1-based level.
// Returns 0 if the type or level is unknown.
func (fr *FacilityRegistry) Eta(facilityType string, level int) float64 {
	vals, ok := fr.Efficiency[facilityType]
	if !ok || level < 1 || level > len(vals) {
		return 0
	}
	return vals[level-1]
}

// --- Registries bundle ------------------------------------------------------

// Registries bundles all in-memory game-data registries for the economy system.
type Registries struct {
	Recipes          RecipeRegistry
	Deposits         DepositRegistry
	Facilities       FacilityRegistry
	SurveyThresholds SurveyThresholds
	DepositWarnings  config.DepositWarningsConfig
	StorageCapacity  map[string][]int // storage class → per-level capacity slices
}

// LoadRegistries reads recipes from recipesPath and game-params from cfg,
// and returns a fully populated Registries bundle ready for use by the tick engine.
func LoadRegistries(recipesPath string, cfg *config.Config) (*Registries, error) {
	recipes, err := loadRecipes(recipesPath)
	if err != nil {
		return nil, err
	}

	prod := cfg.Production

	deposits := make(DepositRegistry, len(prod.DepositInit))
	for goodID, d := range prod.DepositInit {
		deposits[goodID] = DepositSpec{
			BaseUnits:   d.BaseUnits,
			BaseMaxRate: d.BaseMaxRate,
			BaseSlots:   d.BaseSlots,
		}
	}

	qt := prod.Survey.QualityThresholds
	thresholds := SurveyThresholds{
		TypeOnly:    qt.TypeOnly,
		RangeApprox: qt.RangeApprox,
		RangeNarrow: qt.RangeNarrow,
		Exact:       qt.Exact,
	}

	// Build facilityType → output_good from recipe declarations.
	outputGood := make(map[string]string, len(recipes))
	for _, r := range recipes {
		if r.OutputGood != "" {
			outputGood[r.FacilityType] = r.OutputGood
		}
	}

	return &Registries{
		Recipes:  recipes,
		Deposits: deposits,
		Facilities: FacilityRegistry{
			Efficiency:    prod.FacilityEfficiency,
			OutputPerTick: prod.FacilityOutputPerTick,
			BuildTicks:    prod.FacilityBuildTicks,
			OutputGood:    outputGood,
		},
		SurveyThresholds: thresholds,
		DepositWarnings:  prod.DepositWarnings,
		StorageCapacity:  prod.StorageCapacityPerModule,
	}, nil
}

// loadRecipes parses the YAML recipe file and returns a RecipeRegistry.
func loadRecipes(path string) (RecipeRegistry, error) {
	// If path is relative, resolve from working directory (tests) or as-is.
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("economy: resolve recipes path: %w", err)
		}
		path = abs
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("economy: read recipes %s: %w", path, err)
	}

	var rf recipesFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("economy: parse recipes: %w", err)
	}

	reg := make(RecipeRegistry, len(rf.Recipes))
	for i := range rf.Recipes {
		r := &rf.Recipes[i]
		if r.ID == "" {
			return nil, fmt.Errorf("economy: recipe at index %d has no id", i)
		}
		if _, dup := reg[r.ID]; dup {
			return nil, fmt.Errorf("economy: duplicate recipe id %q", r.ID)
		}
		reg[r.ID] = r
	}

	return reg, nil
}
