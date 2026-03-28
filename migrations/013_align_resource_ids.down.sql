-- Revert migration 013 (resource ID alignment)
UPDATE econ2_item_stock SET item_id = 'iron_ore'   WHERE item_id = 'iron';
UPDATE econ2_item_stock SET item_id = 'silicates'  WHERE item_id = 'silicon';
UPDATE econ2_item_stock SET item_id = 'titan'      WHERE item_id = 'titanium';
UPDATE econ2_item_stock SET item_id = 'rare_earths' WHERE item_id = 'rare_earth';
UPDATE econ2_item_stock SET item_id = 'he3'        WHERE item_id = 'helium_3';

UPDATE econ2_orders SET product_id = 'iron_ore'   WHERE product_id = 'iron';
UPDATE econ2_orders SET product_id = 'silicates'  WHERE product_id = 'silicon';
UPDATE econ2_orders SET product_id = 'titan'      WHERE product_id = 'titanium';
UPDATE econ2_orders SET product_id = 'rare_earths' WHERE product_id = 'rare_earth';
UPDATE econ2_orders SET product_id = 'he3'        WHERE product_id = 'helium_3';

UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"iron_ore"')   WHERE config->>'deposit_good_id' = 'iron';
UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"silicates"')  WHERE config->>'deposit_good_id' = 'silicon';
UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"titan"')      WHERE config->>'deposit_good_id' = 'titanium';
UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"rare_earths"') WHERE config->>'deposit_good_id' = 'rare_earth';
UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"he3"')        WHERE config->>'deposit_good_id' = 'helium_3';
