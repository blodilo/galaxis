package planet

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"path/filepath"
	"sort"

	"galaxis/internal/config"
	"galaxis/internal/db"
	"galaxis/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Generator generates planetary systems for all stars in a galaxy.
type Generator struct {
	cfg    *config.Config
	biochem *BiochemConfig
}

// NewGenerator creates a Generator and loads the biochemistry archetype config.
// cfg.ConfigDir must point to the directory containing cfg.PlanetGen.BiochemArchetypesFile.
func NewGenerator(cfg *config.Config) (*Generator, error) {
	biochemPath := filepath.Join(cfg.ConfigDir, cfg.PlanetGen.BiochemArchetypesFile)
	biochem, err := LoadBiochem(biochemPath)
	if err != nil {
		return nil, fmt.Errorf("planet: load biochem: %w", err)
	}
	return &Generator{cfg: cfg, biochem: biochem}, nil
}

// GenerateAll generates planets for all stars in the given slice.
// Planets and moons are batch-inserted. After all stars are processed,
// all stars in the galaxy are marked planets_generated=true.
// An optional emit func can be passed for SSE progress reporting.
func (g *Generator) GenerateAll(
	ctx context.Context,
	pool *pgxpool.Pool,
	galaxyID uuid.UUID,
	stars []model.Star,
	emitFns ...func(string, int, int, string),
) error {
	emit := func(string, int, int, string) {}
	if len(emitFns) > 0 && emitFns[0] != nil {
		emit = emitFns[0]
	}
	const (
		logEvery  = 5000
		batchSize = 200
	)

	totalPlanets := 0
	pendingPlanets := make([]model.Planet, 0, batchSize)
	pendingMoons := make([]model.Moon, 0, batchSize*3)

	flush := func() error {
		if len(pendingPlanets) == 0 && len(pendingMoons) == 0 {
			return nil
		}
		if err := db.InsertPlanets(ctx, pool, pendingPlanets); err != nil {
			return fmt.Errorf("insert planets: %w", err)
		}
		if err := db.InsertMoons(ctx, pool, pendingMoons); err != nil {
			return fmt.Errorf("insert moons: %w", err)
		}
		totalPlanets += len(pendingPlanets)
		pendingPlanets = pendingPlanets[:0]
		pendingMoons = pendingMoons[:0]
		return nil
	}

	for i, star := range stars {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Deterministic RNG per star using planet_seed.
		rng := rand.New(rand.NewPCG(uint64(star.PlanetSeed), uint64(star.PlanetSeed>>32)))
		planets, moons := g.generateSystem(rng, star)
		pendingPlanets = append(pendingPlanets, planets...)
		pendingMoons = append(pendingMoons, moons...)

		if len(pendingPlanets) >= batchSize {
			if err := flush(); err != nil {
				return fmt.Errorf("planet gen: flush: %w", err)
			}
		}

		if (i+1)%logEvery == 0 {
			log.Printf("planet gen: %d/%d stars (%d planets so far)",
				i+1, len(stars), totalPlanets+len(pendingPlanets))
			emit("planets", i+1, len(stars), "")
		}
	}

	if err := flush(); err != nil {
		return fmt.Errorf("planet gen: final flush: %w", err)
	}
	emit("planets", len(stars), len(stars), "")

	// Bulk-mark all stars as planets_generated=true.
	if err := db.MarkAllPlanetsGenerated(ctx, pool, galaxyID); err != nil {
		return fmt.Errorf("planet gen: mark generated: %w", err)
	}

	log.Printf("planet gen: complete – %d planets for %d stars", totalPlanets, len(stars))
	g.logStats(ctx, pool, galaxyID)
	return nil
}

