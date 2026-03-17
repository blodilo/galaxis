// Package model contains shared domain types used by both the galaxy and db packages.
package model

import "github.com/google/uuid"

// StarType enumerates all possible star types.
type StarType string

const (
	StarTypeO         StarType = "O"
	StarTypeB         StarType = "B"
	StarTypeA         StarType = "A"
	StarTypeF         StarType = "F"
	StarTypeG         StarType = "G"
	StarTypeK         StarType = "K"
	StarTypeM         StarType = "M"
	StarTypeWR        StarType = "WR"
	StarTypeRStar     StarType = "RStar"
	StarTypeSStar     StarType = "SStar"
	StarTypePulsar    StarType = "Pulsar"
	StarTypeStellarBH StarType = "StellarBH"
	StarTypeSMBH      StarType = "SMBH"
)

// NebulaType enumerates nebula types.
type NebulaType string

const (
	NebulaHII      NebulaType = "HII"
	NebulaSNR      NebulaType = "SNR"
	NebulaGlobular NebulaType = "Globular"
)

// Star represents a single star in the galaxy.
type Star struct {
	ID              uuid.UUID
	GalaxyID        uuid.UUID
	NebulaID        *uuid.UUID
	X, Y, Z         float64
	Type            StarType
	SpectralClass   string
	MassSolar       float64
	LuminositySolar float64
	RadiusSolar     float64
	TemperatureK    float64
	ColorHex        string
	PlanetSeed      int64
}

// Nebula represents a nebula region in the galaxy.
type Nebula struct {
	ID                         uuid.UUID
	GalaxyID                   uuid.UUID
	Type                       NebulaType
	CenterX, CenterY, CenterZ float64
	RadiusLY                   float64
	Density                    float64
}

// FTLWChunk is a single compressed FTLW voxel chunk for DB storage.
type FTLWChunk struct {
	CX, CY, CZ int
	Data       []byte
}

// ── API response types ─────────────────────────────────────────────────────────

// StarRow is a lightweight star record for list endpoints.
type StarRow struct {
	ID               string  `json:"id"`
	X                float64 `json:"x"`
	Y                float64 `json:"y"`
	Z                float64 `json:"z"`
	StarType         string  `json:"star_type"`
	SpectralClass    string  `json:"spectral_class,omitempty"`
	MassSolar        float64 `json:"mass_solar,omitempty"`
	LuminositySolar  float64 `json:"luminosity_solar,omitempty"`
	RadiusSolar      float64 `json:"radius_solar,omitempty"`
	TemperatureK     float64 `json:"temperature_k,omitempty"`
	ColorHex         string  `json:"color_hex"`
	NebulaID         *string `json:"nebula_id,omitempty"`
	PlanetsGenerated bool    `json:"planets_generated"`
}

// NebulaRow is a nebula record for API responses.
type NebulaRow struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	CenterX  float64 `json:"center_x"`
	CenterY  float64 `json:"center_y"`
	CenterZ  float64 `json:"center_z"`
	RadiusLY float64 `json:"radius_ly"`
	Density  float64 `json:"density"`
}

// GalaxyRow is used for the galaxy list endpoint.
type GalaxyRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Seed      int64  `json:"seed"`
	Status    string `json:"status"`
	StarCount int    `json:"star_count"`
}

// ── Planet types ───────────────────────────────────────────────────────────────

// Planet represents a generated planet in a star system.
type Planet struct {
	ID                    uuid.UUID
	StarID                uuid.UUID
	OrbitIndex            int
	PlanetType            string // rocky | gas_giant | ice_giant | asteroid_belt
	OrbitDistanceAU       float64
	MassEarth             float64
	RadiusEarth           float64
	SurfaceGravityG       float64
	AtmPressureAtm        float64
	AtmComposition        map[string]float64 // gas → volume fraction
	GreenhouseDeltaK      float64
	SurfaceTempK          float64
	Albedo                float64
	AxialTiltDeg          float64
	RotationPeriodH       float64
	HasRings              bool
	BiochemArchetype      string             // dominant archetype ID; "" = uninhabitable
	BiomassPotential      map[string]float64 // archetype_id → 0.0–1.0
	UsableSurfaceFraction float64
	ResourceDeposits      map[string]float64 // resource_id → amount 0.0–1.0
}

// Moon represents a moon orbiting a planet.
type Moon struct {
	ID               uuid.UUID
	PlanetID         uuid.UUID
	OrbitIndex       int
	MassEarth        float64
	RadiusEarth      float64
	CompositionType  string // rocky | icy | mixed
	SurfaceTempK     float64
	ResourceDeposits map[string]float64
}

// PlanetRow is a planet record for API responses.
type PlanetRow struct {
	ID                    string             `json:"id"`
	OrbitIndex            int                `json:"orbit_index"`
	PlanetType            string             `json:"planet_type"`
	OrbitDistanceAU       float64            `json:"orbit_distance_au"`
	MassEarth             float64            `json:"mass_earth"`
	RadiusEarth           float64            `json:"radius_earth"`
	SurfaceGravityG       float64            `json:"surface_gravity_g"`
	AtmPressureAtm        float64            `json:"atm_pressure_atm"`
	AtmComposition        map[string]float64 `json:"atm_composition"`
	GreenhouseDeltaK      float64            `json:"greenhouse_delta_k"`
	SurfaceTempK          float64            `json:"surface_temp_k"`
	Albedo                float64            `json:"albedo"`
	AxialTiltDeg          float64            `json:"axial_tilt_deg"`
	RotationPeriodH       float64            `json:"rotation_period_h"`
	HasRings              bool               `json:"has_rings"`
	BiochemArchetype      string             `json:"biochem_archetype"`
	BiomassPotential      map[string]float64 `json:"biomass_potential"`
	UsableSurfaceFraction float64            `json:"usable_surface_fraction"`
	ResourceDeposits      map[string]float64 `json:"resource_deposits"`
	Moons                 []MoonRow          `json:"moons"`
}

// MoonRow is a moon record for API responses.
type MoonRow struct {
	ID               string             `json:"id"`
	OrbitIndex       int                `json:"orbit_index"`
	MassEarth        float64            `json:"mass_earth"`
	RadiusEarth      float64            `json:"radius_earth"`
	CompositionType  string             `json:"composition_type"`
	SurfaceTempK     float64            `json:"surface_temp_k"`
	ResourceDeposits map[string]float64 `json:"resource_deposits"`
}
