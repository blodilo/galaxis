-- Migration 017: econ2_goals — production goal tracking with BOM-driven order creation

CREATE TABLE econ2_goals (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id           UUID NOT NULL,
  star_id             UUID NOT NULL,
  product_id          TEXT NOT NULL,
  target_qty          FLOAT NOT NULL,
  priority            INT NOT NULL DEFAULT 5,
  status              TEXT NOT NULL DEFAULT 'active',  -- active | completed | cancelled
  transport_overrides JSONB NOT NULL DEFAULT '{}',
  -- Format: { "item_id": "star_uuid" } — BOM node uses transport from that star instead of local production
  created_at          TIMESTAMPTZ DEFAULT now()
);

ALTER TABLE econ2_orders
  ADD COLUMN goal_id UUID REFERENCES econ2_goals(id) ON DELETE SET NULL;