// generateSystem generates planets and moons for one star.
func (g *Generator) generateSystem(rng *rand.Rand, star model.Star) ([]model.Planet, []model.Moon) {
	cfg := g.cfg.PlanetGen
	starType := string(star.Type)
	frostLine := FrostLineAU(star.LuminositySolar, cfg.FrostLineConstantAU)

	lambda := cfg.PlanetCountLambda[starType]
	if lambda <= 0 {
		lambda = 2.0
	}
	n := samplePoisson(rng, lambda)
	if n == 0 {
		return nil, nil
	}

	// Orbital spacing via modified Titius-Bode.
	orbits := make([]float64, n)
	orbits[0] = 0.1 + rng.Float64()*0.3 // first orbit: 0.1–0.4 AU
	for i := 1; i < n; i++ {
		factor := 1.6 + rng.Float64()*0.5 // spacing factor 1.6–2.1
		orbits[i] = orbits[i-1] * factor
	}

	planets := make([]model.Planet, 0, n)
	var moons []model.Moon

	for orbitIdx, d := range orbits {
		p, pMoons := g.generatePlanet(rng, star, orbitIdx, d, frostLine, starType)
		planets = append(planets, p)
		moons = append(moons, pMoons...)
	}

	return planets, moons
}

// generatePlanet generates a single planet and its moons.
func (g *Generator) generatePlanet(
	rng *rand.Rand,
	star model.Star,
	orbitIdx int,
	distAU float64,
	frostLine float64,
	starType string,
) (model.Planet, []model.Moon) {
	cfg := g.cfg.PlanetGen

	pID := uuid.New()
	pType := drawPlanetType(rng, distAU, frostLine)

	massEarth, radiusEarth := drawMassRadius(rng, pType)
	var gravityG float64
	if radiusEarth > 0 {
		gravityG = massEarth / (radiusEarth * radiusEarth)
	}

	albedo := drawAlbedo(rng, pType)
	tEq := EquilibriumTempK(star.LuminositySolar, distAU, albedo)

	// Orbital mechanics (BL-12): Kepler ellipse parameters.
	ecc, argPeri, incl, periAU, aphAU := drawOrbitalMechanics(rng, distAU, frostLine)
	tEqMin := EquilibriumTempK(star.LuminositySolar, aphAU, albedo)  // coldest: at aphelion
	tEqMax := EquilibriumTempK(star.LuminositySolar, periAU, albedo) // hottest: at perihelion

	// Atmosphere: rocky planets only.
	var (
		pressure    float64
		composition map[string]float64
		ghDelta     float64
		archID      string
	)
	if pType == "rocky" {
		pressure, composition, ghDelta, archID = g.generateAtmosphere(
			rng, tEq, gravityG, starType, distAU, frostLine)
	}
	_ = archID // used for composition; dominant archetype resolved below

	surfaceTemp := tEq + ghDelta

	// Biomass potential for all enabled archetypes.
	biomassPotential := g.computeBiomassPotential(pressure, surfaceTemp, gravityG)

	// Dominant archetype = highest biomass_potential.
	dominantArch := ""
	maxBio := 0.0
	for aid, bio := range biomassPotential {
		if bio > maxBio {
			maxBio = bio
			dominantArch = aid
		}
	}

	// Usable surface fraction.
	usable := computeUsableSurface(pType, maxBio, cfg.UsableSurfaceBase, cfg.UsableSurfaceHostileBase)

	// Resource deposits.
	isInnerZone := frostLine <= 0 || distAU < frostLine
	resources := GenerateDeposits(rng, starType, pType, isInnerZone)

	// Axial tilt: biased toward low inclinations.
	axialTilt := rng.Float64() * rng.Float64() * 90.0
	if rng.Float64() < 0.10 { // 10% high-tilt / retrograde
		axialTilt = 90.0 + rng.Float64()*90.0
	}

	rotationH := drawRotationPeriod(rng, pType, distAU)
	hasRings := drawHasRings(rng, pType)

	planet := model.Planet{
		ID:                    pID,
		StarID:                star.ID,
		OrbitIndex:            orbitIdx,
		PlanetType:            pType,
		OrbitDistanceAU:       distAU,
		Eccentricity:          ecc,
		ArgPeriapsisDeg:       argPeri,
		InclinationDeg:        incl,
		PerihelionAU:          periAU,
		AphelionAU:            aphAU,
		TempEqMinK:            tEqMin,
		TempEqMaxK:            tEqMax,
		MassEarth:             massEarth,
		RadiusEarth:           radiusEarth,
		SurfaceGravityG:       gravityG,
		AtmPressureAtm:        pressure,
		AtmComposition:        composition,
		GreenhouseDeltaK:      ghDelta,
		SurfaceTempK:          surfaceTemp,
		Albedo:                albedo,
		AxialTiltDeg:          axialTilt,
		RotationPeriodH:       rotationH,
		HasRings:              hasRings,
		BiochemArchetype:      dominantArch,
		BiomassPotential:      biomassPotential,
		UsableSurfaceFraction: usable,
		ResourceDeposits:      resources,
	}

	moons := g.generateMoons(rng, pID, pType, starType, distAU, tEq, frostLine, massEarth, star.MassSolar)
	return planet, moons
}

