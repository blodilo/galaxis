package db

import (
	"context"
	"fmt"

	"galaxis/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateGalaxy inserts a new galaxy record with status='generating'.
func CreateGalaxy(ctx context.Context, pool *pgxpool.Pool, name string, seed int64, configJSON []byte) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO galaxies (name, seed, config, status)
		 VALUES ($1, $2, $3, 'generating')
		 RETURNING id`,
		name, seed, configJSON,
	).Scan(&id)
	return id, err
}

// SetGalaxyStatus updates the status of a galaxy.
func SetGalaxyStatus(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID, status string) error {
	_, err := pool.Exec(ctx,
		`UPDATE galaxies SET status=$1 WHERE id=$2`,
		status, galaxyID,
	)
	return err
}

// InsertStars batch-inserts a slice of Stars using pgx.Batch.
func InsertStars(ctx context.Context, pool *pgxpool.Pool, stars []model.Star) error {
	if len(stars) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, s := range stars {
		var nebulaID interface{} = nil
		if s.NebulaID != nil {
			nebulaID = *s.NebulaID
		}
		batch.Queue(
			`INSERT INTO stars
			 (id, galaxy_id, nebula_id, x, y, z, star_type, spectral_class,
			  mass_solar, luminosity_solar, radius_solar, temperature_k,
			  color_hex, planet_seed, planets_generated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,false)`,
			s.ID, s.GalaxyID, nebulaID,
			s.X, s.Y, s.Z,
			string(s.Type), s.SpectralClass,
			s.MassSolar, s.LuminositySolar, s.RadiusSolar, s.TemperatureK,
			s.ColorHex, s.PlanetSeed,
		)
	}
	results := pool.SendBatch(ctx, batch)
	defer func() { _ = results.Close() }()
	for range stars {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("db: insert star: %w", err)
		}
	}
	return nil
}

// InsertNebulae batch-inserts nebulae.
func InsertNebulae(ctx context.Context, pool *pgxpool.Pool, nebulae []model.Nebula) error {
	if len(nebulae) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, n := range nebulae {
		batch.Queue(
			`INSERT INTO nebulae
			 (id, galaxy_id, type, center_x, center_y, center_z, radius_ly, density)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			n.ID, n.GalaxyID, string(n.Type),
			n.CenterX, n.CenterY, n.CenterZ, n.RadiusLY, n.Density,
		)
	}
	results := pool.SendBatch(ctx, batch)
	defer func() { _ = results.Close() }()
	for range nebulae {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("db: insert nebula: %w", err)
		}
	}
	return nil
}

// InsertFTLWChunkSlice batch-inserts compressed FTLW chunks.
func InsertFTLWChunkSlice(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID, chunks []model.FTLWChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, c := range chunks {
		batch.Queue(
			`INSERT INTO ftlw_chunks (galaxy_id, chunk_x, chunk_y, chunk_z, data)
			 VALUES ($1,$2,$3,$4,$5)
			 ON CONFLICT (galaxy_id, chunk_x, chunk_y, chunk_z) DO UPDATE SET data=EXCLUDED.data`,
			galaxyID, c.CX, c.CY, c.CZ, c.Data,
		)
	}
	results := pool.SendBatch(ctx, batch)
	defer func() { _ = results.Close() }()
	for range chunks {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("db: insert ftlw chunk: %w", err)
		}
	}
	return nil
}

