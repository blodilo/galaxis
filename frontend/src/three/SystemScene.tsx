import { useMemo, useRef } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { SmartOrbitControls } from './SmartOrbitControls'
import * as THREE from 'three'
import type { Planet, Star } from '../types/galaxy'
import { useVisualParams } from '../context/VisualParamsContext'
import { createStarMaterial, createStarProminenceMaterial, uuidSeed } from './shaders/StarShader'
import { createPlanetMaterial } from './shaders/PlanetShader'
import { createMoonMaterial } from './shaders/MoonShader'
import { BODY_VERTEX } from './shaders/noise.glsl'

// ── BL-18: Logarithmische Planetengröße ──────────────────────────────────────

const LOG_DENOM = Math.log(50 / 0.3 + 1)

export function calcPlanetVisR(radiusEarth: number, maxR: number, visMin: number, visMax: number): number {
  if (radiusEarth <= 0) return 0
  const norm = Math.log(radiusEarth / 0.3 + 1) / LOG_DENOM
  return Math.min(Math.max(visMin, norm * visMax), maxR)
}

// ── BL-12: Ellipsen-Hilfsrechnung ─────────────────────────────────────────────

export function computeOrbitPos(
  a: number, ecc: number,
  argPeriDeg: number, inclDeg: number,
  theta: number,
): THREE.Vector3 {
  const b = a * Math.sqrt(Math.max(0, 1 - ecc * ecc))
  const c = a * ecc
  const pos = new THREE.Vector3(a * Math.cos(theta) - c, 0, b * Math.sin(theta))
  pos.applyEuler(new THREE.Euler(inclDeg * Math.PI / 180, argPeriDeg * Math.PI / 180, 0))
  return pos
}

// ── OrbitRing ─────────────────────────────────────────────────────────────────

function OrbitRing({ radius, color, opacity }: { radius: number; color: string; opacity: number }) {
  const line = useMemo(() => {
    const pts: THREE.Vector3[] = []
    for (let i = 0; i <= 128; i++) {
      const a = (i / 128) * Math.PI * 2
      pts.push(new THREE.Vector3(Math.cos(a) * radius, 0, Math.sin(a) * radius))
    }
    const geo = new THREE.BufferGeometry().setFromPoints(pts)
    const mat = new THREE.LineBasicMaterial({ color, transparent: true, opacity, depthWrite: false })
    return new THREE.LineLoop(geo, mat)
  }, [radius, color, opacity])
  return <primitive object={line} />
}

// ── OrbitEllipse ──────────────────────────────────────────────────────────────

function OrbitEllipse({
  a, ecc, argPeriapsisDeg, inclinationDeg, color, opacity,
}: {
  a: number; ecc: number
  argPeriapsisDeg: number; inclinationDeg: number
  color: string; opacity: number
}) {
  const line = useMemo(() => {
    const b = a * Math.sqrt(Math.max(0, 1 - ecc * ecc))
    const c = a * ecc
    const pts: THREE.Vector3[] = []
    for (let i = 0; i <= 128; i++) {
      const theta = (i / 128) * Math.PI * 2
      pts.push(new THREE.Vector3(a * Math.cos(theta) - c, 0, b * Math.sin(theta)))
    }
    const geo = new THREE.BufferGeometry().setFromPoints(pts)
    const mat = new THREE.LineBasicMaterial({ color, transparent: true, opacity, depthWrite: false })
    return new THREE.LineLoop(geo, mat)
  }, [a, ecc, color, opacity])

  return (
    <group rotation={[inclinationDeg * Math.PI / 180, argPeriapsisDeg * Math.PI / 180, 0]}>
      <primitive object={line} />
    </group>
  )
}

// ── Asteroid Belt — Staub-Ring + InstancedMesh Felsen ────────────────────────

