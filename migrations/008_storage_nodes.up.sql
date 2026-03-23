-- Migration 008: 3-Ebenen Puffer-Topologie
-- Ersetzt den flachen system_storage-Ansatz durch explizite storage_nodes.
-- Jeder Spieler hat pro Standort genau einen Knoten:
--   planetary: Planet-Puffer (unbegrenzt) — Minen und Bodenanlagen
--   orbital:   System-Puffer (begrenzt, ausbaubar) — Aufzüge, Orbital-Anlagen
--   intersystem: Inter-System-Routen (zukünftig)

CREATE TABLE storage_nodes (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id   UUID NOT NULL,
  star_id     UUID NOT NULL REFERENCES stars(id),
  planet_id   UUID REFERENCES planets(id),   -- NULL = orbital
  level       TEXT NOT NULL CHECK (level IN ('planetary', 'orbital', 'intersystem')),
  capacity    FLOAT,                          -- NULL = unbegrenzt (planetar immer NULL)
  storage     JSONB NOT NULL DEFAULT '{}',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ein Orbital-Knoten pro Spieler pro Sternensystem
CREATE UNIQUE INDEX storage_nodes_orbital
  ON storage_nodes(player_id, star_id)
  WHERE planet_id IS NULL AND level = 'orbital';

-- Ein Planetar-Knoten pro Spieler pro Planet
CREATE UNIQUE INDEX storage_nodes_planetary
  ON storage_nodes(player_id, star_id, planet_id)
  WHERE planet_id IS NOT NULL;

CREATE INDEX idx_storage_nodes_player_star ON storage_nodes(player_id, star_id);

-- Bestehende system_storage-Einträge → orbitale Knoten migrieren
INSERT INTO storage_nodes (player_id, star_id, level, capacity, storage)
SELECT player_id, star_id, 'orbital', NULL, contents
FROM system_storage;

-- storage_node_id zu facilities hinzufügen
ALTER TABLE facilities
  ADD COLUMN storage_node_id UUID REFERENCES storage_nodes(id);

-- Bestehende Facilities → orbitalen Knoten des Systems zuweisen
UPDATE facilities f
SET storage_node_id = sn.id
FROM storage_nodes sn
WHERE sn.player_id = f.player_id
  AND sn.star_id   = f.star_id
  AND sn.planet_id IS NULL
  AND sn.level     = 'orbital';
