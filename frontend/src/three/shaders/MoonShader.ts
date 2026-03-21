import * as THREE from 'three'
import type { Moon } from '../../types/galaxy'
import { NOISE_GLSL, BODY_VERTEX } from './noise.glsl'
import { uuidSeed } from './StarShader'

// ── Base color + crater density by composition ────────────────────────────────
const MOON_PARAMS: Record<string, { color: THREE.Vector3; craterDensity: number; roughness: number }> = {
  rocky: { color: new THREE.Vector3(0.48, 0.44, 0.40), craterDensity: 0.75, roughness: 0.70 },
  icy:   { color: new THREE.Vector3(0.82, 0.86, 0.90), craterDensity: 0.28, roughness: 0.30 },
  mixed: { color: new THREE.Vector3(0.62, 0.62, 0.60), craterDensity: 0.52, roughness: 0.50 },
}

const MOON_FRAGMENT = /* glsl */`
${NOISE_GLSL}

uniform vec3  uBaseColor;
uniform float uSeed;
uniform float uCraterDensity;  // 0.25 (icy) – 0.75 (rocky)
uniform float uRoughness;      // surface FBM amplitude

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;

void main() {
  vec3 worldNormal = vWorldNormal;
  // No atmosphere — hard shadows, very little ambient
  vec3 lightDir = normalize(-vWorldPos);
  float lighting = 0.02 + 0.98 * max(0.0, dot(worldNormal, lightDir));

  // ── Terrain ─────────────────────────────────────────────────────────────────
  vec3 tPos    = vLocalDir * 2.2 + vec3(uSeed*53.3, uSeed*37.7, uSeed*71.1);
  float terrain = fbm(tPos, 5) * 0.5 + 0.5;

  // ── Crater field ─────────────────────────────────────────────────────────────
  // Two-scale crater system: large ancient + smaller fresh craters
  vec3 cPos1  = vLocalDir * 3.5 + vec3(uSeed*17.1, uSeed*43.9, uSeed*29.3);
  float cN1   = fbm(cPos1, 3);
  // Crater rim: bright ring at |noise| ≈ threshold
  float rim1  = smoothstep(0.06, 0.14, abs(cN1) - 0.02)
              * (1.0 - smoothstep(0.14, 0.26, abs(cN1) - 0.02));
  // Crater floor: slightly darker
  float floor1 = step(abs(cN1), 0.10);

  vec3 cPos2  = vLocalDir * 7.5 + vec3(uSeed*83.1, uSeed*61.7, uSeed*37.1);
  float cN2   = fbm(cPos2, 3);
  float rim2  = smoothstep(0.08, 0.16, abs(cN2) - 0.02)
              * (1.0 - smoothstep(0.16, 0.28, abs(cN2) - 0.02));
  float floor2 = step(abs(cN2), 0.08);

  // ── Color composition ────────────────────────────────────────────────────────
  // Base: terrain modulates brightness slightly
  vec3 col = uBaseColor * (0.72 + terrain * uRoughness * 0.45);

  // Apply craters (scaled by density)
  float craterEffect = uCraterDensity;
  col = mix(col, uBaseColor * 0.52, floor1  * craterEffect * 0.55);
  col = mix(col, uBaseColor * 1.18, rim1    * craterEffect * 0.45);
  col = mix(col, uBaseColor * 0.58, floor2  * craterEffect * 0.45);
  col = mix(col, uBaseColor * 1.12, rim2    * craterEffect * 0.35);

  // Icy surface: add subtle blue-white sheen in high terrain (frost)
  if (uCraterDensity < 0.4) {
    float frost = smoothstep(0.62, 0.80, terrain) * (1.0 - uCraterDensity * 2.0);
    col = mix(col, vec3(0.90, 0.93, 0.97), frost * 0.35);
  }

  gl_FragColor = vec4(col * lighting, 1.0);
}
`

export function createMoonMaterial(moon: Moon): THREE.ShaderMaterial {
  const p = MOON_PARAMS[moon.composition_type] ?? MOON_PARAMS.rocky
  return new THREE.ShaderMaterial({
    uniforms: {
      uBaseColor:     { value: p.color.clone() },
      uSeed:          { value: uuidSeed(moon.id) },
      uCraterDensity: { value: p.craterDensity },
      uRoughness:     { value: p.roughness },
    },
    vertexShader:   BODY_VERTEX,
    fragmentShader: MOON_FRAGMENT,
  })
}
