-- Migration 002 DOWN: Rückkehr zum kategorischen Atmosphärenmodell
ALTER TABLE planets
    DROP COLUMN atmosphere_pressure_atm,
    DROP COLUMN atmosphere_composition,
    DROP COLUMN greenhouse_delta_k,
    DROP COLUMN axial_tilt_deg,
    DROP COLUMN rotation_period_h,
    DROP COLUMN has_rings,
    DROP COLUMN biochem_archetype,
    DROP COLUMN biomass_potential,
    ADD COLUMN atmosphere_type   TEXT             CHECK (atmosphere_type IN ('terran', 'volcanic', 'cryogenic', 'arid', 'none')),
    ADD COLUMN biomass_potential DOUBLE PRECISION CHECK (biomass_potential BETWEEN 0 AND 1);

ALTER TABLE moons
    DROP COLUMN surface_temp_k;
