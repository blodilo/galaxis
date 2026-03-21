package galaxy

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"sort"

	"galaxis/internal/config"
	"galaxis/internal/db"

	"github.com/google/uuid"
)

// imageAnalysis holds the processed image data used for star placement.
type imageAnalysis struct {
	width, height int
	cdf           []float64 // cumulative distribution function over gamma-boosted luma
	totalLum      float64   // cdf[last] = sum of all gamma-boosted luma values
	intensities   []float32 // raw luma per pixel (for Z-spread)
	initClass     []uint8   // nearest main-sequence spectral index (0=O … 6=M) per pixel
}

// mainSequenceOrder maps spectral index (0–6) → StarType.
// O=0, B=1, A=2, F=3, G=4, K=5, M=6
var mainSequenceOrder = [7]StarType{
	StarTypeO, StarTypeB, StarTypeA, StarTypeF, StarTypeG, StarTypeK, StarTypeM,
}

// spectralQuotas is the IMF-derived fraction of the galaxy per spectral class.
var spectralQuotas = [7]float64{
	0.00003, // O
	0.00130, // B
	0.00600, // A
	0.03000, // F
	0.07600, // G
	0.12100, // K
	0.76567, // M (absorbs remainder)
}

// rgbRef holds the reference RGB (0–255 each) for each spectral class, used for
// nearest-colour classification of image pixels.
var rgbRef = [7][3]float64{
	{157, 180, 255}, // O
	{170, 191, 255}, // B
	{202, 216, 255}, // A
	{251, 248, 255}, // F
	{255, 244, 232}, // G
	{255, 221, 180}, // K
	{255, 189, 111}, // M
}

// zoneProbTable[pixelClass][spectralClass] = probability.
// Converts a pixel's nearest-colour class (0=O … 6=M) into a soft probability
// distribution over spectral classes instead of a hard mapping.
// Each row sums to 1.0.  The peak lies at the matched class but with wide spread
// so that spatially correlated pixel colours do not produce visible spectral bands.
//
//	        O      B      A      F      G      K      M
var zoneProbTable = [7][7]float64{
	{0.20, 0.30, 0.25, 0.15, 0.07, 0.02, 0.01}, // O pixel → hot-biased
	{0.10, 0.25, 0.30, 0.20, 0.10, 0.03, 0.02}, // B pixel
	{0.04, 0.12, 0.25, 0.28, 0.18, 0.09, 0.04}, // A pixel
	{0.02, 0.06, 0.15, 0.25, 0.25, 0.18, 0.09}, // F pixel → neutral
	{0.01, 0.03, 0.08, 0.18, 0.25, 0.27, 0.18}, // G pixel
	{0.01, 0.02, 0.04, 0.10, 0.18, 0.28, 0.37}, // K pixel
	{0.00, 0.01, 0.02, 0.06, 0.12, 0.25, 0.54}, // M pixel → cool-biased
}

// sampleSpectralZone samples a spectral class index from the zone probability table.
// zone is the pixel's nearest-colour class (0–6); dart is a uniform [0,1) random value.
func sampleSpectralZone(zone int, dart float64) int {
	cumulative := 0.0
	for ci, p := range zoneProbTable[zone] {
		cumulative += p
		if dart < cumulative {
			return ci
		}
	}
	return 6 // M as fallback
}

// exoticAffinityClass maps an exotic StarType name → spectral index in mainSequenceOrder.
var exoticAffinityClass = map[string]int{
	"WR":        2, // A
	"RStar":     6, // M
	"SStar":     5, // K
	"Pulsar":    1, // B
	"StellarBH": 6, // M
}

