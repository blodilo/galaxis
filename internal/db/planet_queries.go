package db

import (
	"context"
	"encoding/json"
	"fmt"

	"galaxis/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InsertPlanets batch-inserts planets using pgx.Batch.
func InsertPlanets(ctx context.Context, pool *pgxpool.Pool, planets []model.Planet) error {
	if len(planets) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, p := range planets {
		compJSON, _ := json.Marshal(p.AtmComposition)
		bioJSON, _ := json.Marshal(p.BiomassPotential)
		resJSON, _ := json.Marshal(p.ResourceDeposits)

		var archArg interface{}
		if p.BiochemArchetype != "" {
			archArg = p.BiochemArchetype
		}

		batch.Queue(`
			INSERT INTO planets (
				id, star_id, orbit_index, planet_type, orbit_distance_au,
				eccentricity, arg_periapsis_deg, inclination_deg, perihelion_au, aphelion_au,
				temp_eq_min_k, temp_eq_max_k,
				mass_earth, radius_earth, surface_gravity_g,
				atmosphere_pressure_atm, atmosphere_composition, greenhouse_delta_k,
				surface_temp_k, albedo,
				axial_tilt_deg, rotation_period_h, has_rings,
				biochem_archetype, biomass_potential,
				usable_surface_fraction, resource_deposits
			) VALUES (
				$1,$2,$3,$4,$5,
				$6,$7,$8,$9,$10,
				$11,$12,
				$13,$14,$15,
				$16,$17,$18,
				$19,$20,
				$21,$22,$23,
				$24,$25,
				$26,$27
			)`,
			p.ID, p.StarID, p.OrbitIndex, p.PlanetType, p.OrbitDistanceAU,
			p.Eccentricity, p.ArgPeriapsisDeg, p.InclinationDeg, p.PerihelionAU, p.AphelionAU,
			p.TempEqMinK, p.TempEqMaxK,
			p.MassEarth, p.RadiusEarth, p.SurfaceGravityG,
			p.AtmPressureAtm, compJSON, p.GreenhouseDeltaK,
			p.SurfaceTempK, p.Albedo,
			p.AxialTiltDeg, p.RotationPeriodH, p.HasRings,
			archArg, bioJSON,
			p.UsableSurfaceFraction, resJSON,
		)
	}
	results := pool.SendBatch(ctx, batch)
	defer func() { _ = results.Close() }()
	for range planets {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("db: insert planet: %w", err)
		}
	}
	return nil
}

