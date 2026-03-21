import * as THREE from 'three'
import type { Star } from '../../types/galaxy'
import { NOISE_GLSL, BODY_VERTEX } from './noise.glsl'

// ── Gradient noise (IQ technique, public domain — implementation from scratch) ─
const GRADIENT_NOISE_GLSL = /* glsl */`
vec3 _ghash(vec3 p) {
  p = vec3(dot(p,vec3(127.1,311.7, 74.7)),
           dot(p,vec3(269.5,183.3,246.1)),
           dot(p,vec3(113.5,271.9,124.6)));
  return -1.0 + 2.0*fract(sin(p)*43758.5453);
}
float gnoise(vec3 p) {
  vec3 i=floor(p); vec3 f=fract(p); vec3 u=f*f*(3.0-2.0*f);
  float d000=dot(_ghash(i            ),f            );
  float d100=dot(_ghash(i+vec3(1,0,0)),f-vec3(1,0,0));
  float d010=dot(_ghash(i+vec3(0,1,0)),f-vec3(0,1,0));
  float d110=dot(_ghash(i+vec3(1,1,0)),f-vec3(1,1,0));
  float d001=dot(_ghash(i+vec3(0,0,1)),f-vec3(0,0,1));
  float d101=dot(_ghash(i+vec3(1,0,1)),f-vec3(1,0,1));
  float d011=dot(_ghash(i+vec3(0,1,1)),f-vec3(0,1,1));
  float d111=dot(_ghash(i+vec3(1,1,1)),f-vec3(1,1,1));
  return mix(mix(mix(d000,d100,u.x),mix(d010,d110,u.x),u.y),
             mix(mix(d001,d101,u.x),mix(d011,d111,u.x),u.y),u.z);
}
float gfbm(vec3 p, int n) {
  float v=0.0; float a=0.5;
  if(n>0){v+=a*gnoise(p);p*=2.01;a*=0.5;}
  if(n>1){v+=a*gnoise(p);p*=2.01;a*=0.5;}
  if(n>2){v+=a*gnoise(p);p*=2.01;a*=0.5;}
  if(n>3){v+=a*gnoise(p);p*=2.01;a*=0.5;}
  if(n>4){v+=a*gnoise(p);p*=2.01;a*=0.5;}
  if(n>5){v+=a*gnoise(p);p*=2.01;a*=0.5;}
  return v;
}
`

// ── Sinusoidal / plasma noise (cosNoise FBM, non-linear feedback) ──────────────
// Technique inspired by IQ / Beautypi (SIGGRAPH 2015) — reimplemented from scratch
const COSNOISE_GLSL = /* glsl */`
float cosNoise3(vec3 p) {
  return (sin(p.x)+sin(p.y)+sin(p.z))*0.333;
}
// Rotate xz + scale to avoid axis-aligned repetition
vec3 _rot3(vec3 p) {
  return vec3(0.8*p.x-0.6*p.z, p.y, 0.6*p.x+0.8*p.z)*1.75;
}
// Non-linear turbulent FBM — self-amplifying (spiky plasma surface)
float sunFbm(vec3 p) {
  float h=0.0; float s=0.50;
  h+=s*cosNoise3(p); p=_rot3(p)+vec3(2.41,8.13,5.77); s*=clamp(0.48+0.2*h,0.05,1.0);
  h+=s*cosNoise3(p); p=_rot3(p)+vec3(2.41,8.13,5.77); s*=clamp(0.48+0.2*h,0.05,1.0);
  h+=s*cosNoise3(p); p=_rot3(p)+vec3(2.41,8.13,5.77); s*=clamp(0.48+0.2*h,0.05,1.0);
  h+=s*cosNoise3(p); p=_rot3(p)+vec3(2.41,8.13,5.77); s*=clamp(0.48+0.2*h,0.05,1.0);
  h+=s*cosNoise3(p); p=_rot3(p)+vec3(2.41,8.13,5.77); s*=clamp(0.48+0.2*h,0.05,1.0);
  h+=s*cosNoise3(p);
  return h;
}
`

