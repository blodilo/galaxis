-- Migration 003 Rollback: Entfernt schrittweise Generierungs-Status-Werte
ALTER TABLE galaxies DROP CONSTRAINT galaxies_status_check;
ALTER TABLE galaxies ADD CONSTRAINT galaxies_status_check
    CHECK (status IN ('generating', 'ready', 'active', 'error'));
