-- Migration 009 rollback: Fertigungsaufträge entfernen

ALTER TABLE facilities DROP COLUMN IF EXISTS current_order_id;
DROP TABLE IF EXISTS production_orders;