// QueryGalaxies lists all galaxies with star counts.
func QueryGalaxies(ctx context.Context, pool *pgxpool.Pool) ([]model.GalaxyRow, error) {
	rows, err := pool.Query(ctx,
		`SELECT g.id, g.name, g.seed, g.status,
		        COUNT(s.id)::int AS star_count
		 FROM galaxies g
		 LEFT JOIN stars s ON s.galaxy_id = g.id
		 GROUP BY g.id
		 ORDER BY g.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("db: query galaxies: %w", err)
	}
	defer rows.Close()

	var result []model.GalaxyRow
	for rows.Next() {
		var r model.GalaxyRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Seed, &r.Status, &r.StarCount); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// QueryStarsBbox returns stars within a bounding box (paginated).
func QueryStarsBbox(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID,
	x1, y1, z1, x2, y2, z2 float64, limit, offset int) ([]model.StarRow, error) {

	pgRows, err := pool.Query(ctx,
		`SELECT id, x, y, z, star_type, spectral_class,
		        mass_solar, luminosity_solar, radius_solar, temperature_k,
		        color_hex, nebula_id::text, planets_generated
		 FROM stars
		 WHERE galaxy_id = $1
		   AND x BETWEEN $2 AND $3
		   AND y BETWEEN $4 AND $5
		   AND z BETWEEN $6 AND $7
		 ORDER BY mass_solar DESC NULLS LAST
		 LIMIT $8 OFFSET $9`,
		galaxyID, x1, x2, y1, y2, z1, z2, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("db: query stars bbox: %w", err)
	}
	defer pgRows.Close()

	var result []model.StarRow
	for pgRows.Next() {
		var r model.StarRow
		if err := pgRows.Scan(
			&r.ID, &r.X, &r.Y, &r.Z,
			&r.StarType, &r.SpectralClass,
			&r.MassSolar, &r.LuminositySolar, &r.RadiusSolar, &r.TemperatureK,
			&r.ColorHex, &r.NebulaID, &r.PlanetsGenerated,
		); err != nil {
			return nil, fmt.Errorf("db: scan star: %w", err)
		}
		result = append(result, r)
	}
	return result, pgRows.Err()
}

// QueryStarByID returns full details of a single star.
func QueryStarByID(ctx context.Context, pool *pgxpool.Pool, starID uuid.UUID) (*model.StarRow, error) {
	r := &model.StarRow{}
	err := pool.QueryRow(ctx,
		`SELECT id, x, y, z, star_type, spectral_class,
		        mass_solar, luminosity_solar, radius_solar, temperature_k,
		        color_hex, nebula_id::text, planets_generated
		 FROM stars WHERE id = $1`,
		starID,
	).Scan(
		&r.ID, &r.X, &r.Y, &r.Z,
		&r.StarType, &r.SpectralClass,
		&r.MassSolar, &r.LuminositySolar, &r.RadiusSolar, &r.TemperatureK,
		&r.ColorHex, &r.NebulaID, &r.PlanetsGenerated,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// StarSpectralRecord holds minimal star data for spectral type assignment.
type StarSpectralRecord struct {
	ID            uuid.UUID
	NebulaID      *uuid.UUID
	Type          model.StarType
	PlanetSeed    int64
	SpectralClass string
}

// StarTypeUpdate holds the new spectral data for a single star.
type StarTypeUpdate struct {
	ID              uuid.UUID
	Type            model.StarType
	SpectralClass   string
	MassSolar       float64
	LuminositySolar float64
	RadiusSolar     float64
	TemperatureK    float64
	ColorHex        string
}

// QueryStarsForSpectral returns minimal star data needed for spectral type assignment.
func QueryStarsForSpectral(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID) ([]StarSpectralRecord, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, nebula_id::text, star_type, planet_seed, spectral_class FROM stars WHERE galaxy_id = $1`,
		galaxyID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: query stars for spectral: %w", err)
	}
	defer rows.Close()

	var result []StarSpectralRecord
	for rows.Next() {
		var r StarSpectralRecord
		var nebulaIDStr *string
		if err := rows.Scan(&r.ID, &nebulaIDStr, &r.Type, &r.PlanetSeed, &r.SpectralClass); err != nil {
			return nil, fmt.Errorf("db: scan star spectral: %w", err)
		}
		if nebulaIDStr != nil {
			if id, err := uuid.Parse(*nebulaIDStr); err == nil {
				r.NebulaID = &id
			}
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// QueryStarsFull returns all stars for a galaxy as domain Stars (including planet_seed).
func QueryStarsFull(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID) ([]model.Star, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, galaxy_id, nebula_id, x, y, z, star_type, spectral_class,
		        mass_solar, luminosity_solar, radius_solar, temperature_k,
		        color_hex, planet_seed
		 FROM stars WHERE galaxy_id = $1`,
		galaxyID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: query stars full: %w", err)
	}
	defer rows.Close()

	var result []model.Star
	for rows.Next() {
		var s model.Star
		if err := rows.Scan(
			&s.ID, &s.GalaxyID, &s.NebulaID,
			&s.X, &s.Y, &s.Z,
			&s.Type, &s.SpectralClass,
			&s.MassSolar, &s.LuminositySolar, &s.RadiusSolar, &s.TemperatureK,
			&s.ColorHex, &s.PlanetSeed,
		); err != nil {
			return nil, fmt.Errorf("db: scan star full: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// BulkUpdateStarTypes updates spectral properties for many stars in a batch.
func BulkUpdateStarTypes(ctx context.Context, pool *pgxpool.Pool, updates []StarTypeUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	const batchSize = 1000
	for i := 0; i < len(updates); i += batchSize {
		end := i + batchSize
		if end > len(updates) {
			end = len(updates)
		}
		chunk := updates[i:end]
		batch := &pgx.Batch{}
		for _, u := range chunk {
			batch.Queue(
				`UPDATE stars SET
				 star_type=$2, spectral_class=$3,
				 mass_solar=$4, luminosity_solar=$5,
				 radius_solar=$6, temperature_k=$7, color_hex=$8
				 WHERE id=$1`,
				u.ID, string(u.Type), u.SpectralClass,
				u.MassSolar, u.LuminositySolar, u.RadiusSolar, u.TemperatureK,
				u.ColorHex,
			)
		}
		results := pool.SendBatch(ctx, batch)
		for range chunk {
			if _, err := results.Exec(); err != nil {
				_ = results.Close()
				return fmt.Errorf("db: update star type: %w", err)
			}
		}
		_ = results.Close()
	}
	return nil
}

// DeleteGalaxy removes a galaxy and all associated data (CASCADE).
func DeleteGalaxy(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM galaxies WHERE id=$1`, galaxyID)
	if err != nil {
		return fmt.Errorf("db: delete galaxy: %w", err)
	}
	return nil
}

// QueryNebulae returns all nebulae for a galaxy.
func QueryNebulae(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID) ([]model.NebulaRow, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, type, center_x, center_y, center_z, radius_ly, density
		 FROM nebulae WHERE galaxy_id = $1 ORDER BY radius_ly DESC`,
		galaxyID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: query nebulae: %w", err)
	}
	defer rows.Close()

	var result []model.NebulaRow
	for rows.Next() {
		var r model.NebulaRow
		if err := rows.Scan(
			&r.ID, &r.Type,
			&r.CenterX, &r.CenterY, &r.CenterZ,
			&r.RadiusLY, &r.Density,
		); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
