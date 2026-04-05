-- Migration 016 DOWN
UPDATE econ2_facilities SET factory_type = 'mine'   WHERE factory_type = 'extractor';
UPDATE econ2_facilities SET factory_type = 'smelter' WHERE factory_type = 'refinery';
UPDATE econ2_orders     SET factory_type = 'mine'   WHERE factory_type = 'extractor';
UPDATE econ2_orders     SET factory_type = 'smelter' WHERE factory_type = 'refinery';
-- Note: resource key renames in planets/moons are not reversed (data loss for dropped keys).
