import * as THREE from 'three'
import type { Planet } from '../../types/galaxy'
import { NOISE_GLSL, BODY_VERTEX } from './noise.glsl'
import { uuidSeed } from './StarShader'

// ── Biochem archetype → integer for GLSL branch ───────────────────────────────
const BIOCHEM_IDX: Record<string, number> = {
  terran: 0, thermophilic: 1, anaerobic: 2, cryophilic: 3, chlorine: 4,
}

// ── Gas-giant color palettes (warm/cool/cold based on equilibrium temp) ───────
function gasGiantColors(tempK: number): {
  c1: THREE.Vector3; c2: THREE.Vector3; storm: THREE.Vector3; bandFreq: number
} {
  if (tempK > 500) {
    // Hot: Jupiter-like — brown/orange/cream
    return {
      c1:       new THREE.Vector3(0.62, 0.42, 0.22),
      c2:       new THREE.Vector3(0.85, 0.72, 0.50),
      storm:    new THREE.Vector3(0.78, 0.35, 0.18),
      bandFreq: 7.0,
    }
  } else if (tempK > 200) {
    // Warm: Saturn-like — cream/tan/pale yellow
    return {
      c1:       new THREE.Vector3(0.75, 0.68, 0.50),
      c2:       new THREE.Vector3(0.88, 0.82, 0.65),
      storm:    new THREE.Vector3(0.65, 0.55, 0.40),
      bandFreq: 5.0,
    }
  } else {
    // Cold: ice-blue/gray
    return {
      c1:       new THREE.Vector3(0.55, 0.65, 0.75),
      c2:       new THREE.Vector3(0.72, 0.80, 0.88),
      storm:    new THREE.Vector3(0.40, 0.55, 0.70),
      bandFreq: 4.0,
    }
  }
}

// ── Ice-giant color by equilibrium temperature ────────────────────────────────
function iceGiantColor(tempK: number): THREE.Vector3 {
  if (tempK > 120) return new THREE.Vector3(0.25, 0.55, 0.80)  // Uranus-teal
  return new THREE.Vector3(0.18, 0.35, 0.72)                   // Neptune-blue
}

// ─────────────────────────────────────────────────────────────────────────────
// FRAGMENT SHADERS
// ─────────────────────────────────────────────────────────────────────────────

const GAS_GIANT_FRAGMENT = /* glsl */`
${NOISE_GLSL}

uniform vec3  uC1;          // primary band color
uniform vec3  uC2;          // secondary band color
uniform vec3  uStorm;       // storm oval color
uniform float uSeed;
uniform float uBandFreq;    // band count (latitude wraps)
uniform float uStormLat;    // storm latitude normalized [-1, 1]
uniform float uStormLon;    // storm longitude [0, 2π]
uniform float uStormSize;   // storm radius in UV space

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;
varying vec2 vUv;

void main() {
  // Diffuse lighting from star at world origin
  vec3 worldNormal = vWorldNormal;
  vec3 lightDir    = normalize(-vWorldPos);
  float lighting   = 0.06 + 0.94 * max(0.0, dot(worldNormal, lightDir));

  // Banded atmosphere: sin in latitude + FBM distortion
  vec3 bPos     = vLocalDir * 1.5 + vec3(uSeed*41.7, 0.0, uSeed*19.3);
  float distort = fbm(bPos, 4) * 0.30;
  float lat     = vUv.y + distort;
  float bands   = sin(lat * uBandFreq * 3.14159) * 0.5 + 0.5;

  // Sub-bands (finer banding texture)
  float subD   = fbm(vLocalDir * 3.0 + uSeed * 7.1, 3) * 0.12;
  float subB   = sin((vUv.y + subD) * uBandFreq * 9.42) * 0.5 + 0.5;
  bands        = mix(bands, subB, 0.28);

  vec3 baseColor = mix(uC1, uC2, bands);

  // Storm oval — positioned by seed (persistent per planet)
  float uvLat = vUv.y * 2.0 - 1.0;
  float uvLon = vUv.x * 6.28318;
  float dLat  = uvLat - uStormLat;
  float dLon  = mod(uvLon - uStormLon + 3.14159, 6.28318) - 3.14159;
  // Oval: wider in longitude than latitude
  float stormDist = sqrt(dLat*dLat * 6.0 + dLon*dLon * 0.4);
  float storm     = 1.0 - smoothstep(uStormSize * 0.4, uStormSize, stormDist);
  // Internal swirl in storm
  float swirl = fbm(vLocalDir * 5.0 + uSeed * 3.3, 3) * 0.3;
  vec3 stormFinal = mix(uStorm, uC2, swirl);

  vec3 color = mix(baseColor, stormFinal, storm * 0.65);
  gl_FragColor = vec4(color * lighting, 1.0);
}
`

