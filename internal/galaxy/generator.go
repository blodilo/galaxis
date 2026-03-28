package galaxy

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"galaxis/internal/config"
	"galaxis/internal/db"
	"galaxis/internal/planet"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Generator runs the full galaxy generation pipeline.
type Generator struct {
	cfg  *config.Config
	pool *pgxpool.Pool
}

// NewGenerator creates a Generator with the given config and DB pool.
func NewGenerator(cfg *config.Config, pool *pgxpool.Pool) *Generator {
	return &Generator{cfg: cfg, pool: pool}
}

// Run executes the full pipeline for the given galaxy record.
// The galaxy row must already exist in the DB with status='generating'.
func (g *Generator) Run(ctx context.Context, galaxyID uuid.UUID) error {
	start := time.Now()
	cfg := g.cfg.Galaxy
	rng := rand.New(rand.NewPCG(uint64(cfg.Seed), 0))

	density := newDensityField(cfg.Arms, cfg.ArmWinding, cfg.ArmSpread, cfg.RadiusLY)

	log.Printf("gen: step 1 – SMBH")
	smbh, err := g.placeSMBH(ctx, rng, galaxyID)
	if err != nil {
		return fmt.Errorf("gen: SMBH: %w", err)
	}

	log.Printf("gen: step 2 – nebulae")
	nebulae, err := g.generateNebulae(ctx, rng, galaxyID)
	if err != nil {
		return fmt.Errorf("gen: nebulae: %w", err)
	}
	log.Printf("gen: %d nebulae placed", len(nebulae))

	log.Printf("gen: step 3 – stars (%d)", cfg.NumStars)
	stars, err := g.placeStars(ctx, rng, galaxyID, nebulae, density)
	if err != nil {
		return fmt.Errorf("gen: stars: %w", err)
	}
	log.Printf("gen: %d stars placed in %v", len(stars), time.Since(start))

	log.Printf("gen: step 4 – FTLW grid")
	if err := g.computeFTLW(ctx, galaxyID, stars); err != nil {
		return fmt.Errorf("gen: FTLW: %w", err)
	}

	log.Printf("gen: step 5 – planet systems")
	planetGen, err := planet.NewGenerator(g.cfg)
	if err != nil {
		return fmt.Errorf("gen: planet gen init: %w", err)
	}
	allStars := append([]Star{smbh}, stars...) // include SMBH
	if err := planetGen.GenerateAll(ctx, g.pool, galaxyID, allStars); err != nil {
		return fmt.Errorf("gen: planets: %w", err)
	}

	if err := db.SetGalaxyStatus(ctx, g.pool, galaxyID, "ready"); err != nil {
		return fmt.Errorf("gen: set status: %w", err)
	}

	log.Printf("gen: complete in %v", time.Since(start))
	return nil
}

// placeSMBH inserts the central supermassive black hole and returns the Star.
func (g *Generator) placeSMBH(ctx context.Context, rng *rand.Rand, galaxyID uuid.UUID) (Star, error) {
	props := buildStarProps(rng, StarTypeSMBH, g.cfg.Galaxy.SMBHMassSolar)
	id := uuid.New()
	star := Star{
		ID:              id,
		GalaxyID:        galaxyID,
		X:               0, Y: 0, Z: 0,
		Type:            StarTypeSMBH,
		SpectralClass:   props.SpectralClass,
		MassSolar:       props.Mass,
		LuminositySolar: props.Luminosity,
		RadiusSolar:     props.Radius,
		TemperatureK:    props.Temperature,
		ColorHex:        props.ColorHex,
		PlanetSeed:      planetSeed(g.cfg.Galaxy.Seed, id),
	}
	return star, db.InsertStars(ctx, g.pool, []Star{star})
}

// generateNebulae creates and persists nebulae, returning them for star placement.
func (g *Generator) generateNebulae(ctx context.Context, rng *rand.Rand, galaxyID uuid.UUID) ([]Nebula, error) {
	radius := g.cfg.Galaxy.RadiusLY
	nebulae := make([]Nebula, 0, 80)

	// H-II regions: along spiral arms
	for i := 0; i < 30; i++ {
		angle := rng.Float64() * 2 * math.Pi
		r := radius * (0.15 + 0.65*rng.Float64())
		nebulae = append(nebulae, Nebula{
			ID:       uuid.New(),
			GalaxyID: galaxyID,
			Type:     NebulaHII,
			CenterX:  r * math.Cos(angle),
			CenterY:  r * math.Sin(angle),
			CenterZ:  (rng.Float64()*2 - 1) * 500,
			RadiusLY: 500 + rng.Float64()*2000,
			Density:  0.3 + rng.Float64()*0.7,
		})
	}

	// SNR: scattered, predominantly in disk
	for i := 0; i < 20; i++ {
		angle := rng.Float64() * 2 * math.Pi
		r := radius * rng.Float64() * 0.8
		nebulae = append(nebulae, Nebula{
			ID:       uuid.New(),
			GalaxyID: galaxyID,
			Type:     NebulaSNR,
			CenterX:  r * math.Cos(angle),
			CenterY:  r * math.Sin(angle),
			CenterZ:  (rng.Float64()*2 - 1) * 1000,
			RadiusLY: 200 + rng.Float64()*800,
			Density:  0.2 + rng.Float64()*0.5,
		})
	}

	// Globular clusters: halo (high |z|)
	for i := 0; i < 15; i++ {
		angle := rng.Float64() * 2 * math.Pi
		r := radius * (0.1 + 0.6*rng.Float64())
		zSign := 1.0
		if rng.Float64() < 0.5 {
			zSign = -1
		}
		nebulae = append(nebulae, Nebula{
			ID:       uuid.New(),
			GalaxyID: galaxyID,
			Type:     NebulaGlobular,
			CenterX:  r * math.Cos(angle),
			CenterY:  r * math.Sin(angle),
			CenterZ:  zSign * (2000 + rng.Float64()*8000),
			RadiusLY: 300 + rng.Float64()*500,
			Density:  0.5 + rng.Float64()*0.5,
		})
	}

	if err := db.InsertNebulae(ctx, g.pool, nebulae); err != nil {
		return nil, err
	}
	return nebulae, nil
}