// ── 2D lattice value noise — technique: hash-based lattice, own implementation ──
// Hash constants differ from reference shaders → visually independent result.
const NOISE2D_GLSL = /* glsl */`
float _n2h(vec2 p) {
  return fract(sin(dot(p, vec2(127.169, 311.905)))*43758.545);
}
float noise2D(vec2 p) {
  vec2 i=floor(p); vec2 f=fract(p);
  f=f*f*(3.0-2.0*f);
  return mix(
    mix(_n2h(i),          _n2h(i+vec2(1.0,0.0)),f.x),
    mix(_n2h(i+vec2(0.0,1.0)),_n2h(i+vec2(1.0,1.0)),f.x),f.y);
}
float fbm2D(vec2 p) {
  float v=0.0,a=0.60;
  v+=noise2D(p)*a; p*=2.01; a*=0.50;
  v+=noise2D(p)*a; p*=2.01; a*=0.50;
  v+=noise2D(p)*a; p*=2.01; a*=0.50;
  v+=noise2D(p)*a; p*=2.01; a*=0.50;
  v+=noise2D(p)*a;
  return min(v,1.0);
}
`

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

// ── Shared compact-object branches (BH, Pulsar) — identical across variants ───
const COMPACT_OBJECTS_GLSL = /* glsl */`
  if (uTypeCode == 1) {
    float edge = pow(1.0-mu, 3.5);
    gl_FragColor = vec4(mix(vec3(0.0), uBase*2.5, edge), 1.0);
    return;
  }
  if (uTypeCode == 2) {
    float pulse = sin(uTime*8.0)*0.5+0.5;
    float beam  = pow(max(0.0, dot(vWorldNormal, vec3(0.0,1.0,0.0))), 6.0);
    vec3  col   = uBase*(0.6+0.4*pulse)+vec3(0.4,0.6,1.0)*beam*pulse;
    gl_FragColor = vec4(col, 1.0);
    return;
  }
`

// ─────────────────────────────────────────────────────────────────────────────
// V0 — original Value Noise granulation
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
uniform float uLuminosity;    // brightness multiplier

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));

${COMPACT_OBJECTS_GLSL}

  // ── Normal star ─────────────────────────────────────────────────────────────

  // Limb darkening — Eddington approximation: I = I₀·(1 − u(1 − μ))
  float limb = 1.0 - uLimbDark * (1.0 - mu);

  // ── Animated granulation — two scales blended ────────────────────────────────
  // Supergranulation (large convective cells)
  vec3  gPos1 = vLocalDir * 5.0 + vec3(uSeed * 37.3, uTime * 0.16, uSeed * 53.1);
  float gran1 = fbm(gPos1, 4) * 0.5 + 0.5;
  // Fine granulation (individual granules, faster drift)
  vec3  gPos2 = vLocalDir * 28.0 + vec3(uSeed * 83.7, -uTime * 0.34, uSeed * 17.3);
  float gran2 = fbm(gPos2, 3) * 0.5 + 0.5;
  // Sharpen: bimodal distribution creates crisp bright/dark cell boundaries
  float gran  = smoothstep(0.35, 0.65, gran1 * 0.55 + gran2 * 0.45);

  // ── Sunspots — correct probability: uSpotDensity fraction of surface ─────────
  vec3  sPos   = vLocalDir * 4.5 + vec3(-uTime * 0.055, uSeed * 29.3, uSeed * 0.7);
  float sNoise = fbm(sPos, 3) * 0.5 + 0.5;  // [0,1]

  // threshold = uSpotDensity: ~uSpotDensity fraction of surface below threshold
  float threshold = uSpotDensity;
  float spots     = smoothstep(threshold - 0.04, threshold + 0.06, sNoise);
  // spots ≈ 0 → sunspot umbra, spots ≈ 1 → normal surface

  // Faculae: bright halos just outside sunspot boundary
  float faculae = smoothstep(threshold + 0.04, threshold + 0.20, sNoise) * uSpotDensity * 0.35;

  // ── Color mixing ─────────────────────────────────────────────────────────────
  // Granulation: dark intergranular lanes → bright cell tops
  vec3 surfaceC = mix(uBase * 0.72, uHighlight * 1.15, gran);
  // Sunspots pull toward dark umbra color
  surfaceC = mix(uDark, surfaceC, spots);
  // Faculae: additive bright ring at spot perimeter
  surfaceC += uHighlight * faculae;

  // Limb darkening applied to full surface color
  vec3 col = surfaceC * limb;

  // ── Fresnel corona (chromospheric limb glow) ─────────────────────────────────
  float rim  = 1.0 - mu;
  float halo = smoothstep(0.52, 1.0, rim);
  col += uHighlight * halo * 0.60;

  gl_FragColor = vec4(col * uLuminosity, 1.0);
}
`

// ─────────────────────────────────────────────────────────────────────────────
// V1 — IQ Gradient Noise + Domain Warping
// Smoother, more organic convection cells with flowing warp distortion.
// ─────────────────────────────────────────────────────────────────────────────

const STAR_FRAGMENT_V1 = /* glsl */`
${GRADIENT_NOISE_GLSL}

