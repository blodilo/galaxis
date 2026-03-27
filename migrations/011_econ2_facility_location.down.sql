DROP INDEX IF EXISTS econ2_nodes_moon;

ALTER TABLE econ2_nodes
    DROP COLUMN IF EXISTS moon_id;

ALTER TABLE econ2_facilities
    ADD COLUMN IF NOT EXISTS planet_id UUID;