const ASTEROID_DUST_VERT = /* glsl */`
varying vec2 vUv;
void main() { vUv = uv; gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0); }
`
const ASTEROID_DUST_FRAG = /* glsl */`
uniform float uSeed;
varying vec2 vUv;

float _h(vec2 p) { p=fract(p*vec2(127.1,311.7)); p+=dot(p,p.yx+19.19); return fract((p.x+p.y)*p.x); }
float n2(vec2 p) {
  vec2 i=floor(p); vec2 f=fract(p); vec2 u=f*f*(3.0-2.0*f);
  return mix(mix(_h(i),_h(i+vec2(1,0)),u.x),mix(_h(i+vec2(0,1)),_h(i+vec2(1,1)),u.x),u.y);
}
void main() {
  vec2 uvc = vUv - 0.5;
  float r = length(uvc);
  float inner = smoothstep(0.455, 0.478, r);
  float outer = 1.0 - smoothstep(0.478, 0.500, r);
  float mask  = inner * outer;
  if (mask < 0.005) discard;
  float angle = atan(uvc.y, uvc.x);
  float clump = n2(vec2(angle*3.2+uSeed*7.3, r*9.0+uSeed*3.1)*4.0)*0.6
              + n2(vec2(angle*3.2+uSeed*7.3, r*9.0+uSeed*3.1)*13.0)*0.4;
  float alpha = mask * clump * 0.52;
  vec3  col   = mix(vec3(0.27,0.23,0.19), vec3(0.50,0.44,0.36), clump);
  gl_FragColor = vec4(col, alpha);
}
`

// LCG deterministischer Zufallsgenerator — Seed aus planet.id
export function makeLcg(seed: number): () => number {
  // Sicherstellen, dass s ungerade und nicht 0
  let s = (Math.floor(seed * 2147483647) | 1) >>> 0
  return () => {
    s = (Math.imul(1664525, s) + 1013904223) >>> 0
    return s / 4294967296
  }
}

function AsteroidBelt({ planet }: { planet: Planet }) {
  const orbitAU = planet.orbit_distance_au
  const innerR  = orbitAU * 0.88
  const outerR  = orbitAU * 1.12
  const width   = outerR - innerR

  // Staub-Ring (immer sichtbar, auch bei Zoom-out)
  const dustMesh = useMemo(() => {
    const geo = new THREE.RingGeometry(innerR, outerR, 128, 4)
    const mat = new THREE.ShaderMaterial({
      uniforms: { uSeed: { value: orbitAU * 17.37 } },
      vertexShader:   ASTEROID_DUST_VERT,
      fragmentShader: ASTEROID_DUST_FRAG,
      transparent: true, depthWrite: false, side: THREE.DoubleSide,
    })
    const mesh = new THREE.Mesh(geo, mat)
    mesh.rotation.x = -Math.PI / 2   // XZ-Ebene
    return mesh
  }, [innerR, outerR, orbitAU])

  // InstancedMesh Felsen — 280 Instanzen, deterministisch aus planet.id
  const rockMesh = useMemo(() => {
    const rng   = makeLcg(uuidSeed(planet.id))
    const COUNT = 280

    const geo = new THREE.IcosahedronGeometry(1, 0)
    const mat = new THREE.MeshStandardMaterial({
      color:     0x7a6a58,
      roughness: 0.92,
      metalness: 0.04,
      flatShading: true,      // Harte Kanten zwischen Flächen → Low-Poly-Fels-Look
    })

    const im    = new THREE.InstancedMesh(geo, mat, COUNT)
    const dummy = new THREE.Object3D()

    for (let i = 0; i < COUNT; i++) {
      const angle = rng() * Math.PI * 2
      const dist  = innerR + rng() * width

      dummy.position.x = Math.cos(angle) * dist
      dummy.position.z = Math.sin(angle) * dist
      // Y: Produkt zweier Uniformverteilungen → Dreieck-Verteilung (Mitte dichter)
      dummy.position.y = (rng() - 0.5) * (rng() - 0.5) * width * 0.9

      dummy.rotation.set(rng() * Math.PI, rng() * Math.PI, rng() * Math.PI)

      // Potenzgesetz: viele winzige, selten große — physikalisch korrekt
      const s = Math.max(0.0012, Math.pow(rng(), 3) * 0.009)
      dummy.scale.set(s, s * (0.72 + rng() * 0.56), s)

      dummy.updateMatrix()
      im.setMatrixAt(i, dummy.matrix)
    }
    im.instanceMatrix.needsUpdate = true
    return im
  }, [planet.id, innerR, outerR, width])

  // Inklination + Periapsis-Argument aus Planetendaten
  return (
    <group rotation={[planet.inclination_deg * Math.PI / 180, planet.arg_periapsis_deg * Math.PI / 180, 0]}>
      <primitive object={dustMesh} />
      <primitive object={rockMesh} />
    </group>
  )
}

