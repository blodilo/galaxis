package galaxy

import (
	"math"
	"math/rand/v2"
)

// spectralRange defines mass range and visual properties for a spectral class.
type spectralRange struct {
	MassMin, MassMax float64
	TempMin, TempMax float64
	ColorHex         string
	SpectralClass    string
}

var mainSequenceClasses = map[StarType]spectralRange{
	StarTypeO: {16, 120, 30000, 50000, "#9bb0ff", "O"},
	StarTypeB: {2.1, 16, 10000, 30000, "#aabfff", "B"},
	StarTypeA: {1.4, 2.1, 7500, 10000, "#cad7ff", "A"},
	StarTypeF: {1.04, 1.4, 6000, 7500, "#f8f7ff", "F"},
	StarTypeG: {0.8, 1.04, 5200, 6000, "#fff4ea", "G"},
	StarTypeK: {0.45, 0.8, 3700, 5200, "#ffd2a1", "K"},
	StarTypeM: {0.08, 0.45, 2400, 3700, "#ffb06a", "M"},
}

// nebulaWeights defines draw weights for each star type per nebula context.
// "" = free disk (no nebula).
var nebulaWeights = map[NebulaType]map[StarType]float64{
	NebulaHII: {
		StarTypeO: 8, StarTypeB: 6, StarTypeA: 3, StarTypeF: 2,
		StarTypeG: 0.5, StarTypeK: 0.3, StarTypeM: 0.2,
		StarTypeWR: 2,
	},
	NebulaSNR: {
		StarTypeA: 0.2, StarTypeF: 0.5, StarTypeG: 1, StarTypeK: 2, StarTypeM: 2,
		StarTypePulsar: 4, StarTypeStellarBH: 1,
	},
	NebulaGlobular: {
		StarTypeG: 0.5, StarTypeK: 2, StarTypeM: 5,
		StarTypePulsar: 0.2, StarTypeStellarBH: 0.2,
		StarTypeRStar: 2, StarTypeSStar: 2,
	},
	"": {
		StarTypeB: 0.3, StarTypeA: 1, StarTypeF: 3,
		StarTypeG: 4, StarTypeK: 6, StarTypeM: 12,
		StarTypeRStar: 0.2, StarTypeSStar: 0.1,
	},
}

// drawStarType samples a star type from the weighted distribution for the nebula context.
func drawStarType(rng *rand.Rand, nebulaType NebulaType) StarType {
	weights, ok := nebulaWeights[nebulaType]
	if !ok {
		weights = nebulaWeights[""]
	}

	types := make([]StarType, 0, len(weights))
	cumulative := make([]float64, 0, len(weights))
	total := 0.0
	for t, w := range weights {
		if w > 0 {
			types = append(types, t)
			total += w
			cumulative = append(cumulative, total)
		}
	}

	r := rng.Float64() * total
	for i, c := range cumulative {
		if r <= c {
			return types[i]
		}
	}
	return types[len(types)-1]
}

type starProps struct {
	Mass, Luminosity, Radius, Temperature float64
	ColorHex, SpectralClass               string
}

// buildStarProps derives physical properties for a star of the given type.
func buildStarProps(rng *rand.Rand, t StarType, smbhMass float64) starProps {
	noise := func(base, frac float64) float64 {
		return base * (1 + frac*(rng.Float64()*2-1))
	}
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }

	switch t {
	case StarTypeO, StarTypeB, StarTypeA, StarTypeF, StarTypeG, StarTypeK, StarTypeM:
		r := mainSequenceClasses[t]
		m := lerp(r.MassMin, r.MassMax, rng.Float64())
		m = noise(m, 0.05)
		var l float64
		if m > 0.43 {
			l = noise(math.Pow(m, 3.5), 0.10)
		} else {
			l = noise(0.23*math.Pow(m, 2.3), 0.10)
		}
		var rad float64
		if m > 1 {
			rad = noise(math.Pow(m, 0.8), 0.08)
		} else {
			rad = noise(math.Pow(m, 0.5), 0.08)
		}
		temp := noise(lerp(r.TempMin, r.TempMax, rng.Float64()), 0.03)
		return starProps{m, l, rad, temp, r.ColorHex, r.SpectralClass}

	case StarTypeWR:
		m := lerp(10, 200, rng.Float64())
		return starProps{
			Mass:          m,
			Luminosity:    noise(math.Pow(m, 3.5)*1.5, 0.15),
			Radius:        noise(math.Pow(m, 0.6), 0.10),
			Temperature:   lerp(25000, 100000, rng.Float64()),
			ColorHex:      "#00e5ff",
			SpectralClass: "WR",
		}

	case StarTypeRStar:
		m := lerp(1, 3, rng.Float64())
		return starProps{
			Mass:          m,
			Luminosity:    noise(math.Pow(m, 2.5)*800, 0.20),
			Radius:        lerp(100, 500, rng.Float64()),
			Temperature:   lerp(3000, 4000, rng.Float64()),
			ColorHex:      "#ff4400",
			SpectralClass: "R",
		}

	case StarTypeSStar:
		m := lerp(1, 4, rng.Float64())
		return starProps{
			Mass:          m,
			Luminosity:    noise(math.Pow(m, 2.5)*600, 0.20),
			Radius:        lerp(80, 400, rng.Float64()),
			Temperature:   lerp(3000, 4200, rng.Float64()),
			ColorHex:      "#ff7722",
			SpectralClass: "S",
		}

	case StarTypePulsar:
		m := lerp(1.4, 2.1, rng.Float64())
		return starProps{
			Mass:          m,
			Luminosity:    noise(0.001, 0.50),
			Radius:        10.0 / 695700.0, // 10 km in solar radii
			Temperature:   lerp(100000, 3000000, rng.Float64()),
			ColorHex:      "#e0e8ff",
			SpectralClass: "NS",
		}

	case StarTypeStellarBH:
		m := lerp(5, 100, rng.Float64())
		rs := m * 2953.0 / 695700000.0 // Schwarzschild radius in solar radii
		return starProps{
			Mass:          m,
			Luminosity:    0,
			Radius:        rs,
			Temperature:   0,
			ColorHex:      "#1a0500",
			SpectralClass: "BH",
		}

	case StarTypeSMBH:
		m := smbhMass
		rs := m * 2953.0 / 695700000.0
		return starProps{
			Mass:          m,
			Luminosity:    noise(1e12, 0.50),
			Radius:        rs,
			Temperature:   0,
			ColorHex:      "#ff6600",
			SpectralClass: "SMBH",
		}
	}

	// fallback (should not happen)
	return starProps{Mass: 1, Luminosity: 1, Radius: 1, Temperature: 5800, ColorHex: "#ffffff"}
}
