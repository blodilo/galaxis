-- Migration 014: Deposit-Modell v2
-- Statt separater planet_deposits-Tabelle lebt der gesamte Deposit-Zustand
-- in planets.resource_deposits JSONB (vorher: {key: float} → jetzt: {key: {amount, quality, max_mines}}).
-- Außerdem: untouched-Flag für Planeten/Monde, damit der Generator erst beim ersten Scan überschrieben wird.

-- 1. untouched-Flag hinzufügen
ALTER TABLE planets ADD COLUMN IF NOT EXISTS untouched BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE moons   ADD COLUMN IF NOT EXISTS untouched BOOLEAN NOT NULL DEFAULT TRUE;

-- 2. resource_deposits konvertieren: {key: float} → {key: {amount, quality, max_mines}}
--    Einträge, die bereits das neue Format haben (jsonb-Objekte), bleiben unverändert.
UPDATE planets
SET resource_deposits = (
    SELECT jsonb_object_agg(
        kv.key,
        CASE WHEN jsonb_typeof(kv.value) = 'number'
        THEN jsonb_build_object(
            'amount',    ROUND((kv.value::text::float * 50000)::numeric, 2),
            'quality',   kv.value::text::float,
            'max_mines', GREATEST(1, LEAST(10, ROUND((4 + (random() * 4 - 2))::numeric)::int))
        )
        ELSE kv.value
        END
    )
    FROM jsonb_each(resource_deposits) AS kv
)
WHERE resource_deposits IS NOT NULL
  AND resource_deposits != '{}'::jsonb;

UPDATE moons
SET resource_deposits = (
    SELECT jsonb_object_agg(
        kv.key,
        CASE WHEN jsonb_typeof(kv.value) = 'number'
        THEN jsonb_build_object(
            'amount',    ROUND((kv.value::text::float * 50000)::numeric, 2),
            'quality',   kv.value::text::float,
            'max_mines', GREATEST(1, LEAST(10, ROUND((4 + (random() * 4 - 2))::numeric)::int))
        )
        ELSE kv.value
        END
    )
    FROM jsonb_each(resource_deposits) AS kv
)
WHERE resource_deposits IS NOT NULL
  AND resource_deposits != '{}'::jsonb;

-- 3. planet_deposits-Tabelle verwerfen (Daten sind in planets.resource_deposits)
DROP TABLE IF EXISTS planet_deposits;
