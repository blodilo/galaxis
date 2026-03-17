-- Migration 003: Erweiterte Galaxy-Status-Werte für schrittweise Generierung
-- Fügt 'morphology', 'spectral', 'objects' und 'error' zu galaxies_status_check hinzu.

ALTER TABLE galaxies DROP CONSTRAINT galaxies_status_check;
ALTER TABLE galaxies ADD CONSTRAINT galaxies_status_check
    CHECK (status IN ('generating', 'morphology', 'spectral', 'objects', 'ready', 'active', 'error'));
