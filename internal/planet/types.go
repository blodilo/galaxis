// Package planet contains the planet system generation pipeline (AP2).
// Physikalisches Modell: Frostgrenze (Hayashi 1981), Poisson-Planetenanzahl,
// Stefan-Boltzmann-Gleichgewichtstemperatur, Treibhauseffekt,
// Biochemie-Archetypen (biochemistry_archetypes_v1.0.yaml).
package planet

// GreenhouseGas holds IR forcing parameters per unit partial pressure.
// Fields mirror biochemistry_archetypes_v1.0.yaml.
type GreenhouseGas struct {
	DeltaKPerAtm          float64 `yaml:"delta_k_per_atm"`
	AerosolCoolingKPerAtm float64 `yaml:"aerosol_cooling_k_per_atm"` // SO2 only; negative
}

// Archetype describes one biochemistry template for alien life.
type Archetype struct {
	Enabled                  bool                     `yaml:"enabled"`
	LabelShort               string                   `yaml:"label_short"`
	TempRangeK               [2]float64               `yaml:"temp_range_k"`
	PressureRangeAtm         [2]float64               `yaml:"pressure_range_atm"`
	GravityRangeG            [2]float64               `yaml:"gravity_range_g"`
	CanonicalComposition     map[string]float64       `yaml:"canonical_composition"`
	CompositionJitterFraction float64                 `yaml:"composition_jitter_fraction"`
	GreenhouseGases          map[string]GreenhouseGas `yaml:"greenhouse_gases"`
	BiomassPotentialMax      float64                  `yaml:"biomass_potential_max"`
}

// Balancing holds global balancing parameters from the biochem YAML.
type Balancing struct {
	GreenhouseOverlapCorrection float64            `yaml:"greenhouse_overlap_correction"`
	TargetFraction              map[string]float64 `yaml:"target_fraction"`
	BalanceToleranceFraction    float64            `yaml:"balance_tolerance_fraction"`
}

// BiochemConfig is the parsed biochemistry_archetypes_v*.yaml.
type BiochemConfig struct {
	Archetypes map[string]*Archetype
	SortedIDs  []string // sorted for deterministic CDF
	Balancing  Balancing
}
