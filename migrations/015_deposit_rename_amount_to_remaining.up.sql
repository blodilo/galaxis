-- Migration 015: Rename JSONB field "amount" → "remaining" in resource_deposits
-- "amount" was misleading (sounds like extraction-per-tick); "remaining" is the
-- correct semantic: the still-extractable stock in this deposit.

UPDATE planets
SET resource_deposits = (
    SELECT jsonb_object_agg(
        kv.key,
        CASE WHEN jsonb_typeof(kv.value) = 'object' AND kv.value ? 'amount'
        THEN (kv.value - 'amount') || jsonb_build_object('remaining', kv.value->'amount')
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
        CASE WHEN jsonb_typeof(kv.value) = 'object' AND kv.value ? 'amount'
        THEN (kv.value - 'amount') || jsonb_build_object('remaining', kv.value->'amount')
        ELSE kv.value
        END
    )
    FROM jsonb_each(resource_deposits) AS kv
)
WHERE resource_deposits IS NOT NULL
  AND resource_deposits != '{}'::jsonb;
