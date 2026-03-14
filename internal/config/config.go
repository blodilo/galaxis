// Package config loads and validates the central game-params YAML file.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level structure mirroring game-params_v1.0.yaml.
type Config struct {
	Galaxy        GalaxyConfig        `yaml:"galaxy"`
	FTLW          FTLWConfig          `yaml:"ftlw"`
	Sensors       SensorsConfig       `yaml:"sensors"`
	Time          TimeConfig          `yaml:"time"`
	Economy       EconomyConfig       `yaml:"economy"`
	PlanetGen     PlanetGenConfig     `yaml:"planet_generation"`
	Research      ResearchConfig      `yaml:"research"`
	Combat        CombatConfig        `yaml:"combat"`
	Server        ServerConfig        `yaml:"server"`

	// Runtime fields (not from YAML, set via env/flags)
	DatabaseURL string `yaml:"-"`
	RedisURL    string `yaml:"-"`
}

type GalaxyConfig struct {
	Seed          int64   `yaml:"seed"`
	NumStars      int     `yaml:"num_stars"`
	RadiusLY      float64 `yaml:"radius_ly"`
	Type          string  `yaml:"type"`
	Arms          int     `yaml:"arms"`
	ArmWinding    float64 `yaml:"arm_winding"`
	ArmSpread     float64 `yaml:"arm_spread"`
	SMBHMassSolar float64 `yaml:"smbh_mass_solar"`
}

type FTLWConfig struct {
	VacuumBase        float64 `yaml:"vacuum_base"`
	KFactor           float64 `yaml:"k_factor"`
	CutoffPercent     float64 `yaml:"cutoff_percent"`
	VoxelSizeLY       float64 `yaml:"voxel_size_ly"`
	CoarseVoxelSizeLY float64 `yaml:"coarse_voxel_size_ly"`
	PulsarMultiplier  float64 `yaml:"pulsar_multiplier"`
	BlackHoleMultiplier float64 `yaml:"black_hole_multiplier"`
}

type SensorsConfig struct {
	OpticalK         float64            `yaml:"optical_k"`
	FTLK             float64            `yaml:"ftl_k"`
	ShipThermalK     float64            `yaml:"ship_thermal_k"`
	SensorRatings    map[string]float64 `yaml:"sensor_ratings"`
	ShipThermalSigs  map[string]float64 `yaml:"ship_thermal_signatures"`
	InfoQuality      InfoQualityConfig  `yaml:"info_quality"`
	SurveyDurationTicks          int    `yaml:"survey_duration_ticks"`
	LastKnownPositionDecayTicks  int    `yaml:"last_known_position_decay_ticks"`
}

type InfoQualityConfig struct {
	FullDetailThreshold   float64 `yaml:"full_detail_threshold"`
	MediumDetailThreshold float64 `yaml:"medium_detail_threshold"`
	LowDetailThreshold    float64 `yaml:"low_detail_threshold"`
}

type TimeConfig struct {
	StrategyTickMinutes    int `yaml:"strategy_tick_minutes"`
	CombatTickSeconds      int `yaml:"combat_tick_seconds"`
	CombatOptInWindowHours int `yaml:"combat_opt_in_window_hours"`
	MaxActionQueueDepth    int `yaml:"max_action_queue_depth"`
}

type EconomyConfig struct {
	DetailModeEfficiencyBonus      float64 `yaml:"detail_mode_efficiency_bonus"`
	DetailModeUpgradeDowntimeTicks int     `yaml:"detail_mode_upgrade_downtime_ticks"`
	DetailModeBreakEvenTicks       int     `yaml:"detail_mode_break_even_ticks"`
	BasePopulationGrowthRate       float64 `yaml:"base_population_growth_rate"`
	TaxRateBase                    float64 `yaml:"tax_rate_base"`
	PlanetSurfaceCostExponent      float64 `yaml:"planet_surface_cost_exponent"`
	AsteroidYieldMultiplier        float64 `yaml:"asteroid_yield_multiplier"`
}

type PlanetGenConfig struct {
	FrostLineConstantAU       float64            `yaml:"frost_line_constant_au"`
	AtmosphereTypeWeights     map[string]float64 `yaml:"atmosphere_type_weights"`
	MoonCollisionProbability  float64            `yaml:"moon_collision_probability"`
	GasGiantMoonCountMin      int                `yaml:"gas_giant_moon_count_min"`
	GasGiantMoonCountMax      int                `yaml:"gas_giant_moon_count_max"`
	MaxPlanetsPerSystem       int                `yaml:"max_planets_per_system"`
	UsableSurfaceTerranBase   float64            `yaml:"usable_surface_terran_base"`
	UsableSurfaceHostileBase  float64            `yaml:"usable_surface_hostile_base"`
}

type ResearchConfig struct {
	BaseResearchSpeed       float64 `yaml:"base_research_speed"`
	ScientistResearchBonus  float64 `yaml:"scientist_research_bonus"`
	ScientistRiskReduction  float64 `yaml:"scientist_risk_reduction"`
	LabBonusFactor          float64 `yaml:"lab_bonus_factor"`
	ParallelResearchSlots   int     `yaml:"parallel_research_slots"`
}

type CombatConfig struct {
	RailgunBaseVelocityKmS       float64 `yaml:"railgun_base_velocity_km_s"`
	GraserAntimateriePerShot     float64 `yaml:"graser_antimaterie_cost_per_shot"`
	SandcasterInterceptRadiusKm  float64 `yaml:"sandcaster_intercept_radius_km"`
	CombatArenaRadiusKm          float64 `yaml:"combat_arena_radius_km"`
}

type ServerConfig struct {
	MaxPlayers    int    `yaml:"max_players"`
	MaxAIFactions int    `yaml:"max_ai_factions"`
	InstanceName  string `yaml:"instance_name"`
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

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}

	// Override with environment variables if set
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("REDIS_URL"); v != "" {
		cfg.RedisURL = v
	}

	return &cfg, nil
}

func (c *Config) validate() error {
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
