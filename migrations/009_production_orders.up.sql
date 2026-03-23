-- Migration 009: Fertigungsaufträge (production_orders) + Pool-Scheduler
-- Ermöglicht auftragbasierte Produktion ohne manuelle Facility-Zuweisung.
-- Der Scheduler weist pro Tick idle Facilities zu aktiven Aufträgen zu.

CREATE TABLE production_orders (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id       UUID NOT NULL,
  star_id         UUID NOT NULL REFERENCES stars(id),
  facility_type   TEXT NOT NULL,   -- welcher Pool (alle Facilities dieses Typs im System)
  recipe_id       TEXT NOT NULL,   -- welches Rezept der Pool ausführen soll
  mode            TEXT NOT NULL CHECK (mode IN ('continuous_full', 'continuous_demand', 'batch')),
  batch_remaining INT,             -- batch-Modus: verbleibende Batches (NULL = kontinuierlich)
  good_id         TEXT,            -- continuous_demand: welches Gut überwacht wird
  min_stock       FLOAT,           -- continuous_demand: produziere wenn Lager < min_stock
  target_stock    FLOAT,           -- continuous_demand: pausiere wenn Lager >= target_stock
  priority        INT NOT NULL DEFAULT 0,  -- höher = höhere Priorität bei Zuweisung
  active          BOOL NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index für Scheduler-Query: alle aktiven Aufträge pro Pool
CREATE INDEX idx_production_orders_active
  ON production_orders(player_id, star_id, facility_type)
  WHERE active;

-- Facilities: welchen Auftrag führt diese Anlage gerade aus?
ALTER TABLE facilities
  ADD COLUMN current_order_id UUID REFERENCES production_orders(id);
