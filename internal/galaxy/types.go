// Package galaxy contains the galaxy generation pipeline.
// Domain types live in internal/model to avoid import cycles.
package galaxy

import "galaxis/internal/model"

// Type aliases so existing code inside this package stays readable.
type (
	StarType  = model.StarType
	NebulaType = model.NebulaType
	Star      = model.Star
	Nebula    = model.Nebula
)

const (
	StarTypeO         = model.StarTypeO
	StarTypeB         = model.StarTypeB
	StarTypeA         = model.StarTypeA
	StarTypeF         = model.StarTypeF
	StarTypeG         = model.StarTypeG
	StarTypeK         = model.StarTypeK
	StarTypeM         = model.StarTypeM
	StarTypeWR        = model.StarTypeWR
	StarTypeRStar     = model.StarTypeRStar
	StarTypeSStar     = model.StarTypeSStar
	StarTypePulsar    = model.StarTypePulsar
	StarTypeStellarBH = model.StarTypeStellarBH
	StarTypeSMBH      = model.StarTypeSMBH

	NebulaHII      = model.NebulaHII
	NebulaSNR      = model.NebulaSNR
	NebulaGlobular = model.NebulaGlobular
)
