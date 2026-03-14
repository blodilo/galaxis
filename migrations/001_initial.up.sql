-- Galaxis – Initiales Datenbankschema
-- Migration: 001_initial
-- Referenz: server-core-map_v1.0.md

-- ── Galaxien ──────────────────────────────────────────────────────────────────
CREATE TABLE galaxies (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    seed       BIGINT NOT NULL,
    config     JSONB NOT NULL,
    -- 'generating' | 'ready' | 'active'
    status     TEXT NOT NULL DEFAULT 'generating'
                   CHECK (status IN ('generating', 'ready', 'active')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Nebel ─────────────────────────────────────────────────────────────────────
CREATE TABLE nebulae (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    galaxy_id UUID NOT NULL REFERENCES galaxies(id) ON DELETE CASCADE,
    -- 'HII' | 'SNR' | 'Globular'
    type      TEXT NOT NULL CHECK (type IN ('HII', 'SNR', 'Globular')),
    center_x  DOUBLE PRECISION NOT NULL,
    center_y  DOUBLE PRECISION NOT NULL,
    center_z  DOUBLE PRECISION NOT NULL,
    radius_ly DOUBLE PRECISION NOT NULL,
    density   DOUBLE PRECISION NOT NULL CHECK (density BETWEEN 0 AND 1)
);

CREATE INDEX idx_nebulae_galaxy ON nebulae (galaxy_id);

-- ── Sterne ────────────────────────────────────────────────────────────────────
CREATE TABLE stars (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    galaxy_id         UUID NOT NULL REFERENCES galaxies(id) ON DELETE CASCADE,
    nebula_id         UUID REFERENCES nebulae(id) ON DELETE SET NULL,
    x                 DOUBLE PRECISION NOT NULL,
    y                 DOUBLE PRECISION NOT NULL,
    z                 DOUBLE PRECISION NOT NULL,
    -- Sterntypen: Hauptreihe + Exotika
    star_type         TEXT NOT NULL
                          CHECK (star_type IN (
                              'O', 'B', 'A', 'F', 'G', 'K', 'M',
                              'WR', 'RStar', 'SStar',
                              'Pulsar', 'StellarBH', 'SMBH'
                          )),
    spectral_class    TEXT,
    mass_solar        DOUBLE PRECISION,
    luminosity_solar  DOUBLE PRECISION,
    radius_solar      DOUBLE PRECISION,
    temperature_k     DOUBLE PRECISION,
    color_hex         TEXT,
    planet_seed       BIGINT NOT NULL,
    planets_generated BOOLEAN NOT NULL DEFAULT FALSE
);

-- Räumliche Queries: BBox-Suche nach Position
CREATE INDEX idx_stars_galaxy_pos ON stars (galaxy_id, x, y, z);
-- FTLW-Berechnung: alle massereichen Objekte
CREATE INDEX idx_stars_mass ON stars (galaxy_id, mass_solar DESC NULLS LAST)
    WHERE mass_solar IS NOT NULL;

-- ── FTLW-Voxelgrid ────────────────────────────────────────────────────────────
-- Jeder Chunk = 10×10×10 Voxel, komprimiert als float32-Array (BYTEA).
-- chunk_x/y/z = Chunk-Koordinate (Voxel-Koordinate / 10).
CREATE TABLE ftlw_chunks (
    galaxy_id UUID    NOT NULL REFERENCES galaxies(id) ON DELETE CASCADE,
    chunk_x   INTEGER NOT NULL,
    chunk_y   INTEGER NOT NULL,
    chunk_z   INTEGER NOT NULL,
    -- komprimiertes float32-Array der FTLW-Werte (zlib-compressed)
    data      BYTEA   NOT NULL,
    PRIMARY KEY (galaxy_id, chunk_x, chunk_y, chunk_z)
);

-- ── FTLW-Overrides (Spieler-modifiziert, Lategame) ───────────────────────────
CREATE TABLE ftlw_overrides (
    galaxy_id      UUID NOT NULL REFERENCES galaxies(id) ON DELETE CASCADE,
    voxel_x        INTEGER NOT NULL,
    voxel_y        INTEGER NOT NULL,
    voxel_z        INTEGER NOT NULL,
    multiplier     DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    built_by       UUID,
    structure_type TEXT CHECK (structure_type IN ('tunnel', 'gate', 'stabilizer')),
    PRIMARY KEY (galaxy_id, voxel_x, voxel_y, voxel_z)
);

-- ── Planetensysteme (JIT-generiert) ──────────────────────────────────────────
CREATE TABLE star_systems (
    star_id      UUID PRIMARY KEY REFERENCES stars(id) ON DELETE CASCADE,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Planeten ──────────────────────────────────────────────────────────────────
CREATE TABLE planets (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    star_id                 UUID NOT NULL REFERENCES stars(id) ON DELETE CASCADE,
    orbit_index             SMALLINT NOT NULL,
    planet_type             TEXT NOT NULL
                                CHECK (planet_type IN (
                                    'rocky', 'gas_giant', 'ice_giant', 'asteroid_belt'
                                )),
    orbit_distance_au       DOUBLE PRECISION NOT NULL,
    mass_earth              DOUBLE PRECISION,
    radius_earth            DOUBLE PRECISION,
    surface_gravity_g       DOUBLE PRECISION,
    atmosphere_type         TEXT CHECK (atmosphere_type IN (
                                'terran', 'volcanic', 'cryogenic', 'arid', 'none'
                            )),
    surface_temp_k          DOUBLE PRECISION,
    albedo                  DOUBLE PRECISION,
    usable_surface_fraction DOUBLE PRECISION CHECK (usable_surface_fraction BETWEEN 0 AND 1),
    biomass_potential       DOUBLE PRECISION CHECK (biomass_potential BETWEEN 0 AND 1),
    resource_deposits       JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_planets_star ON planets (star_id, orbit_index);

-- ── Monde ─────────────────────────────────────────────────────────────────────
CREATE TABLE moons (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    planet_id        UUID NOT NULL REFERENCES planets(id) ON DELETE CASCADE,
    orbit_index      SMALLINT NOT NULL,
    mass_earth       DOUBLE PRECISION,
    radius_earth     DOUBLE PRECISION,
    composition_type TEXT CHECK (composition_type IN ('icy', 'rocky', 'mixed')),
    resource_deposits JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_moons_planet ON moons (planet_id);