// generateAtmosphere determines atmosphere for a rocky planet.
// Returns: pressure [atm], composition map, greenhouse delta [K], selected archetype ID.
func (g *Generator) generateAtmosphere(
	rng *rand.Rand,
	tEq, gravityG float64,
	starType string,
	distAU, frostLine float64,
) (float64, map[string]float64, float64, string) {
	cfg := g.cfg.PlanetGen

	// Probability of having an atmosphere.
	if rng.Float64() > atmosphereProbability(starType, distAU, frostLine) {
		return 0, nil, 0, ""
	}

	// Build archetype CDF from target_fraction.
	sortedIDs := g.biochem.SortedIDs
	totalTarget := 0.0
	cdf := make([]float64, len(sortedIDs))
	for i, id := range sortedIDs {
		totalTarget += g.biochem.Balancing.TargetFraction[id]
		cdf[i] = totalTarget
	}

	r := rng.Float64()
	if r > totalTarget {
		// Non-archetype atmosphere (~10% of atmosphere-bearing rocky planets).
		return generateBareAtmosphere(rng)
	}

	// Select archetype by CDF.
	archID := sortedIDs[len(sortedIDs)-1]
	for i, c := range cdf {
		if r <= c {
			archID = sortedIDs[i]
			break
		}
	}

	arch := g.biochem.Archetypes[archID]
	if arch == nil {
		return generateBareAtmosphere(rng)
	}

	// Sample pressure within archetype's range (log-uniform).
	pressure := logUniform(rng, arch.PressureRangeAtm[0], arch.PressureRangeAtm[1])

	// Generate composition from canonical + jitter, renormalize.
	// Sorted iteration ensures deterministic RNG consumption regardless of map order.
	gasKeys := make([]string, 0, len(arch.CanonicalComposition))
	for g := range arch.CanonicalComposition {
		gasKeys = append(gasKeys, g)
	}
	sort.Strings(gasKeys)
	composition := make(map[string]float64, len(arch.CanonicalComposition))
	total := 0.0
	for _, gas := range gasKeys {
		frac := arch.CanonicalComposition[gas]
		jitter := 1.0 + arch.CompositionJitterFraction*(rng.Float64()*2-1)
		v := frac * jitter
		if v < 0 {
			v = 0
		}
		composition[gas] = v
		total += v
	}
	if total > 0 {
		for gas := range composition {
			composition[gas] /= total
		}
	}

	ghDelta := GreenhouseDeltaK(
		composition, pressure, arch,
		cfg.GreenhouseOverlapCorrection,
		cfg.SO2AerosolThresholdWaterAct,
		tEq,
	)

	return pressure, composition, ghDelta, archID
}

// computeBiomassPotential returns biomass_potential for all enabled archetypes.
func (g *Generator) computeBiomassPotential(pressure, tempK, gravityG float64) map[string]float64 {
	result := make(map[string]float64, len(g.biochem.SortedIDs))
	for _, id := range g.biochem.SortedIDs {
		arch := g.biochem.Archetypes[id]
		if pressure <= 0 {
			result[id] = 0
		} else {
			result[id] = BiomassPotential(arch, tempK, pressure, gravityG)
		}
	}
	return result
}