uniform vec3  uBase;
uniform vec3  uHighlight;
uniform vec3  uDark;
uniform float uSeed;
uniform float uLimbDark;
uniform float uSpotDensity;
uniform int   uTypeCode;
uniform float uTime;
uniform float uLuminosity;

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));

${COMPACT_OBJECTS_GLSL}

  float limb = 1.0 - uLimbDark*(1.0-mu);

  // ── Domain-warped gradient noise granulation ─────────────────────────────────
  // Coarse warp field (organic distortion of sample coordinates)
  vec3 wp   = vLocalDir*3.5 + vec3(uSeed*41.3, uTime*0.080, uSeed*67.1);
  vec3 warp = vec3(gfbm(wp,3), gfbm(wp+vec3(5.2,1.3,2.7),3), 0.0)*0.40;

  // Supergranulation — warped sample
  vec3 gPos1 = vLocalDir*5.0 + vec3(uSeed*37.3, uTime*0.14, uSeed*53.1) + warp;
  float gran1 = gfbm(gPos1, 4)*0.5+0.5;

  // Fine granulation — faster drift, no warp
  vec3 gPos2 = vLocalDir*28.0 + vec3(uSeed*83.7, -uTime*0.31, uSeed*17.3);
  float gran2 = gfbm(gPos2, 3)*0.5+0.5;

  // Sharpen: crisp bright cell / dark lane boundaries
  float gran = smoothstep(0.35, 0.65, gran1*0.55 + gran2*0.45);

  // ── Sunspots — correct probability (gradient noise) ──────────────────────────
  vec3  sPos    = vLocalDir*4.5 + vec3(-uTime*0.050, uSeed*29.3, uSeed*0.7);
  float sNoise  = gfbm(sPos, 3)*0.5+0.5;
  float thresh  = uSpotDensity;
  float spots   = smoothstep(thresh-0.04, thresh+0.06, sNoise);
  float faculae = smoothstep(thresh+0.04, thresh+0.20, sNoise)*uSpotDensity*0.35;

  vec3 surfaceC = mix(uBase*0.72, uHighlight*1.15, gran);
  surfaceC = mix(uDark, surfaceC, spots);
  surfaceC += uHighlight*faculae;

  vec3 col = surfaceC*limb;

  // Fresnel corona
  float rim  = 1.0-mu;
  float halo = smoothstep(0.52, 1.0, rim);
  col += uHighlight*halo*0.60;

  gl_FragColor = vec4(col * uLuminosity, 1.0);
}
`

// ─────────────────────────────────────────────────────────────────────────────
// V2 — Sinusoidal / Plasma (cosNoise + non-linear turbulent FBM)
// Spiky hot-cell topology, temperature-gradient coloring, strong corona.
// ─────────────────────────────────────────────────────────────────────────────

const STAR_FRAGMENT_V2 = /* glsl */`
${COSNOISE_GLSL}

uniform vec3  uBase;
uniform vec3  uHighlight;
uniform vec3  uDark;
uniform float uSeed;
uniform float uLimbDark;
uniform float uSpotDensity;
uniform int   uTypeCode;
uniform float uTime;
uniform float uLuminosity;

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));