// InsertMoons batch-inserts moons using pgx.Batch.
func InsertMoons(ctx context.Context, pool *pgxpool.Pool, moons []model.Moon) error {
	if len(moons) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, m := range moons {
		resJSON, _ := json.Marshal(m.ResourceDeposits)
		batch.Queue(`
			INSERT INTO moons (
				id, planet_id, orbit_index, orbit_distance_au,
				mass_earth, radius_earth, composition_type,
				surface_temp_k, resource_deposits
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			m.ID, m.PlanetID, m.OrbitIndex, m.OrbitDistanceAU,
			m.MassEarth, m.RadiusEarth, m.CompositionType,
			m.SurfaceTempK, resJSON,
		)
	}
	results := pool.SendBatch(ctx, batch)
	defer func() { _ = results.Close() }()
	for range moons {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("db: insert moon: %w", err)
		}
	}
	return nil
}

// MarkAllPlanetsGenerated sets planets_generated=true for all stars in a galaxy.
func MarkAllPlanetsGenerated(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`UPDATE stars SET planets_generated = true WHERE galaxy_id = $1`,
		galaxyID,
	)
	return err
}

// PlanetStats holds statistics for the balancing check log.
type PlanetStats struct {
	Total           int
	RockyWithAtm    int
	ArchetypeCounts map[string]int
}

// QueryPlanetStats returns planet statistics for the balancing check log.
func QueryPlanetStats(ctx context.Context, pool *pgxpool.Pool, galaxyID uuid.UUID) (*PlanetStats, error) {
	stats := &PlanetStats{ArchetypeCounts: make(map[string]int)}

	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM planets p
		 JOIN stars s ON s.id = p.star_id
		 WHERE s.galaxy_id = $1`,
		galaxyID,
	).Scan(&stats.Total)
	if err != nil {
		return nil, fmt.Errorf("db: planet stats total: %w", err)
	}

	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM planets p
		 JOIN stars s ON s.id = p.star_id
		 WHERE s.galaxy_id = $1
		   AND p.planet_type = 'rocky'
		   AND p.atmosphere_pressure_atm > 0.1`,
		galaxyID,
	).Scan(&stats.RockyWithAtm)
	if err != nil {
		return nil, fmt.Errorf("db: planet stats rocky_with_atm: %w", err)
	}

	rows, err := pool.Query(ctx,
		`SELECT p.biochem_archetype, COUNT(*) FROM planets p
		 JOIN stars s ON s.id = p.star_id
		 WHERE s.galaxy_id = $1 AND p.biochem_archetype IS NOT NULL
		 GROUP BY p.biochem_archetype`,
		galaxyID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: planet stats archetypes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var archID string
		var count int
		if err := rows.Scan(&archID, &count); err != nil {
			return nil, err
		}
		stats.ArchetypeCounts[archID] = count
	}

	return stats, rows.Err()
}

// QueryPlanetsByStarID returns all planets with their moons for a given star.
func QueryPlanetsByStarID(ctx context.Context, pool *pgxpool.Pool, starID uuid.UUID) ([]model.PlanetRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			id, orbit_index, planet_type, orbit_distance_au,
			eccentricity, arg_periapsis_deg, inclination_deg, perihelion_au, aphelion_au,
			temp_eq_min_k, temp_eq_max_k,
			mass_earth, radius_earth, surface_gravity_g,
			atmosphere_pressure_atm, atmosphere_composition, greenhouse_delta_k,
			surface_temp_k, albedo,
			axial_tilt_deg, rotation_period_h, has_rings,
			COALESCE(biochem_archetype, ''), biomass_potential,
			usable_surface_fraction, resource_deposits
		FROM planets
		WHERE star_id = $1
		ORDER BY orbit_index`,
		starID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: query planets: %w", err)
	}
	defer rows.Close()

	var planets []model.PlanetRow
	for rows.Next() {
		var p model.PlanetRow
		var compJSON, bioJSON, resJSON []byte
		if err := rows.Scan(
			&p.ID, &p.OrbitIndex, &p.PlanetType, &p.OrbitDistanceAU,
			&p.Eccentricity, &p.ArgPeriapsisDeg, &p.InclinationDeg, &p.PerihelionAU, &p.AphelionAU,
			&p.TempEqMinK, &p.TempEqMaxK,
			&p.MassEarth, &p.RadiusEarth, &p.SurfaceGravityG,
			&p.AtmPressureAtm, &compJSON, &p.GreenhouseDeltaK,
			&p.SurfaceTempK, &p.Albedo,
			&p.AxialTiltDeg, &p.RotationPeriodH, &p.HasRings,
			&p.BiochemArchetype, &bioJSON,
			&p.UsableSurfaceFraction, &resJSON,
		); err != nil {
			return nil, fmt.Errorf("db: scan planet: %w", err)
		}
		_ = json.Unmarshal(compJSON, &p.AtmComposition)
		_ = json.Unmarshal(bioJSON, &p.BiomassPotential)
		_ = json.Unmarshal(resJSON, &p.ResourceDeposits)
		planets = append(planets, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch moons for each planet.
	for i := range planets {
		pID, _ := uuid.Parse(planets[i].ID)
		moons, err := queryMoonsByPlanetID(ctx, pool, pID)
		if err != nil {
			return nil, err
		}
		planets[i].Moons = moons
	}

	return planets, nil
}

func queryMoonsByPlanetID(ctx context.Context, pool *pgxpool.Pool, planetID uuid.UUID) ([]model.MoonRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, orbit_index, orbit_distance_au, mass_earth, radius_earth,
		       composition_type, surface_temp_k, resource_deposits
		FROM moons WHERE planet_id = $1 ORDER BY orbit_index`,
		planetID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: query moons: %w", err)
	}
	defer rows.Close()

	moons := []model.MoonRow{}
	for rows.Next() {
		var m model.MoonRow
		var resJSON []byte
		if err := rows.Scan(
			&m.ID, &m.OrbitIndex, &m.OrbitDistanceAU, &m.MassEarth, &m.RadiusEarth,
			&m.CompositionType, &m.SurfaceTempK, &resJSON,
		); err != nil {
			return nil, fmt.Errorf("db: scan moon: %w", err)
		}
		_ = json.Unmarshal(resJSON, &m.ResourceDeposits)
		moons = append(moons, m)
	}
	return moons, rows.Err()
}
