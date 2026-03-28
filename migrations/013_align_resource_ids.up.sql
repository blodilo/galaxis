-- Migration 013: Align economy2 item IDs with planet generator resource IDs.
-- Planet generator uses: iron, silicon, titanium, rare_earth, helium_3
-- Economy2 previously used: iron_ore, silicates, titan, rare_earths, he3
-- This migration updates all JSONB keys and string columns in econ2_* tables.

-- econ2_item_stock: rename item_id column values
UPDATE econ2_item_stock SET item_id = 'iron'     WHERE item_id = 'iron_ore';
UPDATE econ2_item_stock SET item_id = 'silicon'  WHERE item_id = 'silicates';
UPDATE econ2_item_stock SET item_id = 'titanium' WHERE item_id = 'titan';
UPDATE econ2_item_stock SET item_id = 'rare_earth' WHERE item_id = 'rare_earths';
UPDATE econ2_item_stock SET item_id = 'helium_3' WHERE item_id = 'he3';

-- econ2_orders: rename product_id, recipe_id and inputs JSONB
UPDATE econ2_orders SET product_id = 'iron'     WHERE product_id = 'iron_ore';
UPDATE econ2_orders SET product_id = 'silicon'  WHERE product_id = 'silicates';
UPDATE econ2_orders SET product_id = 'titanium' WHERE product_id = 'titan';
UPDATE econ2_orders SET product_id = 'rare_earth' WHERE product_id = 'rare_earths';
UPDATE econ2_orders SET product_id = 'helium_3' WHERE product_id = 'he3';

UPDATE econ2_orders SET recipe_id = 'mine_iron'     WHERE recipe_id = 'mine_iron_ore';
UPDATE econ2_orders SET recipe_id = 'mine_silicon'  WHERE recipe_id = 'mine_silicates';
UPDATE econ2_orders SET recipe_id = 'mine_titanium' WHERE recipe_id = 'mine_titan';
UPDATE econ2_orders SET recipe_id = 'mine_rare_earth' WHERE recipe_id = 'mine_rare_earths';
UPDATE econ2_orders SET recipe_id = 'mine_helium_3' WHERE recipe_id = 'mine_he3';

-- Update inputs JSONB array: replace item_id values inside the array
UPDATE econ2_orders
SET inputs = (
    SELECT jsonb_agg(
        CASE
            WHEN elem->>'item_id' = 'iron_ore'   THEN jsonb_set(elem, '{item_id}', '"iron"')
            WHEN elem->>'item_id' = 'silicates'  THEN jsonb_set(elem, '{item_id}', '"silicon"')
            WHEN elem->>'item_id' = 'titan'      THEN jsonb_set(elem, '{item_id}', '"titanium"')
            WHEN elem->>'item_id' = 'rare_earths' THEN jsonb_set(elem, '{item_id}', '"rare_earth"')
            WHEN elem->>'item_id' = 'he3'        THEN jsonb_set(elem, '{item_id}', '"helium_3"')
            ELSE elem
        END
    )
    FROM jsonb_array_elements(inputs) AS elem
)
WHERE inputs @> '[{"item_id":"iron_ore"}]'::jsonb
   OR inputs @> '[{"item_id":"silicates"}]'::jsonb
   OR inputs @> '[{"item_id":"titan"}]'::jsonb
   OR inputs @> '[{"item_id":"rare_earths"}]'::jsonb
   OR inputs @> '[{"item_id":"he3"}]'::jsonb;

-- Update allocated_inputs JSONB keys
UPDATE econ2_orders
SET allocated_inputs = (allocated_inputs
    - 'iron_ore' - 'silicates' - 'titan' - 'rare_earths' - 'he3'
    || CASE WHEN allocated_inputs ? 'iron_ore'   THEN jsonb_build_object('iron',      allocated_inputs->'iron_ore')   ELSE '{}'::jsonb END
    || CASE WHEN allocated_inputs ? 'silicates'  THEN jsonb_build_object('silicon',   allocated_inputs->'silicates')  ELSE '{}'::jsonb END
    || CASE WHEN allocated_inputs ? 'titan'      THEN jsonb_build_object('titanium',  allocated_inputs->'titan')      ELSE '{}'::jsonb END
    || CASE WHEN allocated_inputs ? 'rare_earths' THEN jsonb_build_object('rare_earth', allocated_inputs->'rare_earths') ELSE '{}'::jsonb END
    || CASE WHEN allocated_inputs ? 'he3'        THEN jsonb_build_object('helium_3',  allocated_inputs->'he3')        ELSE '{}'::jsonb END
)
WHERE allocated_inputs ?| ARRAY['iron_ore','silicates','titan','rare_earths','he3'];

-- econ2_facilities: update deposit_good_id inside config JSONB
UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"iron"')      WHERE config->>'deposit_good_id' = 'iron_ore';
UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"silicon"')   WHERE config->>'deposit_good_id' = 'silicates';
UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"titanium"')  WHERE config->>'deposit_good_id' = 'titan';
UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"rare_earth"') WHERE config->>'deposit_good_id' = 'rare_earths';
UPDATE econ2_facilities SET config = jsonb_set(config, '{deposit_good_id}', '"helium_3"')  WHERE config->>'deposit_good_id' = 'he3';