// analyzeImage opens imagePath (JPEG or PNG), computes luma-based CDF, stores
// raw intensities for Z-spread, and classifies each pixel by nearest spectral colour.
func analyzeImage(imagePath string) (*imageAnalysis, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("analyzeImage: open %s: %w", imagePath, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("analyzeImage: decode %s: %w", imagePath, err)
	}

	bounds := img.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y
	n := w * h

	cdf := make([]float64, n)
	intensities := make([]float32, n)
	initClass := make([]uint8, n)

	for py := bounds.Min.Y; py < bounds.Max.Y; py++ {
		for px := bounds.Min.X; px < bounds.Max.X; px++ {
			idx := (py-bounds.Min.Y)*w + (px - bounds.Min.X)

			// Go's RGBA() returns pre-multiplied alpha values in [0, 65535]
			r16, g16, b16, _ := img.At(px, py).RGBA()
			r := float64(r16) / 65535.0
			g := float64(g16) / 65535.0
			b := float64(b16) / 65535.0

			// Rec.709 luma
			lum := 0.2989*r + 0.5870*g + 0.1140*b

			// Store raw luma for Z-spread
			intensities[idx] = float32(lum)

			// CDF with gamma=1.5 to boost contrast
			prev := 0.0
			if idx > 0 {
				prev = cdf[idx-1]
			}
			cdf[idx] = prev + math.Pow(lum, 1.5)

			// Nearest spectral class by squared Euclidean RGB distance
			r8 := r * 255.0
			g8 := g * 255.0
			b8 := b * 255.0
			bestClass := uint8(0)
			bestDist := math.MaxFloat64
			for ci, ref := range rgbRef {
				dr := r8 - ref[0]
				dg := g8 - ref[1]
				db := b8 - ref[2]
				dist := dr*dr + dg*dg + db*db
				if dist < bestDist {
					bestDist = dist
					bestClass = uint8(ci)
				}
			}
			initClass[idx] = bestClass
		}
	}

	totalLum := cdf[n-1]
	return &imageAnalysis{
		width:       w,
		height:      h,
		cdf:         cdf,
		totalLum:    totalLum,
		intensities: intensities,
		initClass:   initClass,
	}, nil
}

// samplePixelFromCDF performs a dart throw against the CDF and returns the pixel index.
func samplePixelFromCDF(a *imageAnalysis, dart float64) int {
	idx := sort.Search(len(a.cdf), func(i int) bool {
		return a.cdf[i] >= dart
	})
	if idx >= len(a.cdf) {
		idx = len(a.cdf) - 1
	}
	return idx
}

// generatePositionsFromImage places numStars stars using image-CDF importance sampling.
func generatePositionsFromImage(a *imageAnalysis, numStars int, radiusLY float64, galaxyID uuid.UUID, galaxySeed int64) []Star {
	rng := rand.New(rand.NewPCG(uint64(galaxySeed), 0xC0FFEE1))
	stars := make([]Star, 0, numStars)

	w := a.width
	h := a.height

	for len(stars) < numStars {
		dart := rng.Float64() * a.totalLum
		pixelIdx := samplePixelFromCDF(a, dart)

		px := pixelIdx % w
		py := pixelIdx / w

		// Sub-pixel jitter
		xf := float64(px) + (rng.Float64() - 0.5)
		yf := float64(py) + (rng.Float64() - 0.5)

		// Scale to ly coordinates; image centre → (0,0), Y is flipped
		X := (xf/float64(w/2) - 1.0) * radiusLY
		Y := -(yf/float64(h/2) - 1.0) * radiusLY

		// Z-spread: brighter pixels → thicker disk (hot, young stars cluster in thin disk)
		zSpread := (float64(a.intensities[pixelIdx])*0.06 + 0.005) * radiusLY
		Z := rng.NormFloat64() * zSpread

		// Deep randomization: sample spectral class from a broad probability
		// distribution conditioned on the pixel's temperature zone.
		// This breaks the hard pixel-colour → spectral-class mapping that caused
		// visible spectral bands following the image's colour gradient.
		zone := int(a.initClass[pixelIdx])
		sampledClass := sampleSpectralZone(zone, rng.Float64())

		id := uuid.New()
		stars = append(stars, Star{
			ID:              id,
			GalaxyID:        galaxyID,
			X:               X,
			Y:               Y,
			Z:               Z,
			Type:            StarType(mainSequenceOrder[sampledClass]),
			SpectralClass:   "—", // sentinel: Step2Spectral will skip this star
			MassSolar:       0.3,
			LuminositySolar: 0.01,
			RadiusSolar:     0.3,
			TemperatureK:    3200,
			ColorHex:        "#888888",
			PlanetSeed:      planetSeed(galaxySeed, id),
		})
	}

	return stars
}

