-- Revert migration 012
ALTER TABLE planet_deposits
  DROP CONSTRAINT planet_deposits_planet_id_fkey,
  ADD CONSTRAINT planet_deposits_planet_id_fkey
    FOREIGN KEY (planet_id) REFERENCES planets(id);
