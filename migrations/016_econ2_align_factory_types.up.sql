-- Migration 016: Align factory types and resource IDs in economy2 tables
--
-- 1. Rename factory_type 'mine' → 'extractor' in econ2_facilities
--    (renamed in code as part of economy2 v2 refactor)
-- 2. Rename factory_type 'smelter' → 'refinery' in econ2_facilities
-- 3. Rename old resource keys in planets/moons.resource_deposits to economy2 IDs
--    (migration 013 aligned econ2_* tables but missed planets.resource_deposits)

-- 1. factory_type rename
UPDATE econ2_facilities SET factory_type = 'extractor' WHERE factory_type = 'mine';
UPDATE econ2_facilities SET factory_type = 'refinery'  WHERE factory_type = 'smelter';

-- also align econ2_orders factory_type
UPDATE econ2_orders SET factory_type = 'extractor' WHERE factory_type = 'mine';
UPDATE econ2_orders SET factory_type = 'refinery'  WHERE factory_type = 'smelter';

-- 2. Rename resource keys in planets.resource_deposits
UPDATE planets
SET resource_deposits = (
    SELECT jsonb_object_agg(
        CASE kv.key
            WHEN 'iron_ore'      THEN 'iron'
            WHEN 'silicates'     THEN 'silicon'
            WHEN 'titan_ore'     THEN 'titanium'
            WHEN 'rare_earths'   THEN 'rare_earth'
            WHEN 'helium3'       THEN 'helium_3'
            WHEN 'uranium_raw'   THEN 'uranium'
            WHEN 'nickel_ore'    THEN 'nickel'
            ELSE kv.key
        END,
        kv.value
    )
    FROM jsonb_each(resource_deposits) AS kv
    -- drop keys with no economy2 equivalent (exotic_matter, platinum_group, water_ice, chrom)
    WHERE kv.key NOT IN ('exotic_matter', 'platinum_group', 'water_ice', 'chrom')
)
WHERE resource_deposits IS NOT NULL
  AND resource_deposits != '{}'::jsonb;

-- 3. Same for moons
UPDATE moons
SET resource_deposits = (
    SELECT jsonb_object_agg(
        CASE kv.key
            WHEN 'iron_ore'      THEN 'iron'
            WHEN 'silicates'     THEN 'silicon'
            WHEN 'titan_ore'     THEN 'titanium'
            WHEN 'rare_earths'   THEN 'rare_earth'
            WHEN 'helium3'       THEN 'helium_3'
            WHEN 'uranium_raw'   THEN 'uranium'
            WHEN 'nickel_ore'    THEN 'nickel'
            ELSE kv.key
        END,
        kv.value
    )
    FROM jsonb_each(resource_deposits) AS kv
    WHERE kv.key NOT IN ('exotic_matter', 'platinum_group', 'water_ice', 'chrom')
)
WHERE resource_deposits IS NOT NULL
  AND resource_deposits != '{}'::jsonb;
