package planet

import (
	"math"
	"testing"
)

// ── FrostLineAU ────────────────────────────────────────────────────────────────

func TestFrostLineAU_Solar(t *testing.T) {
	// L=1 L☉, standard constant 2.7 AU → frost line = 2.7 AU (Hayashi 1981)
	got := FrostLineAU(1.0, 2.7)
	if math.Abs(got-2.7) > 0.001 {
		t.Errorf("FrostLineAU(1.0, 2.7) = %.4f, want 2.7", got)
	}
}

func TestFrostLineAU_HighLuminosity(t *testing.T) {
	// L=4 → sqrt(4)=2 → frost line doubles to 5.4 AU
	got := FrostLineAU(4.0, 2.7)
	if math.Abs(got-5.4) > 0.001 {
		t.Errorf("FrostLineAU(4.0, 2.7) = %.4f, want 5.4", got)
	}
}

func TestFrostLineAU_DarkStar(t *testing.T) {
	// Black hole / no luminosity → 0
	if got := FrostLineAU(0, 2.7); got != 0 {
		t.Errorf("FrostLineAU(0, 2.7) = %.4f, want 0", got)
	}
	if got := FrostLineAU(-1, 2.7); got != 0 {
		t.Errorf("FrostLineAU(-1, 2.7) = %.4f, want 0", got)
	}
}

func TestFrostLineAU_Monotone(t *testing.T) {
	// More luminous stars have larger frost lines
	k := 2.7
	prev := FrostLineAU(0.01, k)
	for _, l := range []float64{0.1, 1.0, 10.0, 100.0} {
		curr := FrostLineAU(l, k)
		if curr <= prev {
			t.Errorf("FrostLineAU not monotone: L=%.2f gives %.4f, L<%.2f gives %.4f", l, curr, l, prev)
		}
		prev = curr
	}
}

// ── EquilibriumTempK ───────────────────────────────────────────────────────────

func TestEquilibriumTempK_EarthLike(t *testing.T) {
	// Sun-like star (L=1), 1 AU, albedo=0.3 → ~255 K (Earth black-body temperature)
	// Formula: 278.5 * 1^0.25 * 0.7^0.25 / sqrt(1) ≈ 254.8 K
	got := EquilibriumTempK(1.0, 1.0, 0.3)
	if got < 250 || got > 260 {
		t.Errorf("EquilibriumTempK(L=1, d=1AU, albedo=0.3) = %.1f K, want 250–260 K (Earth black-body)", got)
	}
}

func TestEquilibriumTempK_NoLuminosity(t *testing.T) {
	// Dark object → cosmic background 3 K
	got := EquilibriumTempK(0, 1.0, 0.3)
	if got != 3.0 {
		t.Errorf("EquilibriumTempK(L=0, ...) = %.1f, want 3.0", got)
	}
	got = EquilibriumTempK(-1, 1.0, 0.3)
	if got != 3.0 {
		t.Errorf("EquilibriumTempK(L=-1, ...) = %.1f, want 3.0", got)
	}
}

func TestEquilibriumTempK_ZeroDistance(t *testing.T) {
	// d=0 → singularity protection → 3 K
	got := EquilibriumTempK(1.0, 0, 0.3)
	if got != 3.0 {
		t.Errorf("EquilibriumTempK(L=1, d=0, ...) = %.1f, want 3.0", got)
	}
}

func TestEquilibriumTempK_InverseSquareRoot(t *testing.T) {
	// T ∝ 1/sqrt(d): planet at 4 AU should be half as warm as planet at 1 AU
	t1 := EquilibriumTempK(1.0, 1.0, 0.0)
	t4 := EquilibriumTempK(1.0, 4.0, 0.0)
	ratio := t1 / t4
	if math.Abs(ratio-2.0) > 0.001 {
		t.Errorf("T(1AU)/T(4AU) = %.4f, want 2.0 (inverse sqrt law)", ratio)
	}
}

func TestEquilibriumTempK_Albedo(t *testing.T) {
	// Higher albedo → cooler planet
	tLow := EquilibriumTempK(1.0, 1.0, 0.0)
	tHigh := EquilibriumTempK(1.0, 1.0, 0.7)
	if tLow <= tHigh {
		t.Errorf("Higher albedo should give lower temp: T(a=0)=%.1f, T(a=0.7)=%.1f", tLow, tHigh)
	}
}

// ── GreenhouseDeltaK ───────────────────────────────────────────────────────────

func TestGreenhouseDeltaK_Empty(t *testing.T) {
	arch := &Archetype{GreenhouseGases: map[string]GreenhouseGas{
		"CO2": {DeltaKPerAtm: 10},
	}}
	// Empty composition
	if got := GreenhouseDeltaK(nil, 1.0, arch, 0.75, 0.1, 300); got != 0 {
		t.Errorf("empty composition → 0, got %.2f", got)
	}
	// Zero pressure
	if got := GreenhouseDeltaK(map[string]float64{"CO2": 0.9}, 0, arch, 0.75, 0.1, 300); got != 0 {
		t.Errorf("zero pressure → 0, got %.2f", got)
	}
	// Nil archetype
	if got := GreenhouseDeltaK(map[string]float64{"CO2": 0.9}, 1.0, nil, 0.75, 0.1, 300); got != 0 {
		t.Errorf("nil archetype → 0, got %.2f", got)
	}
}

