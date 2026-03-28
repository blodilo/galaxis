-- Migration 012: Add ON DELETE CASCADE to planet_deposits.planet_id
-- Allows galaxy deletion to cascade through stars → planets → planet_deposits.
ALTER TABLE planet_deposits
  DROP CONSTRAINT planet_deposits_planet_id_fkey,
  ADD CONSTRAINT planet_deposits_planet_id_fkey
    FOREIGN KEY (planet_id) REFERENCES planets(id) ON DELETE CASCADE;
