// Visual tuning parameters for the God-Mode viewer and System view.
// All values are runtime-adjustable via VisualTuner and persisted in localStorage.

export interface VisualParams {
  // ── Post-processing ──────────────────────────────────────────────────────────
  exposure: number
  bloomIntensity: number
  bloomThreshold: number
  bloomSmoothing: number

  // ── Stars – global ───────────────────────────────────────────────────────────
  starSizeScale: number     // global size multiplier applied to all type sizes
  starSizeCap: number       // max rendered size in pixels
  starPointScale: number    // distance attenuation factor (higher = bigger at distance)
  starGaussian: number      // falloff sharpness (higher = sharper point)

  // ── Stars – per type base sizes ──────────────────────────────────────────────
  typeSizes: Record<string, number>

  // ── System view – planets ────────────────────────────────────────────────────
  planetVisMax: number      // max radius as fraction of half-gap to neighbor
  planetVisMin: number      // absolute min radius (Three.js units)

  // ── System view – moons ──────────────────────────────────────────────────────
  moonSizeFactor: number    // moon dot radius = planetVisR * moonSizeFactor
  moonOrbitMin: number      // inner orbit = planetVisR * moonOrbitMin
  moonOrbitMax: number      // outer orbit = planetVisR * moonOrbitMax

  // ── Layer visibility ─────────────────────────────────────────────────────────
  layerOrbits: boolean       // Orbitalbahnen einblenden
  layerAxisInfo: boolean     // Rotationsachse + Drehrichtungsbogen
  layerOrbitalChevron: boolean  // Richtungschevron auf Bahn
}

export const DEFAULT_VISUAL_PARAMS: VisualParams = {
  exposure: 1.0,
  bloomIntensity: 0.6,
  bloomThreshold: 0.3,
  bloomSmoothing: 0.9,

  starSizeScale: 1.0,
  starSizeCap: 3.0,
  starPointScale: 300,
  starGaussian: 24,

  typeSizes: {
    SMBH:     8.0,
    StellarBH: 4.0,
    WR:       3.0,
    O:        2.5,
    B:        2.0,
    A:        1.8,
    F:        1.5,
    Pulsar:   2.0,
    RStar:    2.5,
    SStar:    2.2,
    G:        1.2,
    K:        1.0,
    M:        0.8,
  },

  planetVisMax: 0.25,
  planetVisMin: 0.025,

  moonSizeFactor: 0.25,
  moonOrbitMin: 1.8,
  moonOrbitMax: 6.0,

  layerOrbits: true,
  layerAxisInfo: true,
  layerOrbitalChevron: true,
}

export const STORAGE_KEY = 'galaxis_visual_params'

// Range definitions for UI sliders
export interface ParamRange { min: number; max: number; step: number; label: string }

export const PARAM_RANGES: Partial<Record<keyof Omit<VisualParams, 'typeSizes'>, ParamRange>> = {
  exposure:        { min: 0.3,  max: 2.5,  step: 0.05, label: 'Belichtung' },
  bloomIntensity:  { min: 0,    max: 3.0,  step: 0.05, label: 'Bloom Intensität' },
  bloomThreshold:  { min: 0,    max: 1.0,  step: 0.01, label: 'Bloom Schwellwert' },
  bloomSmoothing:  { min: 0,    max: 1.0,  step: 0.01, label: 'Bloom Weichheit' },

  starSizeScale:   { min: 0.1,  max: 5.0,  step: 0.05, label: 'Größe (global)' },
  starSizeCap:     { min: 1.0,  max: 10.0, step: 0.5,  label: 'Max Pixel-Größe' },
  starPointScale:  { min: 50,   max: 800,  step: 10,   label: 'Distanz-Skalierung' },
  starGaussian:    { min: 3,    max: 60,   step: 1,    label: 'Falloff-Schärfe' },

  planetVisMax:    { min: 0.05, max: 0.6,  step: 0.01, label: 'Planet Radius max' },
  planetVisMin:    { min: 0.005,max: 0.1,  step: 0.005,label: 'Planet Radius min' },

  moonSizeFactor:  { min: 0.05, max: 0.6,  step: 0.01, label: 'Mondgröße (× Planet)' },
  moonOrbitMin:    { min: 1.0,  max: 4.0,  step: 0.1,  label: 'Mondorbit innen (×)' },
  moonOrbitMax:    { min: 3.0,  max: 15.0, step: 0.5,  label: 'Mondorbit außen (×)' },
}

export const TYPE_SIZE_RANGES: Record<string, ParamRange> = {
  SMBH:     { min: 1, max: 20, step: 0.5, label: 'SMBH' },
  StellarBH:{ min: 1, max: 12, step: 0.5, label: 'Schwarzes Loch' },
  WR:       { min: 0.5, max: 8, step: 0.5, label: 'Wolf-Rayet' },
  O:        { min: 0.5, max: 8, step: 0.5, label: 'O-Stern' },
  B:        { min: 0.5, max: 6, step: 0.5, label: 'B-Stern' },
  A:        { min: 0.5, max: 6, step: 0.5, label: 'A-Stern' },
  F:        { min: 0.5, max: 5, step: 0.5, label: 'F-Stern' },
  G:        { min: 0.2, max: 4, step: 0.2, label: 'G-Stern' },
  K:        { min: 0.2, max: 4, step: 0.2, label: 'K-Stern' },
  M:        { min: 0.1, max: 3, step: 0.1, label: 'M-Stern' },
  Pulsar:   { min: 0.5, max: 6, step: 0.5, label: 'Pulsar' },
  RStar:    { min: 0.5, max: 8, step: 0.5, label: 'Roter Überriese' },
  SStar:    { min: 0.5, max: 6, step: 0.5, label: 'S-Stern' },
}