// placeStars generates all stars via rejection sampling and persists them.
func (g *Generator) placeStars(ctx context.Context, rng *rand.Rand, galaxyID uuid.UUID, nebulae []Nebula, density *densityField) ([]Star, error) {
	cfg := g.cfg.Galaxy
	maxDens := density.maxDensity()
	stars := make([]Star, 0, cfg.NumStars)
	const batchSize = 1000

	attempts := 0
	for len(stars) < cfg.NumStars {
		attempts++
		// Sample position in bounding box
		x := (rng.Float64()*2 - 1) * cfg.RadiusLY
		y := (rng.Float64()*2 - 1) * cfg.RadiusLY
		z := (rng.Float64()*2 - 1) * 3000 // ±3 scale heights

		// Reject points outside the elliptical galaxy boundary
		if x*x+y*y > cfg.RadiusLY*cfg.RadiusLY {
			continue
		}

		dens := density.Evaluate(x, y, z)
		if rng.Float64()*maxDens > dens {
			continue
		}

		// Find containing nebula (if any)
		var nebulaID *uuid.UUID
		var nebulaType NebulaType
		for i := range nebulae {
			n := &nebulae[i]
			dx, dy, dz := x-n.CenterX, y-n.CenterY, z-n.CenterZ
			if math.Sqrt(dx*dx+dy*dy+dz*dz) <= n.RadiusLY {
				id := n.ID
				nebulaID = &id
				nebulaType = n.Type
				break
			}
		}

		starType := drawStarType(rng, nebulaType)
		props := buildStarProps(rng, starType, cfg.SMBHMassSolar)

		id := uuid.New()
		stars = append(stars, Star{
			ID:              id,
			GalaxyID:        galaxyID,
			NebulaID:        nebulaID,
			X:               x, Y: y, Z: z,
			Type:            starType,
			SpectralClass:   props.SpectralClass,
			MassSolar:       props.Mass,
			LuminositySolar: props.Luminosity,
			RadiusSolar:     props.Radius,
			TemperatureK:    props.Temperature,
			ColorHex:        props.ColorHex,
			PlanetSeed:      planetSeed(cfg.Seed, id),
		})

		// Flush batch
		if len(stars)%batchSize == 0 {
			batch := stars[len(stars)-batchSize:]
			if err := db.InsertStars(ctx, g.pool, batch); err != nil {
				return nil, err
			}
			if len(stars)%10000 == 0 {
				log.Printf("gen: %d/%d stars placed (efficiency: %.1f%%)",
					len(stars), cfg.NumStars,
					float64(len(stars))/float64(attempts)*100)
			}
		}
	}

	// Flush remaining stars
	remaining := len(stars) % batchSize
	if remaining > 0 {
		batch := stars[len(stars)-remaining:]
		if err := db.InsertStars(ctx, g.pool, batch); err != nil {
			return nil, err
		}
	}

	return stars, nil
}

// planetSeed derives a deterministic seed for JIT planet generation.
// seed = first 8 bytes of SHA-256(galaxy_seed || star_uuid).
func planetSeed(galaxySeed int64, starID uuid.UUID) int64 {
	h := sha256.New()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(galaxySeed))
	h.Write(b)
	h.Write(starID[:])
	sum := h.Sum(nil)
	return int64(binary.LittleEndian.Uint64(sum[:8]))
}

// ── Step-by-step generation pipeline ──────────────────────────────────────────

// Step1Morphology runs the image-based morphology pipeline.
// imagePath must point to a JPEG or PNG galaxy image used as the density template.
// emit is called with progress updates; pass nil for no-op.
func (g *Generator) Step1Morphology(ctx context.Context, galaxyID uuid.UUID, imagePath string, emit func(string, int, int, string)) error {
	if emit == nil {
		emit = func(string, int, int, string) {}
	}
	return g.imageStep1Morphology(ctx, galaxyID, imagePath, emit)
}