// applySpectralCascade rebalances star types to match IMF spectral quotas, then
// assigns physical properties to every star.
func applySpectralCascade(stars []Star, smbhMassSolar float64, galaxySeed int64) {
	n := len(stars)
	if n == 0 {
		return
	}

	// Group star indices by their current spectral class index.
	var classLists [7][]int
	for i, s := range stars {
		// Find which main-sequence index this type maps to
		classIdx := 6 // default to M
		for ci, t := range mainSequenceOrder {
			if t == s.Type {
				classIdx = ci
				break
			}
		}
		classLists[classIdx] = append(classLists[classIdx], i)
	}

	// Compute target counts per class
	var targets [7]int
	sum := 0
	for i := 0; i < 6; i++ {
		targets[i] = int(spectralQuotas[i] * float64(n))
		sum += targets[i]
	}
	targets[6] = n - sum // M absorbs remainder

	// Seeded RNG for shuffling (deterministic cascade)
	rng := rand.New(rand.NewPCG(uint64(galaxySeed), 0xCA5CA8E))

	// Pass 1: cooling wave O→M — excess stars in hotter classes flow to the next cooler class
	for i := 0; i <= 5; i++ {
		if len(classLists[i]) > targets[i] {
			excess := len(classLists[i]) - targets[i]
			// Fisher-Yates shuffle via rng
			list := classLists[i]
			for j := len(list) - 1; j > 0; j-- {
				k := int(rng.Uint64() % uint64(j+1))
				list[j], list[k] = list[k], list[j]
			}
			// Move last `excess` entries to the next class
			classLists[i+1] = append(classLists[i+1], list[len(list)-excess:]...)
			classLists[i] = list[:len(list)-excess]
		}
	}

	// Pass 2: heating pull M→O — deficit in hotter classes pulls from the next cooler class
	for i := 5; i >= 0; i-- {
		deficit := targets[i] - len(classLists[i])
		if deficit > 0 {
			src := &classLists[i+1]
			take := deficit
			if take > len(*src) {
				take = len(*src)
			}
			if take <= 0 {
				continue
			}
			// Shuffle the source
			list := *src
			for j := len(list) - 1; j > 0; j-- {
				k := int(rng.Uint64() % uint64(j+1))
				list[j], list[k] = list[k], list[j]
			}
			classLists[i] = append(classLists[i], list[len(list)-take:]...)
			*src = list[:len(list)-take]
		}
	}

	// Assign final types and physical properties
	for classIdx, indices := range classLists {
		starType := mainSequenceOrder[classIdx]
		for _, idx := range indices {
			stars[idx].Type = starType
			// Per-star deterministic RNG
			starRng := rand.New(rand.NewPCG(uint64(stars[idx].PlanetSeed), 0x537EC7A1))
			props := buildStarProps(starRng, starType, smbhMassSolar)
			stars[idx].SpectralClass = props.SpectralClass
			stars[idx].MassSolar = props.Mass
			stars[idx].LuminositySolar = props.Luminosity
			stars[idx].RadiusSolar = props.Radius
			stars[idx].TemperatureK = props.Temperature
			stars[idx].ColorHex = props.ColorHex
		}
	}
}

