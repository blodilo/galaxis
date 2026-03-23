-- Migration 008 rollback
ALTER TABLE facilities DROP COLUMN IF EXISTS storage_node_id;
DROP TABLE IF EXISTS storage_nodes;
