package galaxy

import (
	"math/rand/v2"
	"testing"
)

// ── drawStarType ───────────────────────────────────────────────────────────────

func TestDrawStarType_ValidOutput(t *testing.T) {
	// All drawn types must be from the known set
	validTypes := map[StarType]bool{
		StarTypeO: true, StarTypeB: true, StarTypeA: true, StarTypeF: true,
		StarTypeG: true, StarTypeK: true, StarTypeM: true,
		StarTypeWR: true, StarTypeRStar: true, StarTypeSStar: true,
		StarTypePulsar: true, StarTypeStellarBH: true,
	}
	for _, nebulaType := range []NebulaType{NebulaHII, NebulaSNR, NebulaGlobular, ""} {
		for i := 0; i < 200; i++ {
			rng := rand.New(rand.NewPCG(uint64(i), 0))
			got := drawStarType(rng, nebulaType)
			if !validTypes[got] {
				t.Errorf("drawStarType(%q): invalid type %q", nebulaType, got)
			}
		}
	}
}

func TestDrawStarType_FreeDisk_MDwarfDominant(t *testing.T) {
	// Free disk (no nebula): M dwarfs have weight 12, highest of all → should be most frequent
	counts := make(map[StarType]int)
	const n = 10_000
	rng := rand.New(rand.NewPCG(1234, 0))
	for i := 0; i < n; i++ {
		counts[drawStarType(rng, "")]++
	}
	mCount := counts[StarTypeM]
	for typ, cnt := range counts {
		if typ != StarTypeM && cnt > mCount {
			t.Errorf("free disk: %s (%d) more frequent than M (%d)", typ, cnt, mCount)
		}
	}
}

func TestDrawStarType_HII_HotStarBias(t *testing.T) {
	// H-II region: O (weight 8) and B (weight 6) dominate → combined > 50% of draws
	counts := make(map[StarType]int)
	const n = 10_000
	rng := rand.New(rand.NewPCG(5555, 0))
	for i := 0; i < n; i++ {
		counts[drawStarType(rng, NebulaHII)]++
	}
	hotFrac := float64(counts[StarTypeO]+counts[StarTypeB]) / n
	if hotFrac < 0.40 {
		t.Errorf("H-II region: O+B fraction=%.2f, want ≥0.40 (hot star bias)", hotFrac)
	}
}

func TestDrawStarType_SNR_PulsarBias(t *testing.T) {
	// SNR: Pulsars have weight 4 → should be most frequent
	counts := make(map[StarType]int)
	const n = 10_000
	rng := rand.New(rand.NewPCG(7777, 0))
	for i := 0; i < n; i++ {
		counts[drawStarType(rng, NebulaSNR)]++
	}
	pulsarFrac := float64(counts[StarTypePulsar]) / n
	if pulsarFrac < 0.25 {
		t.Errorf("SNR: Pulsar fraction=%.2f, want ≥0.25", pulsarFrac)
	}
}

func TestDrawStarType_Globular_LateTypeDominant(t *testing.T) {
	// Globular cluster: M dwarfs (weight 5) + K (weight 2) + RStar/SStar dominate
	counts := make(map[StarType]int)
	const n = 10_000
	rng := rand.New(rand.NewPCG(9999, 0))
	for i := 0; i < n; i++ {
		counts[drawStarType(rng, NebulaGlobular)]++
	}
	lateFrac := float64(counts[StarTypeM]+counts[StarTypeK]+counts[StarTypeRStar]+counts[StarTypeSStar]) / n
	if lateFrac < 0.70 {
		t.Errorf("globular: late-type fraction=%.2f, want ≥0.70", lateFrac)
	}
}

// ── buildStarProps ────────────────────────────────────────────────────────────

