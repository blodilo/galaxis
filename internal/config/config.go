// Package config loads and validates the central game-params YAML file.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the top-level structure mirroring game-params_v1.1.yaml.
type Config struct {
	Galaxy    GalaxyConfig    `yaml:"galaxy"             json:"galaxy"`
	FTLW      FTLWConfig      `yaml:"ftlw"               json:"ftlw"`
	Sensors   SensorsConfig   `yaml:"sensors"            json:"sensors"`
	Time      TimeConfig      `yaml:"time"               json:"time"`
	Economy   EconomyConfig   `yaml:"economy"            json:"economy"`
	PlanetGen PlanetGenConfig `yaml:"planet_generation"  json:"planet_generation"`
	Research  ResearchConfig  `yaml:"research"           json:"research"`
	Combat    CombatConfig    `yaml:"combat"             json:"combat"`
	Server    ServerConfig    `yaml:"server"             json:"server"`

	// Runtime fields (not from YAML, set via env/flags or Load())
	DatabaseURL string `yaml:"-" json:"-"`
	RedisURL    string `yaml:"-" json:"-"`
	ConfigDir   string `yaml:"-" json:"-"` // directory containing the config file
}

type GalaxyConfig struct {
	Seed            int64        `yaml:"seed"              json:"seed"`
	NumStars        int          `yaml:"num_stars"         json:"num_stars"`
	RadiusLY        float64      `yaml:"radius_ly"         json:"radius_ly"`
	Type            string       `yaml:"type"              json:"type"`
	Arms            int          `yaml:"arms"              json:"arms"`
	ArmWinding      float64      `yaml:"arm_winding"       json:"arm_winding"`
	ArmSpread       float64      `yaml:"arm_spread"        json:"arm_spread"`
	SMBHMassSolar   float64      `yaml:"smbh_mass_solar"   json:"smbh_mass_solar"`
	ExoticCounts    ExoticCounts `yaml:"exotic_counts"     json:"exotic_counts"`
	ExoticPlacement string       `yaml:"exotic_placement"  json:"exotic_placement"`
}

// ExoticCounts configures how many of each exotic star type are placed per galaxy.
// These are placed in addition to num_stars (they do not count against the cap).
type ExoticCounts struct {
	WR        int `yaml:"wr"         json:"wr"`
	RStar     int `yaml:"rstar"      json:"rstar"`
	SStar     int `yaml:"sstar"      json:"sstar"`
	Pulsar    int `yaml:"pulsar"     json:"pulsar"`
	StellarBH int `yaml:"stellar_bh" json:"stellar_bh"`
}

type FTLWConfig struct {
	VacuumBase          float64 `yaml:"vacuum_base"           json:"vacuum_base"`
	KFactor             float64 `yaml:"k_factor"              json:"k_factor"`
	CutoffPercent       float64 `yaml:"cutoff_percent"        json:"cutoff_percent"`
	VoxelSizeLY         float64 `yaml:"voxel_size_ly"         json:"voxel_size_ly"`
	CoarseVoxelSizeLY   float64 `yaml:"coarse_voxel_size_ly"  json:"coarse_voxel_size_ly"`
	PulsarMultiplier    float64 `yaml:"pulsar_multiplier"     json:"pulsar_multiplier"`
	BlackHoleMultiplier float64 `yaml:"black_hole_multiplier" json:"black_hole_multiplier"`
}

type SensorsConfig struct {
	OpticalK                    float64            `yaml:"optical_k"                      json:"optical_k"`
	FTLK                        float64            `yaml:"ftl_k"                          json:"ftl_k"`
	ShipThermalK                float64            `yaml:"ship_thermal_k"                 json:"ship_thermal_k"`
	SensorRatings               map[string]float64 `yaml:"sensor_ratings"                 json:"sensor_ratings"`
	ShipThermalSigs             map[string]float64 `yaml:"ship_thermal_signatures"        json:"ship_thermal_signatures"`
	InfoQuality                 InfoQualityConfig  `yaml:"info_quality"                   json:"info_quality"`
	SurveyDurationTicks         int                `yaml:"survey_duration_ticks"          json:"survey_duration_ticks"`
	LastKnownPositionDecayTicks int                `yaml:"last_known_position_decay_ticks" json:"last_known_position_decay_ticks"`
}

type InfoQualityConfig struct {
	FullDetailThreshold   float64 `yaml:"full_detail_threshold"   json:"full_detail_threshold"`
	MediumDetailThreshold float64 `yaml:"medium_detail_threshold" json:"medium_detail_threshold"`
	LowDetailThreshold    float64 `yaml:"low_detail_threshold"    json:"low_detail_threshold"`
}

type TimeConfig struct {
	StrategyTickMinutes    int `yaml:"strategy_tick_minutes"     json:"strategy_tick_minutes"`
	CombatTickSeconds      int `yaml:"combat_tick_seconds"       json:"combat_tick_seconds"`
	CombatOptInWindowHours int `yaml:"combat_opt_in_window_hours" json:"combat_opt_in_window_hours"`
	MaxActionQueueDepth    int `yaml:"max_action_queue_depth"    json:"max_action_queue_depth"`
}

