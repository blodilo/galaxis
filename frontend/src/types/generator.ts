// Types mirroring internal/config/config.go (json-tagged field names).

export interface GalaxyParams {
  seed: number
  num_stars: number
  radius_ly: number
  type: string
  arms: number
  arm_winding: number
  arm_spread: number
  smbh_mass_solar: number
}

export interface FTLWParams {
  vacuum_base: number
  k_factor: number
  cutoff_percent: number
  voxel_size_ly: number
  coarse_voxel_size_ly: number
  pulsar_multiplier: number
  black_hole_multiplier: number
}

export interface InfoQuality {
  full_detail_threshold: number
  medium_detail_threshold: number
  low_detail_threshold: number
}

export interface SensorsParams {
  optical_k: number
  ftl_k: number
  ship_thermal_k: number
  sensor_ratings: Record<string, number>
  ship_thermal_signatures: Record<string, number>
  info_quality: InfoQuality
  survey_duration_ticks: number
  last_known_position_decay_ticks: number
}

export interface TimeParams {
  strategy_tick_minutes: number
  combat_tick_seconds: number
  combat_opt_in_window_hours: number
  max_action_queue_depth: number
}

export interface EconomyParams {
  detail_mode_efficiency_bonus: number
  detail_mode_upgrade_downtime_ticks: number
  detail_mode_break_even_ticks: number
  base_population_growth_rate: number
  tax_rate_base: number
  planet_surface_cost_exponent: number
  asteroid_yield_multiplier: number
}

export interface PlanetGenParams {
  frost_line_constant_au: number
  atmosphere_type_weights: Record<string, number> | null
  moon_collision_probability: number
  gas_giant_moon_count_min: number
  gas_giant_moon_count_max: number
  max_planets_per_system: number
  usable_surface_terran_base: number
  usable_surface_hostile_base: number
}

export interface ResearchParams {
  base_research_speed: number
  scientist_research_bonus: number
  scientist_risk_reduction: number
  lab_bonus_factor: number
  parallel_research_slots: number
}

export interface CombatParams {
  railgun_base_velocity_km_s: number
  graser_antimaterie_cost_per_shot: number
  sandcaster_intercept_radius_km: number
  combat_arena_radius_km: number
}

export interface ServerParams {
  max_players: number
  max_ai_factions: number
  instance_name: string
}

export interface GameParams {
  galaxy: GalaxyParams
  ftlw: FTLWParams
  sensors: SensorsParams
  time: TimeParams
  economy: EconomyParams
  planet_generation: PlanetGenParams
  research: ResearchParams
  combat: CombatParams
  server: ServerParams
}

export interface MorphologyTemplate {
  id: string
  name: string
  designation: string
  hubble_type: string
  hubble_description: string
  file: string
  thumbnail_url: string
  orientation: string
  credit: string
  resolution_px: [number, number]
}

export interface GenerateRequest extends GameParams {
  name: string
  morphology_id: string
}

export type JobStatus = 'pending' | 'running' | 'done' | 'error'

export interface GenerateJob {
  job_id: string
  status: JobStatus
  galaxy_id: string | null
  error: string
  created_at: string
  updated_at: string
}
