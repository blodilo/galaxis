-- Migration 014 DOWN: Revert deposit model to v1

-- Recreate planet_deposits table
CREATE TABLE IF NOT EXISTS planet_deposits (
    planet_id  UUID PRIMARY KEY REFERENCES planets(id) ON DELETE CASCADE,
    state      JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Revert resource_deposits: {key: {amount, quality, max_mines}} → {key: float (quality)}
UPDATE planets
SET resource_deposits = (
    SELECT jsonb_object_agg(
        kv.key,
        CASE WHEN jsonb_typeof(kv.value) = 'object'
        THEN to_jsonb((kv.value->>'quality')::float)
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
        CASE WHEN jsonb_typeof(kv.value) = 'object'
        THEN to_jsonb((kv.value->>'quality')::float)
        ELSE kv.value
        END
    )
    FROM jsonb_each(resource_deposits) AS kv
)
WHERE resource_deposits IS NOT NULL
  AND resource_deposits != '{}'::jsonb;

ALTER TABLE planets DROP COLUMN IF EXISTS untouched;
ALTER TABLE moons   DROP COLUMN IF EXISTS untouched;