${COMPACT_OBJECTS_GLSL}

  float limb = 1.0 - uLimbDark*(1.0-mu);

  // ── Two-scale sinusoidal turbulence (bounded frequency, no aliasing) ─────────
  // Supergranulation scale
  vec3 p1 = vLocalDir*7.0 + vec3(uSeed*37.3, uTime*0.11, uSeed*53.1);
  float n1 = cosNoise3(p1)*0.50
           + cosNoise3(p1*2.1 + vec3(1.3, 2.7, 0.9))*0.30
           + cosNoise3(p1*4.2 + vec3(3.1, 0.7, 4.1))*0.20;  // max freq = 7*4.2 = 29

  // Fine granulation scale
  vec3 p2 = vLocalDir*22.0 + vec3(uSeed*83.7, -uTime*0.26, uSeed*17.3);
  float n2 = cosNoise3(p2)*0.65
           + cosNoise3(p2*1.9 + vec3(2.3, 4.1, 1.7))*0.35;  // max freq = 22*1.9 = 42

  float h = n1*0.55 + n2*0.45;  // roughly [-1, 1]

  // Sharpen: bimodal → crisp bright spikes / dark lanes
  float t = smoothstep(-0.28, 0.32, h);

  // Temperature gradient: dark intergranular lanes → base → blazing hot peaks
  vec3 coolC = uBase * 0.16;
  vec3 hotC  = mix(uHighlight * 1.5, vec3(1.0, 0.96, 0.85), 0.42);
  vec3 surfC = mix(coolC, hotC, t);

  vec3 col = surfC*limb;

  // Strong Fresnel corona for plasma look
  float rim  = 1.0-mu;
  float halo = smoothstep(0.48, 1.0, rim);
  col += uHighlight*halo*0.80;

  gl_FragColor = vec4(col * uLuminosity, 1.0);
}
`

// ─────────────────────────────────────────────────────────────────────────────
// V3 — 2D Equal-Area Projection + Four-Stop Palette (close reimplementation)
// Technique: Lambert azimuthal projection, lattice value noise, nested-mix
// palette, polar-coordinate flare layer. Mathematical techniques only — own code.
// ─────────────────────────────────────────────────────────────────────────────

const STAR_FRAGMENT_V3 = /* glsl */`
${NOISE2D_GLSL}

#define G_PI 3.14159265

uniform vec3  uBase;
uniform vec3  uHighlight;
uniform vec3  uDark;
uniform float uSeed;
uniform float uLimbDark;
uniform float uSpotDensity;
uniform int   uTypeCode;
uniform float uTime;
uniform float uLuminosity;

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

// Four-stop gradient: dark → base → highlight → hot-white
// Technique: nested mix with per-segment clamp (standard linear math)
vec3 sunPal(float r) {
  float t0=clamp(r*3.0,    0.0,1.0);
  float t1=clamp(r*3.0-1.0,0.0,1.0);
  float t2=clamp(r*3.0-2.0,0.0,1.0);
  return mix(uDark, mix(uBase, mix(uHighlight, vec3(1.0,0.97,0.90), t2), t1), t0);
}

void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));

${COMPACT_OBJECTS_GLSL}

  float limb = 1.0 - uLimbDark*(1.0-mu);

  // Lambert azimuthal projection: vLocalDir.y=1 (north pole) → disk center r=0
  float polar = acos(clamp(vLocalDir.y,-1.0,1.0));
  float xzLen = length(vLocalDir.xz);
  vec2  dkDir = xzLen > 0.001 ? vLocalDir.xz/xzLen : vec2(1.0,0.0);
  vec2  ws    = dkDir*(polar/(G_PI*0.5));  // r=1 at equator

  // ── Surface granulation ──────────────────────────────────────────────────────
  vec2 wsSurf = ws;
  wsSurf.x += uSeed*0.17 + uTime*0.20;
  wsSurf.y += uSeed*0.23;
  float r = smoothstep(0.40, 1.0, fbm2D(wsSurf*20.0));
  vec3 col = sunPal(r)*limb;

  // ── Polar-coordinate flare layer (azimuthal × radial FBM) ───────────────────
  float l = sin(polar);   // 0 at axis poles, 1 at equator
  float a = atan(vLocalDir.z, vLocalDir.x)/(2.0*G_PI);  // azimuthal [-0.5, 0.5]
  vec2 wsF = vec2(a + l*sin(uTime*0.40)*0.018, l*0.01);
  wsF.y -= uTime*0.040 + uSeed*0.05;
  float rF = smoothstep(0.60, 0.0, fbm2D(wsF*20.0)*smoothstep(0.2,0.9,l)*1.25);
  col = mix(col, sunPal(rF)*limb, pow(smoothstep(0.1,0.6,l),4.0)*rF);

  // ── Outer flare accent ───────────────────────────────────────────────────────
  float rO = smoothstep(0.90, 0.0, fbm2D((wsF+vec2(0.25,0.0))*5.0)*smoothstep(0.4,0.6,l*1.1));
  col = mix(col, sunPal(rO)*limb, pow(smoothstep(0.0,0.8,l*1.2),0.2)*rO*0.10);

  gl_FragColor = vec4(col * uLuminosity, 1.0);
}
`

// ─────────────────────────────────────────────────────────────────────────────
// V4 — Derived: V3 + Domain Warping + Spectral Palette Adaptation
// Extension: warp field displaces granulation samples → plasma river patterns.
// Spectral-adapted hot-end: palette terminus shifts with star's highlight color.
// ─────────────────────────────────────────────────────────────────────────────

const STAR_FRAGMENT_V4 = /* glsl */`
${NOISE2D_GLSL}

