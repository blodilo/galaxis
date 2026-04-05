-- Migration 015 DOWN: Rename "remaining" back to "amount"

UPDATE planets
SET resource_deposits = (
    SELECT jsonb_object_agg(
        kv.key,
        CASE WHEN jsonb_typeof(kv.value) = 'object' AND kv.value ? 'remaining'
        THEN (kv.value - 'remaining') || jsonb_build_object('amount', kv.value->'remaining')
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
        CASE WHEN jsonb_typeof(kv.value) = 'object' AND kv.value ? 'remaining'
        THEN (kv.value - 'remaining') || jsonb_build_object('amount', kv.value->'remaining')
        ELSE kv.value
        END
    )
    FROM jsonb_each(resource_deposits) AS kv
)
WHERE resource_deposits IS NOT NULL
  AND resource_deposits != '{}'::jsonb;