// Step2Spectral assigns real spectral types to all placeholder stars in the galaxy.
// Uses per-star deterministic RNG derived from each star's planet_seed.
// emit is called with progress updates; pass nil for no-op.
func (g *Generator) Step2Spectral(ctx context.Context, galaxyID uuid.UUID, emit func(string, int, int, string)) error {
	if emit == nil {
		emit = func(string, int, int, string) {}
	}
	start := time.Now()
	cfg := g.cfg.Galaxy

	log.Printf("gen: step2 – load stars for spectral assignment")
	stars, err := db.QueryStarsForSpectral(ctx, g.pool, galaxyID)
	if err != nil {
		return fmt.Errorf("step2: %w", err)
	}

	nebulaeRows, err := db.QueryNebulae(ctx, g.pool, galaxyID)
	if err != nil {
		return fmt.Errorf("step2: nebulae: %w", err)
	}
	nebulaTypes := make(map[string]NebulaType, len(nebulaeRows))
	for _, n := range nebulaeRows {
		nebulaTypes[n.ID] = NebulaType(n.Type)
	}

	total := len(stars)
	emit("spectral", 0, total, fmt.Sprintf("%d Sterne geladen", total))
	log.Printf("gen: step2 – assigning types to %d stars", total)

	updates := make([]db.StarTypeUpdate, 0, total)
	const emitEvery = 5000
	for i, s := range stars {
		if s.Type == StarTypeSMBH {
			continue
		}
		// Stars assigned by the image spectral cascade already have a real
		// SpectralClass ("—" is the placeholder from placeStarsPlaceholder;
		// cascade-assigned stars have a concrete class like "G" or "cascade").
		if s.SpectralClass != "—" {
			continue
		}
		var nebulaType NebulaType
		if s.NebulaID != nil {
			nebulaType = nebulaTypes[s.NebulaID.String()]
		}
		// Derive per-star RNG from planet_seed with a distinct stream constant
		rng := rand.New(rand.NewPCG(uint64(s.PlanetSeed), 0x537EC7A1))
		starType := drawStarType(rng, nebulaType)
		props := buildStarProps(rng, starType, cfg.SMBHMassSolar)
		updates = append(updates, db.StarTypeUpdate{
			ID:              s.ID,
			Type:            starType,
			SpectralClass:   props.SpectralClass,
			MassSolar:       props.Mass,
			LuminositySolar: props.Luminosity,
			RadiusSolar:     props.Radius,
			TemperatureK:    props.Temperature,
			ColorHex:        props.ColorHex,
		})
		if (i+1)%emitEvery == 0 {
			emit("spectral", i+1, total, "")
		}
	}

	if err := db.BulkUpdateStarTypes(ctx, g.pool, updates); err != nil {
		return fmt.Errorf("step2: update: %w", err)
	}
	emit("spectral", total, total, "")
	log.Printf("gen: step2 complete in %v", time.Since(start))
	return db.SetGalaxyStatus(ctx, g.pool, galaxyID, "spectral")
}

// Step3Objects computes the FTLW grid for the galaxy.
// emit is called with progress updates; pass nil for no-op.
func (g *Generator) Step3Objects(ctx context.Context, galaxyID uuid.UUID, emit func(string, int, int, string)) error {
	if emit == nil {
		emit = func(string, int, int, string) {}
	}
	start := time.Now()
	log.Printf("gen: step3 – FTLW grid")

	stars, err := db.QueryStarsFull(ctx, g.pool, galaxyID)
	if err != nil {
		return fmt.Errorf("step3: load stars: %w", err)
	}

	emit("objects", 0, 1, fmt.Sprintf("FTLW-Grid für %d Sterne", len(stars)))
	if err := g.computeFTLW(ctx, galaxyID, stars); err != nil {
		return fmt.Errorf("step3: FTLW: %w", err)
	}
	emit("objects", 1, 1, "")
	log.Printf("gen: step3 complete in %v", time.Since(start))
	return db.SetGalaxyStatus(ctx, g.pool, galaxyID, "objects")
}

// Step4Planets generates planet systems for all stars in the galaxy.
// emit is called with progress updates; pass nil for no-op.
func (g *Generator) Step4Planets(ctx context.Context, galaxyID uuid.UUID, emit func(string, int, int, string)) error {
	if emit == nil {
		emit = func(string, int, int, string) {}
	}
	start := time.Now()
	log.Printf("gen: step4 – planet systems")

	stars, err := db.QueryStarsFull(ctx, g.pool, galaxyID)
	if err != nil {
		return fmt.Errorf("step4: load stars: %w", err)
	}

	planetGen, err := planet.NewGenerator(g.cfg)
	if err != nil {
		return fmt.Errorf("step4: planet gen init: %w", err)
	}
	if err := planetGen.GenerateAll(ctx, g.pool, galaxyID, stars, emit); err != nil {
		return fmt.Errorf("step4: planets: %w", err)
	}
	log.Printf("gen: step4 complete in %v", time.Since(start))
	return db.SetGalaxyStatus(ctx, g.pool, galaxyID, "ready")
}