#define G_PI 3.14159265

uniform vec3  uBase;
uniform vec3  uHighlight;
uniform vec3  uDark;
uniform float uSeed;
uniform float uLimbDark;
uniform float uSpotDensity;
uniform int   uTypeCode;
uniform float uTime;
uniform float uLuminosity;

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

vec3 sunPal(float r) {
  float t0=clamp(r*3.0,    0.0,1.0);
  float t1=clamp(r*3.0-1.0,0.0,1.0);
  float t2=clamp(r*3.0-2.0,0.0,1.0);
  // Spectral adaptation: hot terminus blends toward star's own highlight hue
  vec3 hotWhite = mix(vec3(1.0,0.97,0.90), uHighlight*1.9, 0.22);
  return mix(uDark, mix(uBase, mix(uHighlight, hotWhite, t2), t1), t0);
}

void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));

${COMPACT_OBJECTS_GLSL}

  float limb = 1.0 - uLimbDark*(1.0-mu);

  float polar = acos(clamp(vLocalDir.y,-1.0,1.0));
  float xzLen = length(vLocalDir.xz);
  vec2  dkDir = xzLen > 0.001 ? vLocalDir.xz/xzLen : vec2(1.0,0.0);
  vec2  ws    = dkDir*(polar/(G_PI*0.5));

  // ── Domain-warped granulation — flowing plasma rivers ────────────────────────
  vec2 wsSurf = ws;
  wsSurf.x += uSeed*0.17 + uTime*0.18;
  wsSurf.y += uSeed*0.23;
  // Two-axis warp field from low-frequency FBM (bounded, no aliasing)
  vec2 warp = vec2(
    fbm2D(wsSurf*7.0+vec2(uSeed*0.31,0.0))*2.0-1.0,
    fbm2D(wsSurf*7.0+vec2(0.0,uSeed*0.23))*2.0-1.0
  )*0.28;
  float r = smoothstep(0.38, 1.0, fbm2D((wsSurf+warp)*20.0));
  vec3 col = sunPal(r)*limb;

  // ── Polar flare layer with warp interaction at limb ──────────────────────────
  float l = sin(polar);
  float a = atan(vLocalDir.z, vLocalDir.x)/(2.0*G_PI);
  vec2 wsF = vec2(a + l*sin(uTime*0.35)*0.022, l*0.012);
  wsF.y -= uTime*0.045 + uSeed*0.05;
  vec2 flareWarp = warp*0.10*smoothstep(0.4,0.9,l);
  float rF = smoothstep(0.58, 0.0, fbm2D((wsF+flareWarp)*20.0)*smoothstep(0.2,0.9,l)*1.30);
  col = mix(col, sunPal(rF)*limb, pow(smoothstep(0.1,0.6,l),4.0)*rF);

  // ── Outer flare accent ───────────────────────────────────────────────────────
  float rO = smoothstep(0.90, 0.0, fbm2D((wsF+vec2(0.25,0.0))*5.0)*smoothstep(0.4,0.6,l*1.1));
  col = mix(col, sunPal(rO)*limb, pow(smoothstep(0.0,0.8,l*1.2),0.2)*rO*0.10);

  gl_FragColor = vec4(col * uLuminosity, 1.0);
}
`

// ─────────────────────────────────────────────────────────────────────────────
// Prominence / Flare shell — separate transparent sphere over the star
// Only for normal stars (not BH/Pulsar).
// ─────────────────────────────────────────────────────────────────────────────

// V0/V1 prominence (blob FBM)
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

// V2 prominence — anisotropic gradient noise, sharp arch-shaped filaments
const PROMINENCE_FRAGMENT_V2 = /* glsl */`
${GRADIENT_NOISE_GLSL}

uniform vec3  uColor;
uniform float uSeed;
uniform float uTime;

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

