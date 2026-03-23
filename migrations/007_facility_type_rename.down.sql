-- Migration 007 rollback: neue Typnamen → alte generische Namen
UPDATE facilities SET facility_type = 'smelter'    WHERE facility_type = 'steel_mill';
UPDATE facilities SET facility_type = 'refinery'   WHERE facility_type = 'semiconductor_plant';
UPDATE facilities SET facility_type = 'bioreaktor' WHERE facility_type = 'biosynth_lab';
