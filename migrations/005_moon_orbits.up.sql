-- Migration 005: Physikalische Mondorbit-Abstände (Hill-Sphäre, BL-12 Erweiterung)
ALTER TABLE moons ADD COLUMN orbit_distance_au DOUBLE PRECISION NOT NULL DEFAULT 0;