// ── SelectionAura — Rim-Halo hinter dem Planeten ─────────────────────────────

const AURA_VERT = /* glsl */`
varying vec3 vWorldNormal;
varying vec3 vWorldPos;
void main() {
  vWorldNormal = normalize(mat3(modelMatrix) * normal);
  vWorldPos    = (modelMatrix * vec4(position, 1.0)).xyz;
  gl_Position  = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`
const AURA_FRAG = /* glsl */`
varying vec3 vWorldNormal;
varying vec3 vWorldPos;
void main() {
  vec3  viewDir = normalize(cameraPosition - vWorldPos);
  float mu      = max(0.0, dot(vWorldNormal, viewDir));
  float rim     = 1.0 - mu;
  float alpha   = smoothstep(0.12, 1.0, rim * rim) * 0.62;
  gl_FragColor  = vec4(0.22, 0.62, 1.0, alpha);
}
`
const auraMat = new THREE.ShaderMaterial({
  vertexShader:   AURA_VERT,
  fragmentShader: AURA_FRAG,
  transparent: true,
  depthWrite:  false,
})

function SelectionAura({ visR, position }: { visR: number; position: THREE.Vector3 }) {
  return (
    <mesh position={[position.x, position.y, position.z]} material={auraMat}>
      <sphereGeometry args={[visR * 1.22, 24, 12]} />
    </mesh>
  )
}

// ── AxisAnnotation — Rotationsachse + 3/4-Bogen am Nordpol (Layer) ───────────

function AxisAnnotation({ visR, position, axialTiltDeg }: {
  visR: number; position: THREE.Vector3; axialTiltDeg: number
}) {
  const { x: px, y: py, z: pz } = position
  const objects = useMemo(() => {
    const results: THREE.Object3D[] = []
    const mat = () => new THREE.LineBasicMaterial({ color: '#4af0ff', transparent: true, opacity: 0.78, depthWrite: false })

    const tiltRad = axialTiltDeg * Math.PI / 180
    const axisDir = new THREE.Vector3(Math.sin(tiltRad), Math.cos(tiltRad), 0)
    const axisLen = visR * 2.2

    // Axis line
    results.push(new THREE.Line(
      new THREE.BufferGeometry().setFromPoints([
        axisDir.clone().multiplyScalar(-axisLen),
        axisDir.clone().multiplyScalar(axisLen),
      ]),
      mat(),
    ))

    // 3/4 arc at north pole tip
    const tipPos = axisDir.clone().multiplyScalar(axisLen)
    const arcR   = visR * 0.22
    const zRef   = Math.abs(axisDir.z) < 0.9 ? new THREE.Vector3(0, 0, 1) : new THREE.Vector3(1, 0, 0)
    const perp1  = new THREE.Vector3().crossVectors(axisDir, zRef).normalize()
    const perp2  = new THREE.Vector3().crossVectors(axisDir, perp1).normalize()

    const arcPts: THREE.Vector3[] = []
    for (let i = 0; i <= 54; i++) {
      const theta = (i / 54) * 1.5 * Math.PI
      arcPts.push(tipPos.clone()
        .addScaledVector(perp1, Math.cos(theta) * arcR)
        .addScaledVector(perp2, Math.sin(theta) * arcR))
    }
    results.push(new THREE.Line(new THREE.BufferGeometry().setFromPoints(arcPts), mat()))

    // Arrowhead at arc end (theta=3π/2 → tangent direction = perp1)
    const arcEnd = arcPts[arcPts.length - 1]
    const ahSize = arcR * 0.55
    results.push(new THREE.Line(
      new THREE.BufferGeometry().setFromPoints([
        arcEnd.clone().sub(perp1.clone().multiplyScalar(ahSize)).addScaledVector(perp2,  ahSize * 0.45),
        arcEnd,
        arcEnd.clone().sub(perp1.clone().multiplyScalar(ahSize)).addScaledVector(perp2, -ahSize * 0.45),
      ]),
      mat(),
    ))

    return results
  }, [visR, axialTiltDeg])

  return (
    <group position={[px, py, pz]}>
      {objects.map((obj, i) => <primitive key={i} object={obj} />)}
    </group>
  )
}

// ── OrbitChevron — > auf der tatsächlichen Ellipse (Layer) ───────────────────