type EconomyConfig struct {
	DetailModeEfficiencyBonus      float64 `yaml:"detail_mode_efficiency_bonus"       json:"detail_mode_efficiency_bonus"`
	DetailModeUpgradeDowntimeTicks int     `yaml:"detail_mode_upgrade_downtime_ticks" json:"detail_mode_upgrade_downtime_ticks"`
	DetailModeBreakEvenTicks       int     `yaml:"detail_mode_break_even_ticks"       json:"detail_mode_break_even_ticks"`
	BasePopulationGrowthRate       float64 `yaml:"base_population_growth_rate"        json:"base_population_growth_rate"`
	TaxRateBase                    float64 `yaml:"tax_rate_base"                      json:"tax_rate_base"`
	PlanetSurfaceCostExponent      float64 `yaml:"planet_surface_cost_exponent"       json:"planet_surface_cost_exponent"`
	AsteroidYieldMultiplier        float64 `yaml:"asteroid_yield_multiplier"          json:"asteroid_yield_multiplier"`
}

type PlanetGenConfig struct {
	BiochemArchetypesFile        string             `yaml:"biochemistry_archetypes_file"          json:"biochemistry_archetypes_file"`
	FrostLineConstantAU          float64            `yaml:"frost_line_constant_au"                json:"frost_line_constant_au"`
	GreenhouseOverlapCorrection  float64            `yaml:"greenhouse_overlap_correction"         json:"greenhouse_overlap_correction"`
	SO2AerosolThresholdWaterAct  float64            `yaml:"so2_aerosol_threshold_water_activity"  json:"so2_aerosol_threshold_water_activity"`
	PlanetCountLambda            map[string]float64 `yaml:"planet_count_lambda"                   json:"planet_count_lambda"`
	MoonCollisionProbability     float64            `yaml:"moon_collision_probability"            json:"moon_collision_probability"`
	GasGiantMoonCountMin         int                `yaml:"gas_giant_moon_count_min"              json:"gas_giant_moon_count_min"`
	GasGiantMoonCountMax         int                `yaml:"gas_giant_moon_count_max"              json:"gas_giant_moon_count_max"`
	UsableSurfaceBase            float64            `yaml:"usable_surface_base"                   json:"usable_surface_base"`
	UsableSurfaceHostileBase     float64            `yaml:"usable_surface_hostile_base"           json:"usable_surface_hostile_base"`
}

type ResearchConfig struct {
	BaseResearchSpeed      float64 `yaml:"base_research_speed"       json:"base_research_speed"`
	ScientistResearchBonus float64 `yaml:"scientist_research_bonus"  json:"scientist_research_bonus"`
	ScientistRiskReduction float64 `yaml:"scientist_risk_reduction"  json:"scientist_risk_reduction"`
	LabBonusFactor         float64 `yaml:"lab_bonus_factor"          json:"lab_bonus_factor"`
	ParallelResearchSlots  int     `yaml:"parallel_research_slots"   json:"parallel_research_slots"`
}

type CombatConfig struct {
	RailgunBaseVelocityKmS      float64 `yaml:"railgun_base_velocity_km_s"      json:"railgun_base_velocity_km_s"`
	GraserAntimateriePerShot    float64 `yaml:"graser_antimaterie_cost_per_shot" json:"graser_antimaterie_cost_per_shot"`
	SandcasterInterceptRadiusKm float64 `yaml:"sandcaster_intercept_radius_km"  json:"sandcaster_intercept_radius_km"`
	CombatArenaRadiusKm         float64 `yaml:"combat_arena_radius_km"          json:"combat_arena_radius_km"`
}

type ServerConfig struct {
	MaxPlayers    int    `yaml:"max_players"    json:"max_players"`
	MaxAIFactions int    `yaml:"max_ai_factions" json:"max_ai_factions"`
	InstanceName  string `yaml:"instance_name"  json:"instance_name"`
}

// Load reads the YAML file at path and returns a validated Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}

	cfg.ConfigDir = filepath.Dir(path)

	// Override with environment variables if set
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("REDIS_URL"); v != "" {
		cfg.RedisURL = v
	}

	return &cfg, nil
}

// Validate checks that all required config fields are within acceptable ranges.
func (c *Config) Validate() error {
	if c.Galaxy.NumStars <= 0 {
		return fmt.Errorf("galaxy.num_stars must be > 0")
	}
	if c.Galaxy.RadiusLY <= 0 {
		return fmt.Errorf("galaxy.radius_ly must be > 0")
	}
	if c.FTLW.VoxelSizeLY <= 0 {
		return fmt.Errorf("ftlw.voxel_size_ly must be > 0")
	}
	if c.Time.StrategyTickMinutes <= 0 {
		return fmt.Errorf("time.strategy_tick_minutes must be > 0")
	}
	return nil
}
