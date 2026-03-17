package planet

import (
	"math"
	"math/rand/v2"
	"testing"

	"galaxis/internal/config"
	"galaxis/internal/model"

	"github.com/google/uuid"
)

// ── Test helpers ───────────────────────────────────────────────────────────────

// testConfig returns a minimal but valid PlanetGenConfig for tests.
func testPlanetGenConfig() config.PlanetGenConfig {
	return config.PlanetGenConfig{
		BiochemArchetypesFile:       "biochemistry_archetypes_v1.0.yaml",
		FrostLineConstantAU:         2.7,
		GreenhouseOverlapCorrection: 0.75,
		SO2AerosolThresholdWaterAct: 0.10,
		PlanetCountLambda: map[string]float64{
			"O": 1.0, "B": 2.0, "A": 3.0, "F": 4.0,
			"G": 5.0, "K": 4.0, "M": 2.5,
			"WR": 0.5, "RStar": 1.5, "SStar": 1.5,
			"Pulsar": 0.5, "StellarBH": 0.2, "SMBH": 0.0,
		},
		MoonCollisionProbability: 0.30,
		GasGiantMoonCountMin:     2,
		GasGiantMoonCountMax:     6,
		UsableSurfaceBase:        0.60,
		UsableSurfaceHostileBase: 0.05,
	}
}

// newTestGenerator creates a Generator loading biochem from the repo root.
// Tests are run from the package directory, so ../../ is the project root.
func newTestGenerator(t *testing.T) *Generator {
	t.Helper()
	cfg := &config.Config{
		PlanetGen: testPlanetGenConfig(),
		ConfigDir: "../..",
	}
	gen, err := NewGenerator(cfg)
	if err != nil {
		t.Fatalf("NewGenerator: %v", err)
	}
	return gen
}

// testStar returns a G-type star suitable for planetary system generation.
func testStarG() model.Star {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	return model.Star{
		ID:              id,
		GalaxyID:        uuid.MustParse("00000000-0000-0000-0000-000000000099"),
		Type:            model.StarTypeG,
		MassSolar:       1.0,
		LuminositySolar: 1.0,
		RadiusSolar:     1.0,
		TemperatureK:    5778,
		ColorHex:        "#fff4ea",
		PlanetSeed:      42_000_000,
	}
}

// ── Determinism ────────────────────────────────────────────────────────────────

func TestGenerateSystem_Deterministic(t *testing.T) {
	gen := newTestGenerator(t)
	star := testStarG()

	rng1 := rand.New(rand.NewPCG(uint64(star.PlanetSeed), uint64(star.PlanetSeed>>32)))
	rng2 := rand.New(rand.NewPCG(uint64(star.PlanetSeed), uint64(star.PlanetSeed>>32)))

	planets1, moons1 := gen.generateSystem(rng1, star)
	planets2, moons2 := gen.generateSystem(rng2, star)

	if len(planets1) != len(planets2) {
		t.Fatalf("planet count: %d vs %d (non-deterministic)", len(planets1), len(planets2))
	}
	if len(moons1) != len(moons2) {
		t.Fatalf("moon count: %d vs %d (non-deterministic)", len(moons1), len(moons2))
	}
	for i := range planets1 {
		p1, p2 := planets1[i], planets2[i]
		if p1.PlanetType != p2.PlanetType {
			t.Errorf("planet[%d] type mismatch: %s vs %s", i, p1.PlanetType, p2.PlanetType)
		}
		if math.Abs(p1.SurfaceTempK-p2.SurfaceTempK) > 0.001 {
			t.Errorf("planet[%d] surface temp: %.2f vs %.2f", i, p1.SurfaceTempK, p2.SurfaceTempK)
		}
		if math.Abs(p1.OrbitDistanceAU-p2.OrbitDistanceAU) > 1e-9 {
			t.Errorf("planet[%d] orbit: %.6f vs %.6f", i, p1.OrbitDistanceAU, p2.OrbitDistanceAU)
		}
	}
}

func TestGenerateSystem_DifferentSeeds(t *testing.T) {
	gen := newTestGenerator(t)
	star := testStarG()

	rng1 := rand.New(rand.NewPCG(uint64(star.PlanetSeed), uint64(star.PlanetSeed>>32)))
	rng2 := rand.New(rand.NewPCG(uint64(star.PlanetSeed)+1, uint64(star.PlanetSeed>>32)))

	planets1, _ := gen.generateSystem(rng1, star)
	planets2, _ := gen.generateSystem(rng2, star)

	// Two different seeds should produce different systems with very high probability
	same := true
	if len(planets1) != len(planets2) {
		same = false
	} else {
		for i := range planets1 {
			if math.Abs(planets1[i].OrbitDistanceAU-planets2[i].OrbitDistanceAU) > 0.001 {
				same = false
				break
			}
		}
	}
	if same {
		t.Error("different seeds produced identical systems — RNG likely broken")
	}
}

