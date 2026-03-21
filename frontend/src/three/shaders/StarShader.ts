import * as THREE from 'three'
import type { Star } from '../../types/galaxy'
import { NOISE_GLSL, BODY_VERTEX } from './noise.glsl'

// ── Limb-darkening coefficient by spectral type ────────────────────────────────
// Source: Claret (2000), standard solar limb-darkening values as reference
const LIMB_DARK: Record<string, number> = {
  O: 0.28, B: 0.33, A: 0.40, F: 0.52, G: 0.60, K: 0.70, M: 0.80,
  WR: 0.45, RStar: 0.65, SStar: 0.55,
  Pulsar: 0.10, StellarBH: 0.0, SMBH: 0.0,
}

// ── Sunspot relative frequency (proxy: magnetic activity by spectral type) ────
const SPOT_DENSITY: Record<string, number> = {
  O: 0.02, B: 0.04, A: 0.07, F: 0.14, G: 0.28, K: 0.42, M: 0.55,
  WR: 0.0, RStar: 0.20, SStar: 0.15,
  Pulsar: 0.0, StellarBH: 0.0, SMBH: 0.0,
}

// 0=normal star, 1=compact/BH (dark core + edge glow), 2=pulsar
function starTypeCode(type: string): number {
  if (type === 'StellarBH' || type === 'SMBH') return 1
  if (type === 'Pulsar') return 2
  return 0
}

export function uuidSeed(uuid: string): number {
  const hex = uuid.replace(/-/g, '').substring(0, 8)
  return parseInt(hex, 16) / 0xFFFFFFFF
}

// ── Color triad: base / highlight (hotter zones) / dark (sunspots) ────────────
export function starColorTriad(baseHex: string | undefined): {
  base: THREE.Vector3
  highlight: THREE.Vector3
  dark: THREE.Vector3
} {
  const c = new THREE.Color(baseHex ?? '#ffffff')

  // Highlight: mix base toward warm white — convective hot-cell tops
  const h = c.clone().lerp(new THREE.Color('#ffffff'), 0.38)
  h.r = Math.min(1.0, h.r * 1.10)   // slight warm push
  h.g = Math.min(1.0, h.g * 1.04)

  // Dark: sunspot umbra — ~12% base brightness with faint reddish residual
  const d = new THREE.Color(
    Math.min(1, c.r * 0.13 + 0.018),
    c.g * 0.09,
    c.b * 0.07,
  )

  return {
    base:      new THREE.Vector3(c.r, c.g, c.b),
    highlight: new THREE.Vector3(h.r, h.g, h.b),
    dark:      new THREE.Vector3(d.r, d.g, d.b),
  }
}

// ─────────────────────────────────────────────────────────────────────────────

const STAR_FRAGMENT = /* glsl */`
${NOISE_GLSL}

uniform vec3  uBase;          // surface base color
uniform vec3  uHighlight;     // hot granulation tops
uniform vec3  uDark;          // sunspot umbra
uniform float uSeed;
uniform float uLimbDark;      // 0 (none) – 0.8 (strong)
uniform float uSpotDensity;   // 0 – 0.55
uniform int   uTypeCode;      // 0=normal 1=BH 2=pulsar
uniform float uTime;

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));

  // ── Black-hole: dark core + accretion-disk edge glow ────────────────────────
  if (uTypeCode == 1) {
    float edge = pow(1.0 - mu, 3.5);
    gl_FragColor = vec4(mix(vec3(0.0), uBase * 2.5, edge), 1.0);
    return;
  }

  // ── Pulsar: blue-white with pulsating polar beam ─────────────────────────────
  if (uTypeCode == 2) {
    float pulse = sin(uTime * 8.0) * 0.5 + 0.5;
    float beam  = pow(max(0.0, dot(vWorldNormal, vec3(0.0, 1.0, 0.0))), 6.0);
    vec3  col   = uBase * (0.6 + 0.4 * pulse) + vec3(0.4, 0.6, 1.0) * beam * pulse;
    gl_FragColor = vec4(col, 1.0);
    return;
  }

  // ── Normal star ─────────────────────────────────────────────────────────────

  // Limb darkening — Eddington approximation: I = I₀·(1 − u(1 − μ))
  float limb = 1.0 - uLimbDark * (1.0 - mu);

  // ── Animated granulation — two scales blended ────────────────────────────────
  // Supergranulation (large convective cells)
  vec3  gPos1 = vLocalDir * 4.2 + vec3(uSeed * 37.3, uTime * 0.032, uSeed * 53.1);
  float gran1 = fbm(gPos1, 4) * 0.5 + 0.5;
  // Fine granulation (individual granules, faster drift)
  vec3  gPos2 = vLocalDir * 16.0 + vec3(uSeed * 83.7, -uTime * 0.068, uSeed * 17.3);
  float gran2 = fbm(gPos2, 3) * 0.5 + 0.5;
  float gran  = gran1 * 0.58 + gran2 * 0.42;  // [0,1]: bright cells vs dark lanes

  // ── Animated sunspots (large, drift east slowly) ─────────────────────────────
  vec3  sPos   = vLocalDir * 2.8 + vec3(-uTime * 0.011, uSeed * 29.3, uSeed * 0.7);
  float sNoise = fbm(sPos, 3) * 0.5 + 0.5;  // [0,1]

  // Threshold: higher uSpotDensity → lower cutoff → more dark regions
  float threshold = 1.0 - uSpotDensity * 0.72;
  float spots     = smoothstep(threshold - 0.10, threshold + 0.06, sNoise);
  // spots ≈ 0 → sunspot umbra, spots ≈ 1 → normal surface

  // Faculae: bright halos at the spot boundary (just above threshold)
  float faculae = smoothstep(threshold + 0.05, threshold + 0.22, sNoise) * uSpotDensity * 0.22;

  // ── Color mixing ─────────────────────────────────────────────────────────────
  // Granulation: hot cell tops → highlight, dark intergranular lanes → base
  vec3 surfaceC = mix(uBase, uHighlight, gran * 0.58);
  // Sunspots pull toward dark umbra color
  surfaceC = mix(uDark, surfaceC, spots);
  // Faculae: additive bright ring
  surfaceC += uHighlight * faculae;

  // Limb darkening applied to full surface color
  vec3 col = surfaceC * limb;

  // ── Fresnel corona (chromospheric limb glow) ─────────────────────────────────
  float rim  = 1.0 - mu;
  float halo = smoothstep(0.52, 1.0, rim);
  col += uHighlight * halo * 0.60;

  gl_FragColor = vec4(col, 1.0);
}
`