// findBrightnessCentroid returns the luminance-weighted centroid as pixel coordinates.
func findBrightnessCentroid(a *imageAnalysis) (cx, cy float64) {
	var sumX, sumY, sumW float64
	w := a.width
	for idx, lum := range a.intensities {
		x := float64(idx % w)
		y := float64(idx / w)
		weight := float64(lum)
		sumX += x * weight
		sumY += y * weight
		sumW += weight
	}
	if sumW == 0 {
		return float64(a.width) / 2, float64(a.height) / 2
	}
	return sumX / sumW, sumY / sumW
}

// placeExotics generates exotic stars and the SMBH from the image.
func placeExotics(ctx context.Context, a *imageAnalysis, cfg *config.Config, galaxyID uuid.UUID, galaxySeed int64) ([]Star, error) {
	rng := rand.New(rand.NewPCG(uint64(galaxySeed), 0xEE071C5))

	type exoticSpec struct {
		typeName string
		starType StarType
		count    int
	}

	ec := cfg.Galaxy.ExoticCounts
	specs := []exoticSpec{
		{"WR", StarTypeWR, ec.WR},
		{"RStar", StarTypeRStar, ec.RStar},
		{"SStar", StarTypeSStar, ec.SStar},
		{"Pulsar", StarTypePulsar, ec.Pulsar},
		{"StellarBH", StarTypeStellarBH, ec.StellarBH},
	}

	var exotics []Star
	w := a.width

	for _, spec := range specs {
		if spec.count <= 0 {
			continue
		}
		targetClassIdx, hasAffinity := exoticAffinityClass[spec.typeName]
		for j := 0; j < spec.count; j++ {
			const maxAttempts = 50000
			var pixelIdx int
			placed := false

			if hasAffinity {
				for attempt := 0; attempt < maxAttempts; attempt++ {
					dart := rng.Float64() * a.totalLum
					candidate := samplePixelFromCDF(a, dart)
					if int(a.initClass[candidate]) == targetClassIdx {
						pixelIdx = candidate
						placed = true
						break
					}
				}
				if !placed {
					log.Printf("image_gen: exotic %s #%d: affinity rejection failed after %d attempts, falling back to unrestricted sample", spec.typeName, j, maxAttempts)
					dart := rng.Float64() * a.totalLum
					pixelIdx = samplePixelFromCDF(a, dart)
				}
			} else {
				dart := rng.Float64() * a.totalLum
				pixelIdx = samplePixelFromCDF(a, dart)
			}

			px := pixelIdx % w
			py := pixelIdx / w
			xf := float64(px) + (rng.Float64() - 0.5)
			yf := float64(py) + (rng.Float64() - 0.5)
			X := (xf/float64(w/2) - 1.0) * cfg.Galaxy.RadiusLY
			Y := -(yf/float64(a.height/2) - 1.0) * cfg.Galaxy.RadiusLY
			zSpread := (float64(a.intensities[pixelIdx])*0.06 + 0.005) * cfg.Galaxy.RadiusLY
			Z := rng.NormFloat64() * zSpread

			id := uuid.New()
			starRng := rand.New(rand.NewPCG(uint64(planetSeed(galaxySeed, id)), 0x537EC7A1))
			props := buildStarProps(starRng, spec.starType, cfg.Galaxy.SMBHMassSolar)

			exotics = append(exotics, Star{
				ID:              id,
				GalaxyID:        galaxyID,
				X:               X,
				Y:               Y,
				Z:               Z,
				Type:            spec.starType,
				SpectralClass:   "cascade", // Step2 skips this sentinel
				MassSolar:       props.Mass,
				LuminositySolar: props.Luminosity,
				RadiusSolar:     props.Radius,
				TemperatureK:    props.Temperature,
				ColorHex:        props.ColorHex,
				PlanetSeed:      planetSeed(galaxySeed, id),
			})
		}
	}

	// SMBH: place at brightness centroid → mapped to (0, 0, 0)
	smbhID := uuid.New()
	smbhRng := rand.New(rand.NewPCG(uint64(planetSeed(galaxySeed, smbhID)), 0x537EC7A1))
	smbhProps := buildStarProps(smbhRng, StarTypeSMBH, cfg.Galaxy.SMBHMassSolar)
	exotics = append(exotics, Star{
		ID:              smbhID,
		GalaxyID:        galaxyID,
		X:               0,
		Y:               0,
		Z:               0,
		Type:            StarTypeSMBH,
		SpectralClass:   smbhProps.SpectralClass,
		MassSolar:       smbhProps.Mass,
		LuminositySolar: smbhProps.Luminosity,
		RadiusSolar:     smbhProps.Radius,
		TemperatureK:    smbhProps.Temperature,
		ColorHex:        smbhProps.ColorHex,
		PlanetSeed:      planetSeed(galaxySeed, smbhID),
	})

	return exotics, nil
}

