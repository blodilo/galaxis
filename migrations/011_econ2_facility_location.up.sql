-- Migration 011: Facility-Standort über Node statt direktes planet_id-Feld
--
-- Bisher: econ2_facilities.planet_id speicherte den Standort redundant.
-- Neu:    Der Standort einer Facility ergibt sich ausschließlich über
--         econ2_facilities.node_id → econ2_nodes.(planet_id | moon_id).
--         Eine Mine steht auf dem Körper, dessen Vorkommen sie abbaut —
--         das ist der Körper des zugehörigen Nodes.
--
-- Außerdem: moon_id auf econ2_nodes für zukünftige Mond-/Asteroid-Nodes.

ALTER TABLE econ2_nodes
    ADD COLUMN moon_id UUID REFERENCES moons(id);

CREATE UNIQUE INDEX econ2_nodes_moon
    ON econ2_nodes (player_id, star_id, moon_id)
    WHERE moon_id IS NOT NULL AND planet_id IS NULL;

ALTER TABLE econ2_facilities
    DROP COLUMN IF EXISTS planet_id;