func TestGreenhouseDeltaK_LinearCO2(t *testing.T) {
	// CO2 at 0.9 vol, 1 atm, delta=10 K/atm, overlap=1.0 → should give 0.9*10*1.0 = 9 K
	arch := &Archetype{GreenhouseGases: map[string]GreenhouseGas{
		"CO2": {DeltaKPerAtm: 10},
	}}
	got := GreenhouseDeltaK(map[string]float64{"CO2": 0.9}, 1.0, arch, 1.0, 0.1, 300)
	if math.Abs(got-9.0) > 0.001 {
		t.Errorf("CO2 linear: got %.4f, want 9.0", got)
	}
}

func TestGreenhouseDeltaK_H2Saturation(t *testing.T) {
	// H2 CIA saturates at 80 K regardless of high pressure
	arch := &Archetype{GreenhouseGases: map[string]GreenhouseGas{
		"H2": {DeltaKPerAtm: 1000},
	}}
	got := GreenhouseDeltaK(map[string]float64{"H2": 1.0}, 100.0, arch, 1.0, 0.1, 200)
	// 100 atm * 1000 K/atm = 100000 K → clipped to 80
	if got > 80.1 {
		t.Errorf("H2 CIA should saturate at 80 K, got %.2f", got)
	}
}

func TestGreenhouseDeltaK_CH4Haze(t *testing.T) {
	// CH4 > 2% → tholin haze → effect cancelled
	arch := &Archetype{GreenhouseGases: map[string]GreenhouseGas{
		"CH4": {DeltaKPerAtm: 100},
	}}
	highCH4 := map[string]float64{"CH4": 0.05} // 5% > 2% threshold
	got := GreenhouseDeltaK(highCH4, 1.0, arch, 1.0, 0.1, 300)
	if got != 0 {
		t.Errorf("CH4 > 2%% → anti-greenhouse, expected 0, got %.2f", got)
	}

	lowCH4 := map[string]float64{"CH4": 0.01} // 1% < 2% → normal greenhouse
	got2 := GreenhouseDeltaK(lowCH4, 1.0, arch, 1.0, 0.1, 300)
	if got2 <= 0 {
		t.Errorf("CH4 < 2%% → should give positive ΔT, got %.2f", got2)
	}
}

func TestGreenhouseDeltaK_OverlapCorrection(t *testing.T) {
	arch := &Archetype{GreenhouseGases: map[string]GreenhouseGas{
		"CO2": {DeltaKPerAtm: 10},
	}}
	comp := map[string]float64{"CO2": 1.0}
	full := GreenhouseDeltaK(comp, 1.0, arch, 1.0, 0.1, 300)
	corrected := GreenhouseDeltaK(comp, 1.0, arch, 0.75, 0.1, 300)
	if math.Abs(corrected-full*0.75) > 0.001 {
		t.Errorf("overlap correction: full=%.2f, corrected=%.2f, want %.2f", full, corrected, full*0.75)
	}
}

// ── BiomassPotential ───────────────────────────────────────────────────────────

var terranArch = &Archetype{
	TempRangeK:       [2]float64{260, 330},
	PressureRangeAtm: [2]float64{0.5, 3.0},
	GravityRangeG:    [2]float64{0.3, 2.0},
	BiomassPotentialMax: 0.9,
}

func TestBiomassPotential_Center(t *testing.T) {
	// Center of all ranges → maximum output
	tMid := (260.0 + 330.0) / 2      // 295 K
	pMid := (0.5 + 3.0) / 2           // 1.75 atm
	gMid := (0.3 + 2.0) / 2           // 1.15 g
	got := BiomassPotential(terranArch, tMid, pMid, gMid)
	if math.Abs(got-0.9) > 0.001 {
		t.Errorf("center of range: got %.4f, want 0.9 (BiomassPotentialMax)", got)
	}
}

func TestBiomassPotential_OutsideRange(t *testing.T) {
	tests := []struct {
		name string
		t, p, g float64
	}{
		{"too cold", 200, 1.75, 1.15},
		{"too hot", 400, 1.75, 1.15},
		{"too low pressure", 295, 0.1, 1.15},
		{"too high pressure", 295, 5.0, 1.15},
		{"too low gravity", 295, 1.75, 0.1},
		{"too high gravity", 295, 1.75, 3.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BiomassPotential(terranArch, tc.t, tc.p, tc.g)
			if got != 0 {
				t.Errorf("outside range: got %.4f, want 0", got)
			}
		})
	}
}

func TestBiomassPotential_ScalesFromCenter(t *testing.T) {
	// Moving away from center decreases potential
	tMid := (260.0 + 330.0) / 2
	pMid := (0.5 + 3.0) / 2
	gMid := (0.3 + 2.0) / 2

	center := BiomassPotential(terranArch, tMid, pMid, gMid)
	offCenter := BiomassPotential(terranArch, tMid+20, pMid, gMid)
	if offCenter >= center {
		t.Errorf("off-center should be lower: center=%.4f, off=%.4f", center, offCenter)
	}
}