// imageStep1Morphology is the image-based Step1 orchestrator.
// It replaces the analytical placeStarsPlaceholder pipeline.
func (g *Generator) imageStep1Morphology(ctx context.Context, galaxyID uuid.UUID, imagePath string, emit func(string, int, int, string)) error {
	cfg := g.cfg.Galaxy

	emit("morphology", 0, cfg.NumStars, "Analysiere Bild…")

	a, err := analyzeImage(imagePath)
	if err != nil {
		return fmt.Errorf("imageStep1: analyzeImage: %w", err)
	}
	log.Printf("gen: image %dx%d px → %d Sterne", a.width, a.height, cfg.NumStars)
	emit("morphology", 0, cfg.NumStars, fmt.Sprintf("%d×%d px → %d Sterne", a.width, a.height, cfg.NumStars))

	// Step A: position sampling
	stars := generatePositionsFromImage(a, cfg.NumStars, cfg.RadiusLY, galaxyID, cfg.Seed)

	emit("morphology", cfg.NumStars/2, cfg.NumStars, "Spektralkaskade…")

	// Step B: spectral cascade
	applySpectralCascade(stars, cfg.SMBHMassSolar, cfg.Seed)

	emit("morphology", cfg.NumStars, cfg.NumStars, "Exotika platzieren…")

	// Step C: exotics + SMBH
	exotics, err := placeExotics(ctx, a, g.cfg, galaxyID, cfg.Seed)
	if err != nil {
		return fmt.Errorf("imageStep1: placeExotics: %w", err)
	}

	// Step D: free image memory (~200 MB for large images)
	a = nil

	// Step E: insert main stars in batches of 1000
	const batchSize = 1000
	for i := 0; i < len(stars); i += batchSize {
		end := i + batchSize
		if end > len(stars) {
			end = len(stars)
		}
		if err := db.InsertStars(ctx, g.pool, stars[i:end]); err != nil {
			return fmt.Errorf("imageStep1: insert stars batch %d: %w", i/batchSize, err)
		}
		emit("morphology", end, cfg.NumStars, "")
		if end%10000 == 0 {
			log.Printf("gen: step1: %d/%d stars inserted", end, cfg.NumStars)
		}
	}

	// Step F: insert exotic stars
	if len(exotics) <= batchSize {
		if err := db.InsertStars(ctx, g.pool, exotics); err != nil {
			return fmt.Errorf("imageStep1: insert exotics: %w", err)
		}
	} else {
		for i := 0; i < len(exotics); i += batchSize {
			end := i + batchSize
			if end > len(exotics) {
				end = len(exotics)
			}
			if err := db.InsertStars(ctx, g.pool, exotics[i:end]); err != nil {
				return fmt.Errorf("imageStep1: insert exotics batch %d: %w", i/batchSize, err)
			}
		}
	}

	totalStars := cfg.NumStars + len(exotics)
	emit("morphology", totalStars, totalStars, fmt.Sprintf("%d Sterne + %d Exotika", cfg.NumStars, len(exotics)))
	log.Printf("gen: step1 image complete: %d Sterne + %d Exotika", cfg.NumStars, len(exotics))

	return db.SetGalaxyStatus(ctx, g.pool, galaxyID, "morphology")
}