// generateMoons generates moons for a planet using Hill sphere orbital mechanics.
func (g *Generator) generateMoons(
	rng *rand.Rand,
	planetID uuid.UUID,
	pType, starType string,
	distAU, tEq, frostLine float64,
	massEarth, starMassSolar float64,
) []model.Moon {
	cfg := g.cfg.PlanetGen
	rH := hillSphereAU(distAU, massEarth, starMassSolar)

	switch pType {
	case "gas_giant":
		n := cfg.GasGiantMoonCountMin +
			rng.IntN(cfg.GasGiantMoonCountMax-cfg.GasGiantMoonCountMin+1)
		// Inner moon: 0.003–0.02 × r_H (like Io ≈ 0.005 × r_H).
		// Spacing: geometric series with factor 1.8–2.8 per step.
		innerOrbit := logUniform(rng, math.Max(rH*0.003, 1e-5), rH*0.02)
		spacing := 1.8 + rng.Float64()
		moons := make([]model.Moon, n)
		for i := range moons {
			orbitAU := innerOrbit * math.Pow(spacing, float64(i))
			moons[i] = makeMoon(rng, planetID, i, pType, starType, distAU, tEq, frostLine, false, orbitAU)
		}
		return moons

	case "rocky":
		if rng.Float64() < cfg.MoonCollisionProbability {
			// Giant-impact moon (like Earth's Moon: 0.257 × r_H).
			orbitAU := logUniform(rng, math.Max(rH*0.15, 1e-5), math.Max(rH*0.55, 2e-5))
			return []model.Moon{makeMoon(rng, planetID, 0, pType, starType, distAU, tEq, frostLine, true, orbitAU)}
		}

	case "ice_giant":
		n := rng.IntN(3) // 0–2 small moons
		innerOrbit := logUniform(rng, math.Max(rH*0.01, 1e-5), rH*0.05)
		spacing := 1.8 + rng.Float64()
		moons := make([]model.Moon, n)
		for i := range moons {
			orbitAU := innerOrbit * math.Pow(spacing, float64(i))
			moons[i] = makeMoon(rng, planetID, i, pType, starType, distAU, tEq, frostLine, false, orbitAU)
		}
		return moons
	}

	return nil
}

// hillSphereAU computes the Hill sphere radius in AU.
// r_H = d_AU × (m_planet_earth / (3 × M_star_solar × 333000))^(1/3)
// 333000 is the Sun/Earth mass ratio.
func hillSphereAU(distAU, massEarth, starMassSolar float64) float64 {
	if distAU <= 0 || massEarth <= 0 || starMassSolar <= 0 {
		return 0
	}
	return distAU * math.Pow(massEarth/(3.0*starMassSolar*333000.0), 1.0/3.0)
}

// makeMoon creates a single moon record.
func makeMoon(
	rng *rand.Rand,
	planetID uuid.UUID,
	idx int,
	parentType, starType string,
	distAU, tEq, frostLine float64,
	collision bool,
	orbitDistanceAU float64,
) model.Moon {
	var massEarth, radiusEarth float64
	var comp string

	if collision {
		// Giant-impact moon (like Earth's Moon): rocky, ~1–5% of Earth mass.
		massEarth = 0.005 + rng.Float64()*0.020
		radiusEarth = math.Cbrt(massEarth) * (0.8 + rng.Float64()*0.3)
		comp = "rocky"
	} else if parentType == "gas_giant" {
		// Inner gas-giant moon: rocky (like Io); outer: icy (like Ganymede).
		if idx == 0 {
			comp = "rocky"
			massEarth = 0.001 + rng.Float64()*0.05
		} else {
			comp = "icy"
			if rng.Float64() < 0.3 {
				comp = "mixed"
			}
			massEarth = 0.001 + rng.Float64()*0.03
		}
		radiusEarth = math.Cbrt(massEarth) * (0.7 + rng.Float64()*0.5)
	} else {
		comp = "icy"
		massEarth = 0.0001 + rng.Float64()*0.005
		radiusEarth = math.Cbrt(massEarth) * (0.6 + rng.Float64()*0.5)
	}

	moonTemp := tEq * (0.85 + rng.Float64()*0.25)

	isInner := frostLine <= 0 || distAU < frostLine
	var resources map[string]float64
	if comp == "icy" || comp == "mixed" {
		resources = GenerateDeposits(rng, starType, "ice_giant", false)
	} else {
		resources = GenerateDeposits(rng, starType, "rocky", isInner)
	}

	return model.Moon{
		ID:               uuid.New(),
		PlanetID:         planetID,
		OrbitIndex:       idx,
		OrbitDistanceAU:  orbitDistanceAU,
		MassEarth:        massEarth,
		RadiusEarth:      radiusEarth,
		CompositionType:  comp,
		SurfaceTempK:     moonTemp,
		ResourceDeposits: resources,
	}
}

