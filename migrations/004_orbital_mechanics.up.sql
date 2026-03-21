-- Migration 004: Elliptische Orbits + Temperaturspanne (BL-12)
-- Fügt Kepler-Orbital-Parameter und Gleichgewichtstemperatur-Bereich zur planets-Tabelle hinzu.
-- Referenz: progress_v1.1.md (BL-12), game-params_v1.3.yaml

ALTER TABLE planets
    ADD COLUMN eccentricity      DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN arg_periapsis_deg DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN inclination_deg   DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN perihelion_au     DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN aphelion_au       DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN temp_eq_min_k     DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN temp_eq_max_k     DOUBLE PRECISION NOT NULL DEFAULT 0;
