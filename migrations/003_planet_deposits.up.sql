-- planet_deposits: geological ground truth per planet.
-- state JSONB maps good_id → { remaining, max_rate, slots, survey_quality }
-- In economy2 context: max_rate is read as max_mines (max simultaneous mine facilities).
CREATE TABLE planet_deposits (
    planet_id  UUID PRIMARY KEY REFERENCES planets(id),
    state      JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
