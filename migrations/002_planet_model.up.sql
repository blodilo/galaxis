-- Migration 002: Physikalisches Atmosphärenmodell + Biochemie-Archetypen (AP2)
-- Ersetzt kategorisches atmosphere_type und skalares biomass_potential durch
-- physikalisches Modell: Druck, Zusammensetzung JSONB, Treibhauseffekt.
-- Referenz: progress_v1.0.md (AP2), biochemistry_archetypes_v1.0.yaml, game-params_v1.2.yaml

ALTER TABLE planets
    DROP COLUMN atmosphere_type,
    DROP COLUMN biomass_potential,
    ADD COLUMN atmosphere_pressure_atm  DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN atmosphere_composition   JSONB            NOT NULL DEFAULT '{}',
    ADD COLUMN greenhouse_delta_k       DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN axial_tilt_deg           DOUBLE PRECISION,
    ADD COLUMN rotation_period_h        DOUBLE PRECISION,
    ADD COLUMN has_rings                BOOLEAN          NOT NULL DEFAULT FALSE,
    ADD COLUMN biochem_archetype        TEXT,
    ADD COLUMN biomass_potential        JSONB            NOT NULL DEFAULT '{}';

ALTER TABLE moons
    ADD COLUMN surface_temp_k DOUBLE PRECISION;
