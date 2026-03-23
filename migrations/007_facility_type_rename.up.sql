-- Migration 007: facility_type Umbenennung (generisch → output-spezifisch)
-- Hintergrund: smelter/refinery/bioreaktor waren zu generisch (mehrere Outputs möglich).
-- Jetzt: jeder facility_type produziert genau ein Gut; Rezepte = Tech-Level-Varianten.
-- Referenz: recipes_v1.1.yaml, game-params_v1.8.yaml

-- smelter → steel_mill (Standard-Stahl ist der häufigste Smelter-Einsatz im MVP)
UPDATE facilities SET facility_type = 'steel_mill'       WHERE facility_type = 'smelter';

-- refinery → semiconductor_plant (häufigster Einsatz im MVP)
UPDATE facilities SET facility_type = 'semiconductor_plant' WHERE facility_type = 'refinery';

-- bioreaktor → biosynth_lab
UPDATE facilities SET facility_type = 'biosynth_lab'     WHERE facility_type = 'bioreaktor';

-- Hinweis: titansteel_forge, chrom_alloy_plant, keramik_plant, fuel_processor,
-- reprocessing_plant, coolant_plant sind neue Typen ohne Altdaten → kein UPDATE nötig.