// ── Physical constraints ────────────────────────────────────────────────────────

func TestGenerateSystem_PlanetProperties(t *testing.T) {
	gen := newTestGenerator(t)
	star := testStarG()

	rng := rand.New(rand.NewPCG(uint64(star.PlanetSeed), uint64(star.PlanetSeed>>32)))
	planets, _ := gen.generateSystem(rng, star)

	for i, p := range planets {
		// Surface temperature must be physically plausible
		if p.SurfaceTempK < 2 {
			t.Errorf("planet[%d] temp %.1f K < 2 K (below cosmic background)", i, p.SurfaceTempK)
		}
		if p.SurfaceTempK > 10000 {
			t.Errorf("planet[%d] temp %.1f K > 10000 K (unphysically hot surface)", i, p.SurfaceTempK)
		}
		// Orbit must be positive and within galaxy-generation bounds
		if p.OrbitDistanceAU <= 0 {
			t.Errorf("planet[%d] orbit %.4f AU <= 0", i, p.OrbitDistanceAU)
		}
		// Albedo in [0, 1]
		if p.Albedo < 0 || p.Albedo > 1 {
			t.Errorf("planet[%d] albedo %.4f out of [0,1]", i, p.Albedo)
		}
		// Orbit index must match position in list
		if p.OrbitIndex != i {
			t.Errorf("planet[%d] orbit_index=%d, want %d", i, p.OrbitIndex, i)
		}
		// Non-asteroid mass and radius
		if p.PlanetType != "asteroid_belt" {
			if p.MassEarth <= 0 {
				t.Errorf("planet[%d] (%s) mass=%.4f <= 0", i, p.PlanetType, p.MassEarth)
			}
			if p.RadiusEarth <= 0 {
				t.Errorf("planet[%d] (%s) radius=%.4f <= 0", i, p.PlanetType, p.RadiusEarth)
			}
		}
	}
}

func TestGenerateSystem_OrbitSpacing(t *testing.T) {
	gen := newTestGenerator(t)
	star := testStarG()

	rng := rand.New(rand.NewPCG(uint64(star.PlanetSeed), uint64(star.PlanetSeed>>32)))
	planets, _ := gen.generateSystem(rng, star)

	// Orbits must be strictly increasing (Titius-Bode spacing)
	for i := 1; i < len(planets); i++ {
		if planets[i].OrbitDistanceAU <= planets[i-1].OrbitDistanceAU {
			t.Errorf("orbit[%d]=%.4f AU ≤ orbit[%d]=%.4f AU (not strictly increasing)",
				i, planets[i].OrbitDistanceAU, i-1, planets[i-1].OrbitDistanceAU)
		}
	}
}

// ── drawPlanetType ────────────────────────────────────────────────────────────

func TestDrawPlanetType_Inner(t *testing.T) {
	// Inside frost line → always rocky
	for i := 0; i < 100; i++ {
		rng := rand.New(rand.NewPCG(uint64(i), 0))
		got := drawPlanetType(rng, 1.0, 3.0)
		if got != "rocky" {
			t.Errorf("inside frost line: got %q, want rocky (i=%d)", got, i)
		}
	}
}

func TestDrawPlanetType_NoFrostLine(t *testing.T) {
	// frostLine=0 means not applicable → rocky inside, mixed beyond
	rng := rand.New(rand.NewPCG(1, 0))
	got := drawPlanetType(rng, 10.0, 0)
	validTypes := map[string]bool{"gas_giant": true, "ice_giant": true, "asteroid_belt": true}
	if !validTypes[got] {
		t.Errorf("beyond frost line (frostLine=0): got %q", got)
	}
}

func TestDrawPlanetType_BeyondFrost(t *testing.T) {
	// Far beyond frost line → gas giants / ice giants / asteroid belts
	validOuter := map[string]bool{"gas_giant": true, "ice_giant": true, "asteroid_belt": true}
	for i := 0; i < 50; i++ {
		rng := rand.New(rand.NewPCG(uint64(i), 0))
		got := drawPlanetType(rng, 20.0, 2.0)
		if !validOuter[got] {
			t.Errorf("outer system (d=20 AU, frost=2 AU): got %q (should not be rocky)", got)
		}
	}
}

// ── samplePoisson ──────────────────────────────────────────────────────────────

func TestSamplePoisson_ZeroLambda(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 0))
	if n := samplePoisson(rng, 0); n != 0 {
		t.Errorf("Poisson(0) = %d, want 0", n)
	}
	if n := samplePoisson(rng, -1); n != 0 {
		t.Errorf("Poisson(-1) = %d, want 0", n)
	}
}

