import { useMemo, useRef } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { SmartOrbitControls } from './SmartOrbitControls'
import * as THREE from 'three'
import type { Planet } from '../types/galaxy'
import { createPlanetMaterial } from './shaders/PlanetShader'
import { createMoonMaterial } from './shaders/MoonShader'

// ── Kreisbogen-Orbit ──────────────────────────────────────────────────────────

function OrbitRing({ radius }: { radius: number }) {
  const line = useMemo(() => {
    const pts: THREE.Vector3[] = []
    for (let i = 0; i <= 96; i++) {
      const a = (i / 96) * Math.PI * 2
      pts.push(new THREE.Vector3(Math.cos(a) * radius, 0, Math.sin(a) * radius))
    }
    const geo = new THREE.BufferGeometry().setFromPoints(pts)
    const mat = new THREE.LineBasicMaterial({ color: '#304060', transparent: true, opacity: 0.55, depthWrite: false })
    return new THREE.LineLoop(geo, mat)
  }, [radius])
  return <primitive object={line} />
}

// ── Rotierender Planetenkörper ────────────────────────────────────────────────

function RotatingPlanet({ planet, visR }: { planet: Planet; visR: number }) {
  const mat     = useMemo(() => createPlanetMaterial(planet), [planet.id])
  const meshRef = useRef<THREE.Mesh>(null!)

  // Langsame Eigenrotation
  useFrame((_, delta) => {
    meshRef.current.rotation.y += delta * 0.12
  })

  return (
    <mesh ref={meshRef} material={mat}>
      <sphereGeometry args={[visR, 48, 24]} />
    </mesh>
  )
}

// ── Mond-Mesh mit Eigenrotation ───────────────────────────────────────────────

function MoonOrbit({
  moon,
  moonR,
  orbitR,
  phase: _phase,
}: {
  moon: import('../types/galaxy').Moon
  moonR: number
  orbitR: number
  phase: number
}) {
  const mat     = useMemo(() => createMoonMaterial(moon), [moon.id])
  const groupRef = useRef<THREE.Group>(null!)

  // Mondrevolution + Eigenrotation
  useFrame((_, delta) => {
    groupRef.current.rotation.y += delta * (0.18 / orbitR)
  })

  return (
    <>
      <OrbitRing radius={orbitR} />
      <group ref={groupRef}>
        <mesh position={[orbitR, 0, 0]} material={mat}>
          <sphereGeometry args={[moonR, 14, 8]} />
        </mesh>
      </group>
    </>
  )
}

// ── Szenen-Inhalt ─────────────────────────────────────────────────────────────

function MoonSystemContent({ planet }: { planet: Planet }) {
  const PLANET_VIS_R = 1.0   // Planet immer als Einheit

  // Monde: Orbits in [2.5×, 7×] PLANET_VIS_R skaliert
  const moonData = useMemo(() => {
    if (planet.moons.length === 0) return []
    const orbits    = planet.moons.map(m => m.orbit_distance_au)
    const maxOrbit  = Math.max(...orbits, 1e-10)
    const minVis    = PLANET_VIS_R * 2.5
    const maxVis    = PLANET_VIS_R * 7.0
    const moonR     = Math.max(0.055, PLANET_VIS_R * 0.12)

    return planet.moons.map((moon, i) => ({
      moon,
      orbitR: minVis + (moon.orbit_distance_au / maxOrbit) * (maxVis - minVis),
      moonR,
      phase: i * 2.1,
    }))
  }, [planet])

  return (
    <>
      {/* Ambientes Licht als Sternersatz */}
      <ambientLight intensity={0.08} />
      <pointLight position={[20, 10, 15]} intensity={1.4} color="#fff8e8" />

      <RotatingPlanet planet={planet} visR={PLANET_VIS_R} />

      {moonData.map(({ moon, orbitR, moonR, phase }) => (
        <MoonOrbit
          key={moon.id}
          moon={moon}
          moonR={moonR}
          orbitR={orbitR}
          phase={phase}
        />
      ))}

      {/* Keine Monde vorhanden */}
      {planet.moons.length === 0 && (
        <></>
      )}
    </>
  )
}

// ── Export ────────────────────────────────────────────────────────────────────

interface Props {
  planet: Planet
}

export function MoonSystemScene({ planet }: Props) {
  // Kamera: zoomt auf den äußersten Mond-Orbit heraus
  const camDist = useMemo(() => {
    if (planet.moons.length === 0) return 6
    return 7.0 * 1.6    // maxVis * Faktor
  }, [planet])

  // Kamera startet bei ≈ camDist * 1.17 (Pythagoras aus 0.6 und 1.0).
  // maxDistance mit Puffer: 1.45 × camDist.
  const maxDistance = camDist * 1.45

  return (
    <Canvas
      camera={{ position: [0, camDist * 0.6, camDist], fov: 55 }}
      style={{ background: '#00000a' }}
    >
      <MoonSystemContent planet={planet} />
      <SmartOrbitControls maxDistance={maxDistance} />
    </Canvas>
  )
}
