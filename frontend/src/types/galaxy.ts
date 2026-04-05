export type StarType =
  | 'O' | 'B' | 'A' | 'F' | 'G' | 'K' | 'M'
  | 'WR' | 'RStar' | 'SStar'
  | 'Pulsar' | 'StellarBH' | 'SMBH'

export type NebulaType = 'HII' | 'SNR' | 'Globular'

export type GalaxyStatus =
  | 'generating'
  | 'morphology' | 'spectral' | 'objects'
  | 'ready' | 'active' | 'error'

export interface Galaxy {
  id: string
  name: string
  seed: number
  status: GalaxyStatus
  star_count: number
  created_at?: string
}

export interface Star {
  id: string
  x: number
  y: number
  z: number
  star_type: StarType
  spectral_class: string
  mass_solar: number
  luminosity_solar: number
  radius_solar: number
  temperature_k: number
  color_hex: string
  nebula_id: string | null
  planets_generated: boolean
}

export interface Nebula {
  id: string
  type: NebulaType
  center_x: number
  center_y: number
  center_z: number
  radius_ly: number
  density: number
}

export interface StarFilter {
  O: boolean; B: boolean; A: boolean; F: boolean
  G: boolean; K: boolean; M: boolean
  WR: boolean; RStar: boolean; SStar: boolean
  Pulsar: boolean; StellarBH: boolean; SMBH: boolean
  HII: boolean; SNR: boolean; Globular: boolean
  showFTLW: boolean
  onlyWithPlanets: boolean
}

// DepositEntry matches the v2 JSONB format (migration 015: remaining = still-extractable stock).
export interface DepositEntry {
  remaining: number // current extractable stock
  quality: number   // geological modifier 0–1
  max_mines: number // max simultaneous extractors
}

export interface Moon {
  id: string
  orbit_index: number
  orbit_distance_au: number
  mass_earth: number
  radius_earth: number
  composition_type: 'rocky' | 'icy' | 'mixed'
  surface_temp_k: number
  resource_deposits: Record<string, DepositEntry>
}

export type PlanetType = 'rocky' | 'gas_giant' | 'ice_giant' | 'asteroid_belt'

export interface Planet {
  id: string
  orbit_index: number
  planet_type: PlanetType
  orbit_distance_au: number
  eccentricity: number
  arg_periapsis_deg: number
  inclination_deg: number
  perihelion_au: number
  aphelion_au: number
  temp_eq_min_k: number
  temp_eq_max_k: number
  mass_earth: number
  radius_earth: number
  surface_gravity_g: number
  atm_pressure_atm: number
  atm_composition: Record<string, number>
  greenhouse_delta_k: number
  surface_temp_k: number
  albedo: number
  axial_tilt_deg: number
  rotation_period_h: number
  has_rings: boolean
  biochem_archetype: string
  biomass_potential: Record<string, number>
  usable_surface_fraction: number
  resource_deposits: Record<string, DepositEntry>
  moons: Moon[]
}

export interface SystemData {
  star: Star
  planets: Planet[]
}

export const DEFAULT_FILTER: StarFilter = {
  O: true, B: true, A: true, F: true,
  G: true, K: true, M: true,
  WR: true, RStar: true, SStar: true,
  Pulsar: true, StellarBH: true, SMBH: true,
  HII: true, SNR: true, Globular: true,
  showFTLW: false,
  onlyWithPlanets: false,
}
