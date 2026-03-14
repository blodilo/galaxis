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
