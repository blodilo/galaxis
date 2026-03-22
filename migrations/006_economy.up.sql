-- Migration 006: Economy System
-- Creates 5 tables for the production/economy MVP (AP4).
-- State stored in JSONB; relational columns only for FK and WHERE fields.

-- Deposit state per planet (lazy-initialized on first survey).
-- state: { "iron_ore": { "remaining": 49850, "max_rate": 30, "slots": 3,
--           "survey_quality": 0.85 }, ... }
CREATE TABLE planet_deposits (
  planet_id  UUID PRIMARY KEY REFERENCES planets(id),
  state      JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Production facilities.
-- config: { "level": 1, "recipe_id": "titansteel", "ticks_remaining": 2,
--           "efficiency_acc": 0.72, "deposit_id": "iron_ore" }
CREATE TABLE facilities (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id     UUID NOT NULL,
  star_id       UUID NOT NULL REFERENCES stars(id),
  planet_id     UUID REFERENCES planets(id),
  facility_type TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'idle',
  config        JSONB NOT NULL DEFAULT '{}',
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_facilities_star   ON facilities(star_id);
CREATE INDEX idx_facilities_player ON facilities(player_id);
CREATE INDEX idx_facilities_status ON facilities(status);

-- Per-player system storage.
-- contents: { "iron_ore": 47.5, "steel": 12.0, "semiconductor_wafer": 3.2 }
-- Sensitivity class resolved from GoodRegistry at runtime — not stored here.
CREATE TABLE system_storage (
  player_id  UUID NOT NULL,
  star_id    UUID NOT NULL REFERENCES stars(id),
  contents   JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (player_id, star_id)
);

-- Rolling tick log (kept to last 100 ticks per player+system).
-- events: [ { "type": "produced", "facility_id": "...", "good": "titansteel",
--             "qty": 4.0, "acc_before": 0.72, "acc_after": 0.12 }, ... ]
CREATE TABLE production_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id  UUID NOT NULL,
  star_id    UUID NOT NULL REFERENCES stars(id),
  tick_n     BIGINT NOT NULL,
  events     JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_production_log_star_tick ON production_log(star_id, tick_n DESC);

-- Player survey snapshots: persisted knowledge per (player, planet).
-- snapshot: { "iron_ore": { "present": true, "remaining_approx": "40000-60000",
--              "remaining_exact": 49850, "max_rate": 30, "slots": 3 } }
-- Filled fields depend on quality (see survey quality model in game-params).
-- Truth lives in planet_deposits.state; this is "last known state at scan time".
CREATE TABLE player_surveys (
  player_id   UUID NOT NULL,
  planet_id   UUID NOT NULL REFERENCES planets(id),
  surveyed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  tick_n      BIGINT NOT NULL,
  quality     FLOAT NOT NULL,
  snapshot    JSONB NOT NULL,
  PRIMARY KEY (player_id, planet_id)
);
CREATE INDEX idx_player_surveys_player ON player_surveys(player_id);
