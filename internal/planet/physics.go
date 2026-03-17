package planet

import (
	"math"
	"sort"
)

// FrostLineAU computes the snow line distance (AU) using Hayashi (1981):
//
//	d_frost = sqrt(L / L☉) * frost_line_constant_au
//
// Returns 0 for dark stars (luminosity <= 0).
func FrostLineAU(luminositySolar, frostLineConstantAU float64) float64 {
	if luminositySolar <= 0 {
		return 0
	}
	return math.Sqrt(luminositySolar) * frostLineConstantAU
}

// EquilibriumTempK computes the planetary equilibrium temperature using the
// simplified Stefan-Boltzmann formula:
//
//	T_eq = 278.5 · L^0.25 · (1−albedo)^0.25 / sqrt(d_AU)
//
// Source: standard radiative equilibrium, see Pierrehumbert 2010.
// Returns 3 K (cosmic background) for dark or central stars.
func EquilibriumTempK(luminositySolar, distanceAU, albedo float64) float64 {
	if luminositySolar <= 0 || distanceAU <= 0 {
		return 3.0
	}
	albedoFactor := 1.0 - albedo
	if albedoFactor < 0 {
		albedoFactor = 0
	}
	return 278.5 * math.Pow(luminositySolar, 0.25) *
		math.Pow(albedoFactor, 0.25) / math.Sqrt(distanceAU)
}

// GreenhouseDeltaK computes the surface temperature increase from greenhouse gases.
// Implements special cases from biochemistry_archetypes_v1.0.yaml:
//   - H2 CIA: non-linear, saturates at ~3 bar (Pierrehumbert & Gaidos 2011)
//   - CH4 haze: effect cancelled when CH4 > 2 vol% (Pavlov et al. 2001)
//   - SO2: aerosol cooling when T_eq < 500 K and H2O > threshold (Bullock & Grinspoon 2001)
//
// overlapCorrection: spectral overlap factor (default 0.75, Pierrehumbert 2010).
func GreenhouseDeltaK(
	composition map[string]float64,
	pressure float64,
	archetype *Archetype,
	overlapCorrection float64,
	so2AerosolThreshold float64,
	tEq float64,
) float64 {
	if len(composition) == 0 || pressure <= 0 || archetype == nil {
		return 0
	}

	xH2O := composition["H2O"]
	xCH4 := composition["CH4"]
	xSO2 := composition["SO2"]
	_ = xSO2 // handled via archetype.GreenhouseGases["SO2"]

	// Sorted iteration ensures deterministic results regardless of map order.
	gasKeys := make([]string, 0, len(archetype.GreenhouseGases))
	for g := range archetype.GreenhouseGases {
		gasKeys = append(gasKeys, g)
	}
	sort.Strings(gasKeys)

	total := 0.0
	for _, gas := range gasKeys {
		gg := archetype.GreenhouseGases[gas]
		frac, ok := composition[gas]
		if !ok || frac <= 0 {
			continue
		}
		pp := frac * pressure // partial pressure in atm

		var delta float64
		switch gas {
		case "H2":
			// Collision-Induced Absorption: non-linear, saturates.
			// Source: Pierrehumbert & Gaidos 2011, ApJL 734(1) L13.
			delta = math.Min(pp*gg.DeltaKPerAtm, 80.0)
		case "CH4":
			// Tholin haze: anti-greenhouse when CH4 > 2%.
			// Source: Pavlov et al. 2001, JGR 106(E10).
			if xCH4 > 0.02 {
				delta = 0
			} else {
				delta = pp * gg.DeltaKPerAtm
			}
		case "SO2":
			// Bifurcation: greenhouse (hot/dry) vs. H2SO4 aerosol (cool/wet).
			// Source: Bullock & Grinspoon 2001, Icarus 150(1).
			if tEq < 500 && xH2O > so2AerosolThreshold && gg.AerosolCoolingKPerAtm != 0 {
				delta = pp * gg.AerosolCoolingKPerAtm // negative (cooling)
			} else {
				delta = pp * gg.DeltaKPerAtm
			}
		default:
			delta = pp * gg.DeltaKPerAtm
		}
		total += delta
	}

	// Spectral overlap correction (Pierrehumbert 2010, Kap. 4).
	return total * overlapCorrection
}

// BiomassPotential computes how well a planet (tempK, pressureAtm, gravityG) fits
// the given archetype. Returns 0 if outside habitable range; otherwise scales
// smoothly to the archetype's biomass_potential_max.
func BiomassPotential(a *Archetype, tempK, pressureAtm, gravityG float64) float64 {
	if tempK < a.TempRangeK[0] || tempK > a.TempRangeK[1] {
		return 0
	}
	if pressureAtm < a.PressureRangeAtm[0] || pressureAtm > a.PressureRangeAtm[1] {
		return 0
	}
	if gravityG < a.GravityRangeG[0] || gravityG > a.GravityRangeG[1] {
		return 0
	}

	tMid := (a.TempRangeK[0] + a.TempRangeK[1]) / 2
	tHalf := (a.TempRangeK[1] - a.TempRangeK[0]) / 2
	pMid := (a.PressureRangeAtm[0] + a.PressureRangeAtm[1]) / 2
	pHalf := (a.PressureRangeAtm[1] - a.PressureRangeAtm[0]) / 2
	gMid := (a.GravityRangeG[0] + a.GravityRangeG[1]) / 2
	gHalf := (a.GravityRangeG[1] - a.GravityRangeG[0]) / 2

	scoreT := math.Max(1.0-math.Abs(tempK-tMid)/tHalf, 0)
	scoreP := math.Max(1.0-math.Abs(pressureAtm-pMid)/pHalf, 0)
	scoreG := math.Max(1.0-math.Abs(gravityG-gMid)/gHalf, 0)

	return a.BiomassPotentialMax * scoreT * scoreP * scoreG
}
