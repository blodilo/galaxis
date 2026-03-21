-- Migration 004 DOWN: Entferne Kepler-Orbital-Parameter

ALTER TABLE planets
    DROP COLUMN eccentricity,
    DROP COLUMN arg_periapsis_deg,
    DROP COLUMN inclination_deg,
    DROP COLUMN perihelion_au,
    DROP COLUMN aphelion_au,
    DROP COLUMN temp_eq_min_k,
    DROP COLUMN temp_eq_max_k;
