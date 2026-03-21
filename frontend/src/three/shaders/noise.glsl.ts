// Hash-based value noise — no external dependencies, GLSL ES 2.0 safe.
// Mathematical technique from: Inigo Quilez, "Value Noise" (iquilezles.org, public domain algorithms)
// Implementation written from scratch.

export const NOISE_GLSL = /* glsl */`

// ── Hash helpers ──────────────────────────────────────────────────────────────
float _h1(float n) { return fract(sin(n) * 43758.5453123); }

float _h3(vec3 p) {
  p = fract(p * vec3(127.1, 311.7, 74.7));
  p += dot(p, p.yzx + 19.19);
  return fract((p.x + p.y) * p.z);
}

// ── Trilinear value noise → [-1, 1] ──────────────────────────────────────────
float snoise(vec3 p) {
  vec3 i = floor(p);
  vec3 f = fract(p);
  vec3 u = f * f * (3.0 - 2.0 * f);   // smoothstep curve

  float n000 = _h3(i + vec3(0.0, 0.0, 0.0));
  float n100 = _h3(i + vec3(1.0, 0.0, 0.0));
  float n010 = _h3(i + vec3(0.0, 1.0, 0.0));
  float n110 = _h3(i + vec3(1.0, 1.0, 0.0));
  float n001 = _h3(i + vec3(0.0, 0.0, 1.0));
  float n101 = _h3(i + vec3(1.0, 0.0, 1.0));
  float n011 = _h3(i + vec3(0.0, 1.0, 1.0));
  float n111 = _h3(i + vec3(1.0, 1.0, 1.0));

  float nx00 = mix(n000, n100, u.x);
  float nx10 = mix(n010, n110, u.x);
  float nx01 = mix(n001, n101, u.x);
  float nx11 = mix(n011, n111, u.x);
  float nxy0 = mix(nx00, nx10, u.y);
  float nxy1 = mix(nx01, nx11, u.y);
  return mix(nxy0, nxy1, u.z) * 2.0 - 1.0;
}

// ── FBM — unrolled to 6 octaves, GLSL ES 2.0 safe ────────────────────────────
// int parameter selects depth; unrolling avoids non-constant loop issues.
float fbm(vec3 p, int n) {
  float v = 0.0;
  float a = 0.5;
  if (n > 0) { v += a * snoise(p); p *= 2.01; a *= 0.5; }
  if (n > 1) { v += a * snoise(p); p *= 2.01; a *= 0.5; }
  if (n > 2) { v += a * snoise(p); p *= 2.01; a *= 0.5; }
  if (n > 3) { v += a * snoise(p); p *= 2.01; a *= 0.5; }
  if (n > 4) { v += a * snoise(p); p *= 2.01; a *= 0.5; }
  if (n > 5) { v += a * snoise(p); p *= 2.01; a *= 0.5; }
  return v;
}
`

// ── Shared vertex shader ───────────────────────────────────────────────────────
// vLocalDir: unit-sphere direction — used as noise input (no UV seam)
// vWorldPos: world-space position — used for star-lighting direction
// vNormal:   model-space normal   — used to compute world normal in fragment
// vUv:       texture UV           — used for latitude bands

export const BODY_VERTEX = /* glsl */`
varying vec3 vNormal;
varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;
varying vec2 vUv;

void main() {
  vNormal      = normal;
  vWorldNormal = normalize(mat3(modelMatrix) * normal);
  vLocalDir    = normalize(position);
  vWorldPos    = (modelMatrix * vec4(position, 1.0)).xyz;
  vUv          = uv;
  gl_Position  = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`