// ─────────────────────────────────────────────────────────────────────────────
// Prominence / Flare shell — separate transparent sphere over the star
// Only for normal stars (not BH/Pulsar).
// ─────────────────────────────────────────────────────────────────────────────

const PROMINENCE_FRAGMENT = /* glsl */`
${NOISE_GLSL}

uniform vec3  uColor;   // star base color (hot, will be boosted)
uniform float uSeed;
uniform float uTime;

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));

  // Only visible near the limb
  float limb     = 1.0 - mu;
  float limbMask = smoothstep(0.18, 0.92, limb);

  // Animated coarse loop structure (prominence arches)
  vec3  f1 = vLocalDir * 5.2 + vec3(uTime * 0.048, uSeed * 23.1, -uTime * 0.031);
  float n1 = fbm(f1, 4) * 0.5 + 0.5;

  // Finer filament detail
  vec3  f2 = vLocalDir * 13.0 + vec3(-uTime * 0.072, uSeed * 41.7, uTime * 0.042);
  float n2 = fbm(f2, 3) * 0.5 + 0.5;

  float flame = n1 * 0.68 + n2 * 0.32;

  // Hard cutoff: only the hottest filament peaks are visible
  float alpha = smoothstep(0.56, 0.76, flame) * limbMask * 0.92;

  // Super-bright, warm prominence color
  vec3 col = uColor * (2.4 + n1 * 1.6);

  gl_FragColor = vec4(col, alpha);
}
`

export function createStarProminenceMaterial(star: Star): THREE.ShaderMaterial {
  const { base } = starColorTriad(star.color_hex)
  return new THREE.ShaderMaterial({
    uniforms: {
      uColor: { value: base },
      uSeed:  { value: uuidSeed(star.id) + 0.5 },  // offset so prominence differs from surface
      uTime:  { value: 0 },
    },
    vertexShader:   BODY_VERTEX,
    fragmentShader: PROMINENCE_FRAGMENT,
    transparent: true,
    depthWrite:  false,
    blending:    THREE.AdditiveBlending,
    side:        THREE.FrontSide,
  })
}

export function createStarMaterial(star: Star): THREE.ShaderMaterial {
  const { base, highlight, dark } = starColorTriad(star.color_hex)
  return new THREE.ShaderMaterial({
    uniforms: {
      uBase:        { value: base },
      uHighlight:   { value: highlight },
      uDark:        { value: dark },
      uSeed:        { value: uuidSeed(star.id) },
      uLimbDark:    { value: LIMB_DARK[star.star_type] ?? 0.5 },
      uSpotDensity: { value: SPOT_DENSITY[star.star_type] ?? 0.2 },
      uTypeCode:    { value: starTypeCode(star.star_type) },
      uTime:        { value: 0 },
    },
    vertexShader:   BODY_VERTEX,
    fragmentShader: STAR_FRAGMENT,
  })
}