function OrbitChevron({ position, tangent, orbitNormal, size }: {
  position: THREE.Vector3
  tangent: THREE.Vector3
  orbitNormal: THREE.Vector3
  size: number
}) {
  const obj = useMemo(() => {
    const radial = new THREE.Vector3().crossVectors(tangent, orbitNormal).normalize()
    const tip    = position.clone()
    const b1     = tip.clone().sub(tangent.clone().multiplyScalar(size)).addScaledVector(radial,  size * 0.65)
    const b2     = tip.clone().sub(tangent.clone().multiplyScalar(size)).addScaledVector(radial, -size * 0.65)
    const geo    = new THREE.BufferGeometry().setFromPoints([b1, tip, b2])
    const mat    = new THREE.LineBasicMaterial({ color: '#f0c040', transparent: true, opacity: 0.9, depthWrite: false })
    return new THREE.Line(geo, mat)
  }, [position, tangent, orbitNormal, size])
  return <primitive object={obj} />
}

// ── Stern (mit Prominenz-Shell) ───────────────────────────────────────────────

function StarBody({ star, shaderVariant }: { star: Star; shaderVariant: 0|1|2|3|4|5 }) {
  const { params } = useVisualParams()
  const colorOverride   = params.spectralColors[star.star_type]
  const showProminences = params.layerProminences

  const r = useMemo(() => {
    const type = star.star_type
    if (type === 'SMBH')      return 0.6
    if (type === 'StellarBH') return 0.18
    if (type === 'Pulsar')    return 0.12
    const sr = star.radius_solar ?? 1
    return Math.min(Math.max(sr * 0.04, 0.12), 0.5)
  }, [star])

  const mat        = useMemo(() => createStarMaterial(star, shaderVariant as 0|1|2|3|4|5, colorOverride),        [star.id, shaderVariant, colorOverride])
  const promMat    = useMemo(() => createStarProminenceMaterial(star, shaderVariant as 0|1|2|3|4|5, colorOverride), [star.id, shaderVariant, colorOverride])
  const matRef     = useRef(mat)
  const promMatRef = useRef(promMat)
  matRef.current     = mat
  promMatRef.current = promMat

  const luminosityRef = useRef(params.starLuminosity)
  const animSpeedRef  = useRef(params.starAnimSpeed)
  const v5Ref = useRef({
    scale: params.v5CellScale, lifetime: params.v5Lifetime,
    rise: params.v5RiseTime, radius: params.v5MaxRadius, lane: params.v5LaneWidth,
  })
  luminosityRef.current = params.starLuminosity
  animSpeedRef.current  = params.starAnimSpeed
  v5Ref.current = {
    scale: params.v5CellScale, lifetime: params.v5Lifetime,
    rise: params.v5RiseTime, radius: params.v5MaxRadius, lane: params.v5LaneWidth,
  }

  useFrame((_, delta) => {
    const dt = delta * animSpeedRef.current
    const u = matRef.current.uniforms
    if (u.uTime)        u.uTime.value += dt
    if (u.uLuminosity)  u.uLuminosity.value  = luminosityRef.current
    if (u.uV5Scale)     u.uV5Scale.value     = v5Ref.current.scale
    if (u.uV5Lifetime)  u.uV5Lifetime.value  = v5Ref.current.lifetime
    if (u.uV5RiseTime)  u.uV5RiseTime.value  = v5Ref.current.rise
    if (u.uV5MaxRadius) u.uV5MaxRadius.value = v5Ref.current.radius
    if (u.uV5LaneWidth) u.uV5LaneWidth.value = v5Ref.current.lane
    const p = promMatRef.current.uniforms
    if (p.uTime) p.uTime.value += dt
  })

  const showProminencesLayer = showProminences && star.star_type !== 'StellarBH' && star.star_type !== 'SMBH' && star.star_type !== 'Pulsar'

  return (
    <>
      <mesh material={mat}>
        <sphereGeometry args={[r, 128, 64]} />
      </mesh>
      {showProminencesLayer && (
        <mesh material={promMat}>
          <sphereGeometry args={[r * 1.30, 96, 48]} />
        </mesh>
      )}
    </>
  )
}

// ── Planet ────────────────────────────────────────────────────────────────────