// logStats prints biochem archetype distribution for the balancing check.
func (g *Generator) logStats(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID) {
	stats, err := db.QueryPlanetStats(ctx, pool, galaxyID)
	if err != nil {
		log.Printf("planet gen: stats query failed: %v", err)
		return
	}

	tol := g.biochem.Balancing.BalanceToleranceFraction
	log.Printf("planet gen: === BALANCING STATS ===")
	log.Printf("planet gen: total planets: %d  rocky_with_atm: %d", stats.Total, stats.RockyWithAtm)
	for _, id := range g.biochem.SortedIDs {
		count := stats.ArchetypeCounts[id]
		target := g.biochem.Balancing.TargetFraction[id]
		actual := 0.0
		if stats.RockyWithAtm > 0 {
			actual = float64(count) / float64(stats.RockyWithAtm)
		}
		diff := math.Abs(actual - target)
		flag := ""
		if diff > tol {
			flag = " ⚠ BALANCING DRIFT"
		}
		log.Printf("planet gen:   %-12s  target=%.0f%%  actual=%.1f%%  n=%d%s",
			id, target*100, actual*100, count, flag)
	}
	log.Printf("planet gen: ===========================")
}

// ── Helper functions ──────────────────────────────────────────────────────────

// drawOrbitalMechanics samples Kepler orbital elements for a planet.
// Returns: eccentricity, argument of periapsis [deg], inclination [deg],
// perihelion [AU], aphelion [AU].
// Eccentricity: Rayleigh-distributed (σ=0.06 inner, 0.12 outer system).
// Inclination:  Rayleigh-distributed (σ=6°), 2% high-inclination tail.
func drawOrbitalMechanics(rng *rand.Rand, distAU, frostLine float64) (ecc, argPeri, incl, periAU, aphAU float64) {
	// Eccentricity — Rayleigh distribution: e = σ·√(−2·ln U)
	sigma := 0.06
	if frostLine > 0 && distAU >= frostLine {
		sigma = 0.12 // outer system orbits are more excited
	}
	ecc = sigma * math.Sqrt(-2*math.Log(math.Max(rng.Float64(), 1e-15)))
	if ecc > 0.85 {
		ecc = 0.85
	}

	// Argument of periapsis: uniform [0°, 360°)
	argPeri = rng.Float64() * 360.0

	// Inclination — Rayleigh σ=6°, 2% high-inclination
	inclSigma := 6.0
	incl = inclSigma * math.Sqrt(-2*math.Log(math.Max(rng.Float64(), 1e-15)))
	if rng.Float64() < 0.02 {
		incl = 20.0 + rng.Float64()*50.0
	}
	if incl > 90.0 {
		incl = 90.0
	}

	periAU = math.Max(distAU*(1-ecc), 1e-4)
	aphAU = distAU * (1 + ecc)
	return
}

func drawPlanetType(rng *rand.Rand, d, frostLine float64) string {
	beyondFrost := frostLine <= 0 || d >= frostLine
	if !beyondFrost {
		return "rocky"
	}

	r := rng.Float64()
	if frostLine > 0 && d < 2*frostLine {
		// Snow line region: mostly gas giants.
		if r < 0.55 {
			return "gas_giant"
		}
		if r < 0.80 {
			return "ice_giant"
		}
		return "asteroid_belt"
	}
	// Outer disk: ice giants dominate.
	if r < 0.25 {
		return "gas_giant"
	}
	if r < 0.60 {
		return "ice_giant"
	}
	return "asteroid_belt"
}