const ICE_GIANT_FRAGMENT = /* glsl */`
${NOISE_GLSL}

uniform vec3  uColor;
uniform float uSeed;

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;
varying vec2 vUv;

void main() {
  vec3 worldNormal = vWorldNormal;
  vec3 lightDir    = normalize(-vWorldPos);
  float lighting   = 0.06 + 0.94 * max(0.0, dot(worldNormal, lightDir));

  // Subtle horizontal bands
  vec3 bPos  = vLocalDir * 1.2 + vec3(uSeed*31.7, 0.0, uSeed*17.3);
  float band = fbm(bPos, 3);
  float stripe = sin(vUv.y * 8.0 * 3.14159 + band * 1.8) * 0.5 + 0.5;

  // Surface variation
  float surf = fbm(vLocalDir * 2.5 + uSeed * 5.5, 4) * 0.06;

  vec3 c2    = uColor * vec3(0.78, 1.08, 1.02); // cyan shift for secondary band
  vec3 baseC = mix(uColor, c2, stripe * 0.45);

  // Atmosphere limb glow
  vec3 viewDir = normalize(cameraPosition - vWorldPos);
  float limb   = 1.0 - max(0.0, dot(worldNormal, viewDir));
  float haze   = smoothstep(0.72, 1.0, limb) * 0.45;

  vec3 col = (baseC + surf) * lighting;
  col = mix(col, uColor * 1.6, haze);
  gl_FragColor = vec4(col, 1.0);
}
`

const ROCKY_PLANET_FRAGMENT = /* glsl */`
${NOISE_GLSL}

// 0=terran 1=thermophilic 2=anaerobic 3=cryophilic 4=chlorine
uniform int   uBiochem;
uniform float uSeed;
uniform float uSurfaceTemp;   // K
uniform float uAtmPressure;   // atm

varying vec3 vWorldNormal;
varying vec3 vLocalDir;
varying vec3 vWorldPos;
varying vec2 vUv;

void main() {
  vec3 worldNormal = vWorldNormal;
  vec3 lightDir    = normalize(-vWorldPos);
  float lighting   = 0.06 + 0.94 * max(0.0, dot(worldNormal, lightDir));

  // Terrain heightmap (6 octaves of FBM for continental-scale detail)
  vec3 tPos  = vLocalDir * 2.5 + vec3(uSeed*47.3, uSeed*23.7, uSeed*67.1);
  float elev = fbm(tPos, 6) * 0.5 + 0.5; // [0, 1]

  // Latitude: 0 = equator, 1 = pole
  float latitude = abs(vUv.y * 2.0 - 1.0);

  // ── Polar ice caps ──────────────────────────────────────────────────────────
  // Ice cap radius: full at 200K, shrinks to zero at 330K
  float iceRadius = clamp(1.0 - (uSurfaceTemp - 200.0) / 130.0, 0.0, 1.0);
  float iceBlend  = smoothstep(iceRadius - 0.08, iceRadius + 0.08, latitude);

  // ── Ocean / dry surface ─────────────────────────────────────────────────────
  // Liquid water requires 273–350 K and atmosphere
  float hasWater = step(273.0, uSurfaceTemp) * step(uSurfaceTemp, 350.0)
                 * clamp(uAtmPressure, 0.0, 1.0);
  float isOcean  = step(elev, 0.43) * hasWater;

  // ── Biochem color palettes ──────────────────────────────────────────────────
  vec3 oceanC, lowC, midC, highC, snowC;

  if (uBiochem == 0) {           // Terran — Earth-like
    oceanC = vec3(0.07, 0.24, 0.53);
    lowC   = vec3(0.18, 0.44, 0.14);  // vegetation green
    midC   = vec3(0.50, 0.37, 0.19);  // soil/rock
    highC  = vec3(0.52, 0.48, 0.43);  // high rock
    snowC  = vec3(0.92, 0.93, 0.96);
  } else if (uBiochem == 1) {    // Thermophilic — hot desert
    oceanC = vec3(0.44, 0.34, 0.18);  // baked sand
    lowC   = vec3(0.68, 0.44, 0.14);  // orange dust
    midC   = vec3(0.54, 0.34, 0.11);  // dark rock
    highC  = vec3(0.38, 0.29, 0.21);  // dark summit
    snowC  = vec3(0.83, 0.76, 0.62);  // yellowish, not ice
  } else if (uBiochem == 2) {    // Anaerobic — no O₂, purple microbes
    oceanC = vec3(0.14, 0.20, 0.28);
    lowC   = vec3(0.24, 0.34, 0.14);  // dark green mats
    midC   = vec3(0.34, 0.19, 0.34);  // purple/mauve
    highC  = vec3(0.38, 0.34, 0.28);
    snowC  = vec3(0.80, 0.80, 0.86);
  } else if (uBiochem == 3) {    // Cryophilic — ice world
    oceanC = vec3(0.52, 0.62, 0.79);  // frozen sea
    lowC   = vec3(0.48, 0.58, 0.74);  // ice tundra
    midC   = vec3(0.58, 0.64, 0.70);  // grey ice
    highC  = vec3(0.74, 0.79, 0.84);
    snowC  = vec3(0.90, 0.92, 0.96);
  } else {                       // Chlorine — yellow-green haze world
    oceanC = vec3(0.28, 0.38, 0.09);
    lowC   = vec3(0.53, 0.53, 0.09);
    midC   = vec3(0.48, 0.44, 0.07);
    highC  = vec3(0.43, 0.38, 0.11);
    snowC  = vec3(0.74, 0.71, 0.48);
  }

  // ── Lava world (surface temp > 700 K) ──────────────────────────────────────
  if (uSurfaceTemp > 700.0) {
    vec3 lPos  = vLocalDir * 4.0 + uSeed * 3.1;
    float lava = fbm(lPos, 5) * 0.5 + 0.5;
    vec3 lavaC = mix(vec3(0.04, 0.01, 0.01), vec3(0.92, 0.28, 0.0), lava);
    gl_FragColor = vec4(lavaC * lighting, 1.0);
    return;
  }

  // ── Terrain color blend ─────────────────────────────────────────────────────
  vec3 terrainC;
  terrainC = mix(lowC, midC, smoothstep(0.44, 0.66, elev));
  terrainC = mix(terrainC, highC, smoothstep(0.66, 0.86, elev));

  // Apply ocean
  terrainC = mix(terrainC, oceanC, isOcean);

  // Polar ice overlay
  terrainC = mix(terrainC, snowC, iceBlend);

  // Mountain snow (high elevation, non-equatorial, not already ice)
  float mtnSnow = smoothstep(0.78, 0.92, elev)
                * (1.0 - iceBlend) * (1.0 - isOcean)
                * clamp(1.0 - (uSurfaceTemp - 250.0) / 100.0, 0.0, 1.0);
  terrainC = mix(terrainC, snowC, mtnSnow);

  // ── Atmosphere limb glow ────────────────────────────────────────────────────
  vec3 viewDir = normalize(cameraPosition - vWorldPos);
  float limb   = 1.0 - max(0.0, dot(worldNormal, viewDir));
  float haze   = smoothstep(0.72, 1.0, limb) * clamp(uAtmPressure * 0.4, 0.0, 0.55);

  // Atmosphere tint by biochem
  vec3 hazeC = (uBiochem == 0 || uBiochem == 2)
    ? vec3(0.30, 0.52, 1.00)       // blue (O2 or no O2 but methane)
    : (uBiochem == 4)
    ? vec3(0.80, 0.90, 0.28)       // chlorine — yellow-green
    : vec3(0.80, 0.80, 0.90);      // neutral

  vec3 finalC = terrainC * lighting;
  finalC = mix(finalC, hazeC * 1.3, haze);
  gl_FragColor = vec4(finalC, 1.0);
}
`