function PlanetBody({
  planet, selected, visR, position, showOrbit, onSelect, onDoubleClick,
}: {
  planet: Planet
  selected: boolean
  visR: number
  position: THREE.Vector3
  showOrbit: boolean
  onSelect: () => void
  onDoubleClick: () => void
}) {
  const mat = useMemo(() => createPlanetMaterial(planet), [planet.id])

  if (planet.planet_type === 'asteroid_belt') {
    return <AsteroidBelt planet={planet} />
  }

  const { x: px, y: py, z: pz } = position

  return (
    <group>
      {showOrbit && (
        <OrbitEllipse
          a={planet.orbit_distance_au}
          ecc={planet.eccentricity}
          argPeriapsisDeg={planet.arg_periapsis_deg}
          inclinationDeg={planet.inclination_deg}
          color={selected ? '#4a7fa8' : '#1a3050'}
          opacity={selected ? 0.85 : 0.4}
        />
      )}

      {selected && <SelectionAura visR={visR} position={position} />}

      <mesh
        position={[px, py, pz]}
        material={mat}
        onClick={(e) => { e.stopPropagation(); onSelect() }}
        onDoubleClick={(e) => { e.stopPropagation(); onDoubleClick() }}
      >
        <sphereGeometry args={[visR, 24, 12]} />
      </mesh>
    </group>
  )
}

// ── Moon mesh ─────────────────────────────────────────────────────────────────

function MoonMesh({ moon, position, radius }: { moon: import('../types/galaxy').Moon; position: [number, number, number]; radius: number }) {
  const mat = useMemo(() => createMoonMaterial(moon), [moon.id])
  return (
    <mesh position={position} material={mat}>
      <sphereGeometry args={[radius, 10, 6]} />
    </mesh>
  )
}

// ── Scene-Inhalt ──────────────────────────────────────────────────────────────

