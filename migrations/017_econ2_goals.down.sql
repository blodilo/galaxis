-- Migration 017 DOWN
ALTER TABLE econ2_orders DROP COLUMN IF EXISTS goal_id;
DROP TABLE IF EXISTS econ2_goals;