// ─────────────────────────────────────────────────────────────────────────────
// Factory
// ─────────────────────────────────────────────────────────────────────────────

export function createPlanetMaterial(planet: Planet): THREE.ShaderMaterial {
  const seed  = uuidSeed(planet.id)
  const tempK = (planet.temp_eq_min_k + planet.temp_eq_max_k) / 2

  if (planet.planet_type === 'gas_giant') {
    const { c1, c2, storm, bandFreq } = gasGiantColors(tempK)
    // Storm position: derive from second part of UUID
    const seed2 = parseInt(planet.id.replace(/-/g, '').substring(8, 16), 16) / 0xFFFFFFFF
    return new THREE.ShaderMaterial({
      uniforms: {
        uC1:        { value: c1 },
        uC2:        { value: c2 },
        uStorm:     { value: storm },
        uSeed:      { value: seed },
        uBandFreq:  { value: bandFreq },
        uStormLat:  { value: seed2 * 1.2 - 0.6 },    // [-0.6, 0.6] latitude
        uStormLon:  { value: seed2 * Math.PI * 2 },
        uStormSize: { value: 0.15 + seed * 0.12 },    // storm radius
      },
      vertexShader:   BODY_VERTEX,
      fragmentShader: GAS_GIANT_FRAGMENT,
    })
  }

  if (planet.planet_type === 'ice_giant') {
    return new THREE.ShaderMaterial({
      uniforms: {
        uColor: { value: iceGiantColor(tempK) },
        uSeed:  { value: seed },
      },
      vertexShader:   BODY_VERTEX,
      fragmentShader: ICE_GIANT_FRAGMENT,
    })
  }

  // rocky (default)
  return new THREE.ShaderMaterial({
    uniforms: {
      uBiochem:     { value: BIOCHEM_IDX[planet.biochem_archetype] ?? 0 },
      uSeed:        { value: seed },
      uSurfaceTemp: { value: planet.surface_temp_k },
      uAtmPressure: { value: planet.atm_pressure_atm },
    },
    vertexShader:   BODY_VERTEX,
    fragmentShader: ROCKY_PLANET_FRAGMENT,
  })
}
