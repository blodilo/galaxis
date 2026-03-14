package galaxy

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"

	"galaxis/internal/config"
	"galaxis/internal/db"
	"galaxis/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const chunkSize = 10 // each chunk is 10×10×10 voxels

// FTLWGrid holds the in-memory voxel grid and grid dimensions.
type FTLWGrid struct {
	cfg      config.FTLWConfig
	radiusLY float64

	// Grid extents in voxel units
	nx, ny, nz int
	originX, originY, originZ float64 // world coordinate of voxel [0,0,0]

	// flat array: index = ix*ny*nz + iy*nz + iz
	values []float32
}

// newFTLWGrid initialises the grid and fills it with vacuum_base.
func newFTLWGrid(ftlwCfg config.FTLWConfig, radiusLY float64) *FTLWGrid {
	vs := ftlwCfg.VoxelSizeLY
	// Grid spans ±radius in x/y, ±3000 ly in z
	halfZ := 3000.0
	nx := int(math.Ceil(2*radiusLY/vs)) + 2
	ny := int(math.Ceil(2*radiusLY/vs)) + 2
	nz := int(math.Ceil(2*halfZ/vs)) + 2

	g := &FTLWGrid{
		cfg:      ftlwCfg,
		radiusLY: radiusLY,
		nx:       nx, ny: ny, nz: nz,
		originX: -radiusLY - vs,
		originY: -radiusLY - vs,
		originZ: -halfZ - vs,
		values:  make([]float32, nx*ny*nz),
	}
	// Fill with vacuum_base
	vacuum := float32(ftlwCfg.VacuumBase)
	for i := range g.values {
		g.values[i] = vacuum
	}
	return g
}

// add adds the contribution of a single star to all voxels within its cutoff radius.
func (g *FTLWGrid) add(x, y, z, effectiveMass float64) {
	if effectiveMass <= 0 {
		return
	}
	cfg := g.cfg
	vs := cfg.VoxelSizeLY
	threshold := cfg.CutoffPercent / 100.0 * cfg.VacuumBase
	if threshold <= 0 {
		threshold = 1e-6
	}

	// Cutoff radius: r_cutoff = sqrt(k * M_eff / threshold)
	rCutoff := math.Sqrt(cfg.KFactor * effectiveMass / threshold)

	// Find voxel index range
	ixMin := g.worldToVoxel(x-rCutoff, g.originX)
	ixMax := g.worldToVoxel(x+rCutoff, g.originX)
	iyMin := g.worldToVoxel(y-rCutoff, g.originY)
	iyMax := g.worldToVoxel(y+rCutoff, g.originY)
	izMin := g.worldToVoxel(z-rCutoff, g.originZ)
	izMax := g.worldToVoxel(z+rCutoff, g.originZ)

	ixMin = clamp(ixMin, 0, g.nx-1)
	ixMax = clamp(ixMax, 0, g.nx-1)
	iyMin = clamp(iyMin, 0, g.ny-1)
	iyMax = clamp(iyMax, 0, g.ny-1)
	izMin = clamp(izMin, 0, g.nz-1)
	izMax = clamp(izMax, 0, g.nz-1)

	for ix := ixMin; ix <= ixMax; ix++ {
		vx := g.originX + (float64(ix)+0.5)*vs
		for iy := iyMin; iy <= iyMax; iy++ {
			vy := g.originY + (float64(iy)+0.5)*vs
			for iz := izMin; iz <= izMax; iz++ {
				vz := g.originZ + (float64(iz)+0.5)*vs
				dx, dy, dz := vx-x, vy-y, vz-z
				r2 := dx*dx + dy*dy + dz*dz
				if r2 < 1 {
					r2 = 1 // avoid division by zero at star position
				}
				contribution := cfg.KFactor * effectiveMass / r2
				if contribution < threshold {
					continue
				}
				idx := ix*g.ny*g.nz + iy*g.nz + iz
				g.values[idx] += float32(contribution)
			}
		}
	}
}

func (g *FTLWGrid) worldToVoxel(world, origin float64) int {
	return int((world - origin) / g.cfg.VoxelSizeLY)
}

// computeFTLW builds and persists the FTLW grid from all stars.
func (gen *Generator) computeFTLW(ctx context.Context, galaxyID uuid.UUID, stars []Star) error {
	cfg := gen.cfg
	grid := newFTLWGrid(cfg.FTLW, cfg.Galaxy.RadiusLY)

	for i, s := range stars {
		effectiveMass := s.MassSolar
		switch s.Type {
		case StarTypePulsar:
			effectiveMass *= cfg.FTLW.PulsarMultiplier
		case StarTypeStellarBH, StarTypeSMBH:
			effectiveMass *= cfg.FTLW.BlackHoleMultiplier
		}
		grid.add(s.X, s.Y, s.Z, effectiveMass)

		if (i+1)%10000 == 0 {
			log.Printf("ftlw: processed %d/%d stars", i+1, len(stars))
		}
	}

	return persistFTLW(ctx, gen.pool, galaxyID, grid)
}

// persistFTLW writes all non-trivial chunks to the database.
func persistFTLW(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID, grid *FTLWGrid) error {
	numChunksX := (grid.nx + chunkSize - 1) / chunkSize
	numChunksY := (grid.ny + chunkSize - 1) / chunkSize
	numChunksZ := (grid.nz + chunkSize - 1) / chunkSize

	dbChunks := make([]model.FTLWChunk, 0, numChunksX*numChunksY*numChunksZ)
	vacuum := float32(grid.cfg.VacuumBase)

	for cx := 0; cx < numChunksX; cx++ {
		for cy := 0; cy < numChunksY; cy++ {
			for cz := 0; cz < numChunksZ; cz++ {
				raw := make([]byte, chunkSize*chunkSize*chunkSize*4)
				allVacuum := true

				for lx := 0; lx < chunkSize; lx++ {
					ix := cx*chunkSize + lx
					if ix >= grid.nx {
						break
					}
					for ly := 0; ly < chunkSize; ly++ {
						iy := cy*chunkSize + ly
						if iy >= grid.ny {
							break
						}
						for lz := 0; lz < chunkSize; lz++ {
							iz := cz*chunkSize + lz
							if iz >= grid.nz {
								break
							}
							v := grid.values[ix*grid.ny*grid.nz+iy*grid.nz+iz]
							if v != vacuum {
								allVacuum = false
							}
							off := (lx*chunkSize*chunkSize + ly*chunkSize + lz) * 4
							binary.LittleEndian.PutUint32(raw[off:], math.Float32bits(v))
						}
					}
				}

				if allVacuum {
					continue // skip trivial chunks to save space
				}

				compressed, err := zlibCompress(raw)
				if err != nil {
					return fmt.Errorf("ftlw: compress chunk (%d,%d,%d): %w", cx, cy, cz, err)
				}
				dbChunks = append(dbChunks, model.FTLWChunk{CX: cx, CY: cy, CZ: cz, Data: compressed})
			}
		}
	}

	log.Printf("ftlw: persisting %d non-trivial chunks", len(dbChunks))
	return db.InsertFTLWChunkSlice(ctx, pool, galaxyID, dbChunks)
}

func zlibCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