function SystemContent({
  star, planets, selectedPlanet, onSelectPlanet, onDoubleClickPlanet,
}: {
  star: Star
  planets: Planet[]
  selectedPlanet: Planet | null
  onSelectPlanet: (p: Planet | null) => void
  onDoubleClickPlanet: (p: Planet) => void
}) {
  const { params } = useVisualParams()

  // BL-18: Radien, begrenzt durch halbe Orbital-Lücke + globales 1/10-Limit
  const visRadii = useMemo<Map<string, number>>(() => {
    const nonBelt = planets.filter(p => p.planet_type !== 'asteroid_belt')
    const sorted  = [...nonBelt].sort((a, b) => a.orbit_distance_au - b.orbit_distance_au)
    const minOrbitAU = sorted.length > 0 ? sorted[0].orbit_distance_au : 1
    const globalMaxR = minOrbitAU * 0.1

    const map = new Map<string, number>()
    for (let i = 0; i < sorted.length; i++) {
      const p       = sorted[i]
      const prevAU  = i > 0 ? sorted[i - 1].orbit_distance_au : 0
      const nextAU  = i < sorted.length - 1 ? sorted[i + 1].orbit_distance_au : p.orbit_distance_au * 1.8
      const halfGap = Math.min(p.orbit_distance_au - prevAU, nextAU - p.orbit_distance_au) * 0.4
      map.set(p.id, calcPlanetVisR(p.radius_earth, Math.min(halfGap, globalMaxR), params.planetVisMin, params.planetVisMax))
    }
    return map
  }, [planets, params.planetVisMin, params.planetVisMax])

  // BL-12: Positionen auf Kepler-Ellipse
  const positions = useMemo<Map<string, THREE.Vector3>>(() => {
    const map = new Map<string, THREE.Vector3>()
    for (const p of planets) {
      if (p.planet_type === 'asteroid_belt') continue
      map.set(p.id, computeOrbitPos(p.orbit_distance_au, p.eccentricity, p.arg_periapsis_deg, p.inclination_deg, p.orbit_index * 1.2))
    }
    return map
  }, [planets])

  // Orbit-Normalen für Chevron
  const orbitNormals = useMemo<Map<string, THREE.Vector3>>(() => {
    const map = new Map<string, THREE.Vector3>()
    for (const p of planets) {
      if (p.planet_type === 'asteroid_belt') continue
      const n = new THREE.Vector3(0, 1, 0)
      n.applyEuler(new THREE.Euler(p.inclination_deg * Math.PI / 180, p.arg_periapsis_deg * Math.PI / 180, 0))
      map.set(p.id, n.normalize())
    }
    return map
  }, [planets])

  const fallback = (p: Planet) => new THREE.Vector3(p.orbit_distance_au, 0, 0)

  // Chevron-Position: Punkt auf der Ellipse etwas vor dem Planeten
  const chevronData = useMemo(() => {
    if (!selectedPlanet || selectedPlanet.planet_type === 'asteroid_belt') return null
    const p     = selectedPlanet
    const theta = p.orbit_index * 1.2
    const visR  = visRadii.get(p.id) ?? 0.06
    const dTheta = Math.max(0.06, 3.5 * visR / p.orbit_distance_au)
    const chevPos  = computeOrbitPos(p.orbit_distance_au, p.eccentricity, p.arg_periapsis_deg, p.inclination_deg, theta + dTheta)
    const prevPos  = computeOrbitPos(p.orbit_distance_au, p.eccentricity, p.arg_periapsis_deg, p.inclination_deg, theta + dTheta - 0.008)
    const tangent  = chevPos.clone().sub(prevPos).normalize()
    const normal   = orbitNormals.get(p.id) ?? new THREE.Vector3(0, 1, 0)
    return { pos: chevPos, tangent, normal, size: visR * 0.5 }
  }, [selectedPlanet, visRadii, orbitNormals])

  return (
    <>
      {/* Punktlicht am Stern-Ursprung — beleuchtet MeshStandardMaterial-Objekte (Asteroiden) */}
      <pointLight position={[0, 0, 0]} intensity={4.0} color="#fff8e8" decay={1.5} />
      <ambientLight intensity={0.06} />

      <StarBody star={star} shaderVariant={params.starShaderVariant as 0|1|2|3|4|5} />

      {planets.map(p => (
        <PlanetBody
          key={p.id}
          planet={p}
          selected={p.id === selectedPlanet?.id}
          visR={visRadii.get(p.id) ?? 0.06}
          position={positions.get(p.id) ?? fallback(p)}
          showOrbit={params.layerOrbits}
          onSelect={() => onSelectPlanet(p.id === selectedPlanet?.id ? null : p)}
          onDoubleClick={() => onDoubleClickPlanet(p)}
        />
      ))}

      {/* Rotationsachse-Annotation (eigener Layer) */}
      {selectedPlanet && params.layerAxisInfo && selectedPlanet.planet_type !== 'asteroid_belt' && (
        <AxisAnnotation
          visR={visRadii.get(selectedPlanet.id) ?? 0.06}
          position={positions.get(selectedPlanet.id) ?? fallback(selectedPlanet)}
          axialTiltDeg={selectedPlanet.axial_tilt_deg ?? 0}
        />
      )}

      {/* Richtungschevron auf der Ellipse (eigener Layer) */}
      {chevronData && params.layerOrbitalChevron && (
        <OrbitChevron
          position={chevronData.pos}
          tangent={chevronData.tangent}
          orbitNormal={chevronData.normal}
          size={chevronData.size}
        />
      )}
    </>
  )
}

// ── Export ────────────────────────────────────────────────────────────────────

interface Props {
  star: Star
  planets: Planet[]
  selectedPlanet: Planet | null
  onSelectPlanet: (p: Planet | null) => void
  onDoubleClickPlanet: (p: Planet) => void
}

export function SystemScene({ star, planets, selectedPlanet, onSelectPlanet, onDoubleClickPlanet }: Props) {
  const maxOrbit = useMemo(
    () => planets.length ? Math.max(...planets.map(p => p.orbit_distance_au)) * 1.5 : 10,
    [planets],
  )

  // Kamera startet bei ≈ maxOrbit * 2.28 (Pythagoras aus 1.4 und 1.8).
  // maxDistance mit leichtem Puffer: 2.5 × maxOrbit.
  const maxDistance = Math.max(8, maxOrbit * 2.5)

  return (
    <Canvas
      camera={{ position: [0, maxOrbit * 1.4, maxOrbit * 1.8], fov: 60 }}
      style={{ background: '#000008' }}
    >
      <SystemContent
        star={star}
        planets={planets}
        selectedPlanet={selectedPlanet}
        onSelectPlanet={onSelectPlanet}
        onDoubleClickPlanet={onDoubleClickPlanet}
      />
      <SmartOrbitControls maxDistance={maxDistance} />
    </Canvas>
  )
}