void main() {
  vec3  viewDir  = normalize(cameraPosition - vWorldPos);
  float mu       = max(0.0, dot(vWorldNormal, viewDir));
  float limb     = 1.0-mu;
  float limbMask = smoothstep(0.15, 0.90, limb);

  // Anisotropic stretch: tall & narrow → arch filaments
  vec3 aDir = vLocalDir * vec3(2.0, 9.0, 2.0);

  vec3 f1 = aDir + vec3(uTime*0.045, uSeed*23.1, -uTime*0.028);
  float n1 = gfbm(f1, 4)*0.5+0.5;

  vec3 f2 = aDir*1.7 + vec3(-uTime*0.068, uSeed*41.7, uTime*0.039);
  float n2 = gfbm(f2, 3)*0.5+0.5;

  float flame = n1*0.65 + n2*0.35;

  // Hard cutoff — only sharp filament peaks visible (no blob)
  float alpha = smoothstep(0.70, 0.86, flame)*limbMask*0.88;

  vec3 col = uColor*(3.0 + n1*1.5);
  gl_FragColor = vec4(col, alpha);
}
`

// V3/V4 prominence — polar coordinates, natural arch-shaped flares (shared)
const PROMINENCE_FRAGMENT_V3 = /* glsl */`
${NOISE2D_GLSL}

#define G_PI 3.14159265

uniform vec3  uColor;
uniform float uSeed;
uniform float uTime;

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));

  float polar   = acos(clamp(vLocalDir.y,-1.0,1.0));
  float l       = sin(polar);   // radial extent: 0 at axis, 1 at equator
  float a       = atan(vLocalDir.z, vLocalDir.x)/(2.0*G_PI);

  // Polar FBM: (azimuthal, radial) → natural arch/flare structures
  vec2 wsF = vec2(a + l*0.30 + uTime*0.090, l*0.40);
  wsF.y   -= uTime*0.035 + uSeed*0.04;
  // Forward threshold: only show bright FBM peaks (filament tips) → sparse arches
  float fbmVal = fbm2D(wsF*16.0)*smoothstep(0.28,0.85,l);
  float rF     = smoothstep(0.68, 1.0, fbmVal);

  float limb     = 1.0-mu;
  // Tight limb mask: only show very close to the stellar edge
  float limbMask = smoothstep(0.50, 0.96, limb);
  float alpha    = pow(smoothstep(0.30,0.85,l),2.0)*rF*limbMask;

  vec3 col = uColor*(2.8 + rF*1.8);
  gl_FragColor = vec4(col, alpha);
}
`

// ─────────────────────────────────────────────────────────────────────────────
// V5 — Temporal Voronoi Granulation (Power Diagram)
// Growing convection cells; dark intergranular lanes from F1≈F2 competition.
// Own implementation: weighted Voronoi / power diagram, temporal lifecycle.
// ─────────────────────────────────────────────────────────────────────────────

const STAR_FRAGMENT_V5 = /* glsl */`
uniform vec3  uBase;
uniform vec3  uHighlight;
uniform vec3  uDark;
uniform float uSeed;
uniform float uLimbDark;
uniform float uSpotDensity;
uniform int   uTypeCode;
uniform float uTime;
uniform float uLuminosity;
uniform float uV5Scale;      // grid cells across hemisphere (default 28)
uniform float uV5Lifetime;   // full cell cycle in seconds (default 12)
uniform float uV5RiseTime;   // growth phase duration in seconds (default 7)
uniform float uV5MaxRadius;  // max cell radius in grid units (default 0.72)
uniform float uV5LaneWidth;  // dark lane width between competing cells (default 0.25)

// Deterministic cell data from grid ID: uniforms must be declared first
vec3 _vcd(vec3 id) {
  float n = dot(id, vec3(1.0, 57.0, 113.0)) + uSeed * 17.3;
  return vec3(fract(sin(n      )*43758.545),
              fract(sin(n+1.31 )*43758.545),
              fract(sin(n+2.74 )*43758.545));
}
// Smooth lifecycle: 0 → peak over tRise seconds, peak → 0 by tLife
float cellLife(float age, float tRise, float tLife) {
  return smoothstep(0.0, tRise, age) * (1.0 - smoothstep(tRise, tLife, age));
}

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));