func TestSamplePoisson_Mean(t *testing.T) {
	// Large sample: mean should be close to λ
	const n = 100_000
	for _, lambda := range []float64{1.0, 3.0, 5.0} {
		sum := 0
		rng := rand.New(rand.NewPCG(99, 0))
		for i := 0; i < n; i++ {
			sum += samplePoisson(rng, lambda)
		}
		mean := float64(sum) / n
		if math.Abs(mean-lambda) > 0.05*lambda {
			t.Errorf("Poisson(λ=%.1f): mean=%.4f, want ≈%.1f (±5%%)", lambda, mean, lambda)
		}
	}
}

// ── logUniform ────────────────────────────────────────────────────────────────

func TestLogUniform_Range(t *testing.T) {
	rng := rand.New(rand.NewPCG(77, 0))
	for i := 0; i < 1000; i++ {
		v := logUniform(rng, 0.1, 10.0)
		if v < 0.1 || v > 10.0 {
			t.Errorf("logUniform(0.1, 10.0) = %.6f out of [0.1, 10.0]", v)
		}
	}
}

func TestLogUniform_LogMean(t *testing.T) {
	// Mean in log-space should be midpoint of log-range
	rng := rand.New(rand.NewPCG(88, 0))
	const n = 100_000
	logSum := 0.0
	for i := 0; i < n; i++ {
		logSum += math.Log(logUniform(rng, 1.0, 100.0))
	}
	logMean := logSum / n
	expected := (math.Log(1.0) + math.Log(100.0)) / 2 // = ln(10) ≈ 2.303
	if math.Abs(logMean-expected) > 0.02 {
		t.Errorf("logUniform log-mean = %.4f, want %.4f", logMean, expected)
	}
}

// ── drawMassRadius ────────────────────────────────────────────────────────────

func TestDrawMassRadius_TypeRanges(t *testing.T) {
	tests := []struct {
		pType           string
		massMin, massMax float64
		radiusMin, radiusMax float64
	}{
		{"rocky", 0.04, 4.0, 0.3, 2.0},
		{"gas_giant", 15.0, 5000.0, 2.5, 16.0},
		{"ice_giant", 2.0, 70.0, 1.0, 7.0},
	}
	for _, tc := range tests {
		for i := 0; i < 200; i++ {
			rng := rand.New(rand.NewPCG(uint64(i), 0))
			mass, radius := drawMassRadius(rng, tc.pType)
			if mass < tc.massMin || mass > tc.massMax {
				t.Errorf("%s: mass=%.4f out of [%.2f, %.2f]", tc.pType, mass, tc.massMin, tc.massMax)
			}
			if radius < tc.radiusMin || radius > tc.radiusMax {
				t.Errorf("%s: radius=%.4f out of [%.2f, %.2f]", tc.pType, radius, tc.radiusMin, tc.radiusMax)
			}
		}
	}
}

func TestDrawMassRadius_AsteroidBelt(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 0))
	mass, radius := drawMassRadius(rng, "asteroid_belt")
	if mass != 0 || radius != 0 {
		t.Errorf("asteroid_belt: mass=%.4f radius=%.4f, want both 0", mass, radius)
	}
}

// ── computeUsableSurface ──────────────────────────────────────────────────────

func TestComputeUsableSurface_GasGiant(t *testing.T) {
	if got := computeUsableSurface("gas_giant", 0.9, 0.6, 0.05); got != 0 {
		t.Errorf("gas_giant usable surface = %.4f, want 0", got)
	}
	if got := computeUsableSurface("ice_giant", 0.9, 0.6, 0.05); got != 0 {
		t.Errorf("ice_giant usable surface = %.4f, want 0", got)
	}
	if got := computeUsableSurface("asteroid_belt", 0.9, 0.6, 0.05); got != 0 {
		t.Errorf("asteroid_belt usable surface = %.4f, want 0", got)
	}
}

func TestComputeUsableSurface_Rocky(t *testing.T) {
	// With biomass → usable = base * sqrt(biomass)
	got := computeUsableSurface("rocky", 1.0, 0.6, 0.05)
	if math.Abs(got-0.6) > 0.001 {
		t.Errorf("rocky, maxBio=1.0: got %.4f, want 0.6 (base * sqrt(1.0))", got)
	}

	// No biomass → hostile base
	got2 := computeUsableSurface("rocky", 0, 0.6, 0.05)
	if math.Abs(got2-0.05) > 0.001 {
		t.Errorf("rocky, no biomass: got %.4f, want 0.05 (hostile base)", got2)
	}
}
