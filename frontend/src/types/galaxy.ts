export type StarType =
  | 'O' | 'B' | 'A' | 'F' | 'G' | 'K' | 'M'
  | 'WR' | 'RStar' | 'SStar'
  | 'Pulsar' | 'StellarBH' | 'SMBH'

export type NebulaType = 'HII' | 'SNR' | 'Globular'

export interface Galaxy {
  id: string
  name: string
  seed: number
  status: 'generating' | 'ready' | 'active' | 'error'
  star_count: number
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
}

export const DEFAULT_FILTER: StarFilter = {
  O: true, B: true, A: true, F: true,
  G: true, K: true, M: true,
  WR: true, RStar: true, SStar: true,
  Pulsar: true, StellarBH: true, SMBH: true,
  HII: true, SNR: true, Globular: true,
  showFTLW: false,
}