${COMPACT_OBJECTS_GLSL}

  float limb = 1.0 - uLimbDark * (1.0 - mu);

  // 3D grid on sphere surface
  vec3 sp = vLocalDir * uV5Scale + uSeed * 37.3;
  vec3 ip  = floor(sp);
  vec3 fp  = fract(sp);

  // F1 = nearest, F2 = second nearest effective distance (power diagram)
  float F1 = 1e9, F2 = 1e9;
  float bright1 = 0.0, age1 = 0.0;

  for (int kz = -1; kz <= 1; kz++)
  for (int ky = -1; ky <= 1; ky++)
  for (int kx = -1; kx <= 1; kx++) {
    vec3 nb  = vec3(float(kx), float(ky), float(kz));
    vec3 cd  = _vcd(ip + nb);
    vec3 ctr = nb + cd;

    // Birth offsets spread over 1.67× cycle → smooth staggered rebirths
    float age    = mod(uTime + cd.z * uV5Lifetime * 1.67, uV5Lifetime);
    float life   = cellLife(age, uV5RiseTime, uV5Lifetime);
    float radius = life * uV5MaxRadius;

    float d    = length(fp - ctr);
    // Effective (normalised) distance → Power Diagram: larger cells win more area
    float dEff = (radius > 0.02) ? d / radius : 1e9;

    if (dEff < F1) { F2 = F1; F1 = dEff; bright1 = life; age1 = age; }
    else if (dEff < F2) { F2 = dEff; }
  }

  // ── Core glow: hot at centre (small F1), fades toward boundary ───────────────
  float coreGlow = bright1 * max(0.0, 1.0 - F1 * 1.1);
  float heatAge  = 1.0 - smoothstep(0.0, uV5RiseTime, age1) * 0.30;

  // ── Intergranular lane: dark where F1 ≈ F2 (equal competition) ───────────────
  float laneContest = 1.0 - smoothstep(0.0, uV5LaneWidth, F2 - F1);
  float laneStr     = laneContest * smoothstep(0.10, 0.40, bright1);

  // ── Full-coverage color: dead cells fade to dim base, never black ─────────────
  vec3 surfC = mix(uBase * 0.72, uHighlight * 1.12, coreGlow * heatAge);
  surfC = mix(uDark * 0.45, surfC, 1.0 - laneStr * 0.88);
  // Guarantee coverage: even dead/newborn cells show at least 35% base influence
  surfC = mix(uBase * 0.55, surfC, clamp(bright1 * 2.0, 0.35, 1.0));

  vec3 col = surfC * limb;

  // Fresnel corona
  float rim  = 1.0 - mu;
  float halo = smoothstep(0.52, 1.0, rim);
  col += uHighlight * halo * 0.55;

  gl_FragColor = vec4(col * uLuminosity, 1.0);
}
`

export function createStarProminenceMaterial(star: Star, variant: 0|1|2|3|4|5 = 0, colorOverride?: string): THREE.ShaderMaterial {
  const { base } = starColorTriad(colorOverride ?? star.color_hex)
  return new THREE.ShaderMaterial({
    uniforms: {
      uColor: { value: base },
      uSeed:  { value: uuidSeed(star.id) + 0.5 },
      uTime:  { value: 0 },
    },
    vertexShader:   BODY_VERTEX,
    fragmentShader: variant === 5 ? PROMINENCE_FRAGMENT_V2   // V5: gradient noise arches
                 : variant >= 3  ? PROMINENCE_FRAGMENT_V3
                 : variant === 2 ? PROMINENCE_FRAGMENT_V2
                 : PROMINENCE_FRAGMENT,
    transparent: true,
    depthWrite:  false,
    blending:    THREE.AdditiveBlending,
    side:        THREE.FrontSide,
  })
}

export function createStarMaterial(star: Star, variant: 0|1|2|3|4|5 = 0, colorOverride?: string): THREE.ShaderMaterial {
  const { base, highlight, dark } = starColorTriad(colorOverride ?? star.color_hex)
  const fragmentShader = variant === 5 ? STAR_FRAGMENT_V5
                       : variant === 4 ? STAR_FRAGMENT_V4
                       : variant === 3 ? STAR_FRAGMENT_V3
                       : variant === 2 ? STAR_FRAGMENT_V2
                       : variant === 1 ? STAR_FRAGMENT_V1
                       : STAR_FRAGMENT
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
      uLuminosity:  { value: 1.0 },
      uV5Scale:     { value: 28.0 },
      uV5Lifetime:  { value: 12.0 },
      uV5RiseTime:  { value: 7.0 },
      uV5MaxRadius: { value: 0.72 },
      uV5LaneWidth: { value: 0.25 },
    },
    vertexShader:   BODY_VERTEX,
    fragmentShader,
  })
}