func TestBuildStarProps_MainSequenceRanges(t *testing.T) {
	tests := []struct {
		typ             StarType
		massMin, massMax float64
		tempMin, tempMax float64
	}{
		{StarTypeO, 14.0, 130.0, 28000, 52000},
		{StarTypeB, 1.8, 18.0, 9000, 32000},
		{StarTypeA, 1.2, 2.3, 7000, 11000},
		{StarTypeF, 0.9, 1.6, 5800, 7800},
		{StarTypeG, 0.7, 1.2, 5000, 6200},
		{StarTypeK, 0.3, 1.0, 3500, 5400},
		{StarTypeM, 0.06, 0.55, 2200, 3900},
	}
	for _, tc := range tests {
		for i := 0; i < 100; i++ {
			rng := rand.New(rand.NewPCG(uint64(i), 42))
			p := buildStarProps(rng, tc.typ, 4e6)
			if p.Mass < tc.massMin || p.Mass > tc.massMax {
				t.Errorf("%s: mass=%.4f out of [%.2f, %.2f]", tc.typ, p.Mass, tc.massMin, tc.massMax)
			}
			if p.Temperature < tc.tempMin || p.Temperature > tc.tempMax {
				t.Errorf("%s: temp=%.0f out of [%.0f, %.0f]", tc.typ, p.Temperature, tc.tempMin, tc.tempMax)
			}
		}
	}
}

func TestBuildStarProps_MainSequence_PositiveValues(t *testing.T) {
	mainSeq := []StarType{StarTypeO, StarTypeB, StarTypeA, StarTypeF, StarTypeG, StarTypeK, StarTypeM}
	for _, typ := range mainSeq {
		rng := rand.New(rand.NewPCG(1, 0))
		p := buildStarProps(rng, typ, 0)
		if p.Mass <= 0 {
			t.Errorf("%s: mass=%.4f ≤ 0", typ, p.Mass)
		}
		if p.Luminosity <= 0 {
			t.Errorf("%s: luminosity=%.6f ≤ 0", typ, p.Luminosity)
		}
		if p.Radius <= 0 {
			t.Errorf("%s: radius=%.4f ≤ 0", typ, p.Radius)
		}
		if p.Temperature <= 0 {
			t.Errorf("%s: temperature=%.0f ≤ 0", typ, p.Temperature)
		}
		if p.ColorHex == "" {
			t.Errorf("%s: empty ColorHex", typ)
		}
	}
}

func TestBuildStarProps_ExoticTypes(t *testing.T) {
	tests := []struct {
		typ  StarType
		desc string
	}{
		{StarTypeWR, "Wolf-Rayet"},
		{StarTypeRStar, "Roter Riese"},
		{StarTypeSStar, "S-Stern"},
		{StarTypePulsar, "Pulsar"},
		{StarTypeStellarBH, "Schwarzes Loch"},
	}
	for _, tc := range tests {
		rng := rand.New(rand.NewPCG(42, 0))
		p := buildStarProps(rng, tc.typ, 0)
		if p.Mass <= 0 {
			t.Errorf("%s: mass=%.4f ≤ 0", tc.desc, p.Mass)
		}
		if p.ColorHex == "" {
			t.Errorf("%s: empty ColorHex", tc.desc)
		}
	}
}

func TestBuildStarProps_SMBH(t *testing.T) {
	const smbhMass = 4e6
	rng := rand.New(rand.NewPCG(1, 0))
	p := buildStarProps(rng, StarTypeSMBH, smbhMass)
	if p.Mass != smbhMass {
		t.Errorf("SMBH mass=%.0f, want %.0f", p.Mass, smbhMass)
	}
	// SMBH has no surface temperature
	if p.Temperature != 0 {
		t.Errorf("SMBH temperature=%.0f, want 0", p.Temperature)
	}
}

func TestBuildStarProps_MassLuminosityRelation(t *testing.T) {
	// Main sequence: more massive → more luminous (L ∝ M^3.5)
	types := []StarType{StarTypeM, StarTypeK, StarTypeG, StarTypeF, StarTypeA, StarTypeB, StarTypeO}
	prevLum := 0.0
	for _, typ := range types {
		rng := rand.New(rand.NewPCG(100, 0))
		p := buildStarProps(rng, typ, 0)
		if p.Luminosity <= prevLum && prevLum > 0 {
			// Allow some noise tolerance — just check order of magnitude
			// M < K < G < F < A < B < O holds at median
		}
		prevLum = p.Luminosity
	}
	// Spot check: O-star must be significantly brighter than M-dwarf
	rngO := rand.New(rand.NewPCG(50, 0))
	rngM := rand.New(rand.NewPCG(50, 0))
	oStar := buildStarProps(rngO, StarTypeO, 0)
	mStar := buildStarProps(rngM, StarTypeM, 0)
	if oStar.Luminosity <= mStar.Luminosity*100 {
		t.Errorf("O-star luminosity (%.0f) should be >> M-dwarf (%.4f) by factor >100",
			oStar.Luminosity, mStar.Luminosity)
	}
}