func drawMassRadius(rng *rand.Rand, pType string) (mass, radius float64) {
	noise := func(v, frac float64) float64 {
		return v * (1 + frac*(rng.Float64()*2-1))
	}
	switch pType {
	case "rocky":
		mass = logUniform(rng, 0.05, 3.0)
		radius = noise(math.Pow(mass, 0.27), 0.08)
		if radius < 0.3 {
			radius = 0.3
		}
	case "gas_giant":
		mass = logUniform(rng, 20, 4000)
		// R_Jupiter = 11.2 R_Earth, M_Jupiter = 318 M_Earth
		radius = noise(11.2*math.Pow(mass/318.0, 0.13), 0.10)
		radius = math.Max(3.0, math.Min(15.0, radius))
	case "ice_giant":
		mass = logUniform(rng, 3, 60)
		radius = noise(2.0*math.Pow(mass/15.0, 0.28), 0.10)
		radius = math.Max(1.5, math.Min(6.0, radius))
	case "asteroid_belt":
		return 0, 0
	}
	return
}

func drawAlbedo(rng *rand.Rand, pType string) float64 {
	switch pType {
	case "rocky":
		return 0.07 + rng.Float64()*0.28
	case "gas_giant":
		return 0.30 + rng.Float64()*0.40
	case "ice_giant":
		return 0.40 + rng.Float64()*0.30
	default:
		return 0.10
	}
}

func drawRotationPeriod(rng *rand.Rand, pType string, distAU float64) float64 {
	switch pType {
	case "rocky":
		if distAU < 0.15 {
			// Tidally locked: rough approximation of orbital period in hours.
			return distAU * 8760 * math.Sqrt(distAU)
		}
		return logUniform(rng, 10, 1000)
	case "gas_giant", "ice_giant":
		return 8 + rng.Float64()*20 // fast rotators: 8–28 h
	default:
		return 24
	}
}

func drawHasRings(rng *rand.Rand, pType string) bool {
	switch pType {
	case "gas_giant":
		return rng.Float64() < 0.25
	case "ice_giant":
		return rng.Float64() < 0.15
	case "rocky":
		return rng.Float64() < 0.02
	default:
		return false
	}
}

func atmosphereProbability(starType string, distAU, frostLine float64) float64 {
	switch starType {
	case "Pulsar", "StellarBH", "SMBH":
		return 0.10 // intense radiation strips atmospheres
	}
	if distAU < 0.1 {
		return 0.05 // too close: atmosphere evaporated
	}
	if frostLine > 0 {
		if distAU < 0.3*frostLine {
			return 0.40 // hot inner zone
		}
		if distAU < 1.5*frostLine {
			return 0.80 // habitable zone
		}
	}
	return 0.50 // outer cool zone
}

func generateBareAtmosphere(rng *rand.Rand) (float64, map[string]float64, float64, string) {
	pressure := logUniform(rng, 0.0001, 0.5)
	var comp map[string]float64
	if rng.Float64() < 0.5 {
		comp = map[string]float64{"CO2": 0.95, "N2": 0.05}
	} else {
		comp = map[string]float64{"N2": 0.98, "Ar": 0.02}
	}
	return pressure, comp, 0, ""
}

func computeUsableSurface(pType string, maxBiomass, usableBase, usableHostile float64) float64 {
	switch pType {
	case "gas_giant", "ice_giant", "asteroid_belt":
		return 0
	}
	if maxBiomass > 0 {
		return usableBase * math.Sqrt(maxBiomass)
	}
	return usableHostile
}

// samplePoisson samples from Poisson(λ) using Knuth's algorithm.
func samplePoisson(rng *rand.Rand, lambda float64) int {
	if lambda <= 0 {
		return 0
	}
	L := math.Exp(-lambda)
	k := 0
	p := 1.0
	for {
		k++
		p *= rng.Float64()
		if p <= L {
			break
		}
	}
	return k - 1
}

// logUniform samples uniformly in log-space [min, max].
func logUniform(rng *rand.Rand, min, max float64) float64 {
	if min <= 0 {
		min = 1e-10
	}
	logMin := math.Log(min)
	logMax := math.Log(max)
	return math.Exp(logMin + rng.Float64()*(logMax-logMin))
}
