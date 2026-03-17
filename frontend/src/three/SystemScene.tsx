import { useMemo } from 'react'
import { Canvas } from '@react-three/fiber'
import { OrbitControls } from '@react-three/drei'
import * as THREE from 'three'
import type { Planet, Star } from '../types/galaxy'

// ── Visuelle Konstanten ───────────────────────────────────────────────────────

const PLANET_COLORS: Record<string, string> = {
  rocky:         '#c87941',
  gas_giant:     '#d4a86a',
  ice_giant:     '#6ab4d4',
  asteroid_belt: '#888888',
}

// Visueller Radius in AU (nicht physikalisch)
const PLANET_VIS_R: Record<string, number> = {
  rocky:         0.08,
  gas_giant:     0.25,
  ice_giant:     0.18,
  asteroid_belt: 0.04,
}

const MOON_COMP_COLORS: Record<string, string> = {
  rocky: '#a09080',
  icy:   '#b0c8d8',
  mixed: '#90a898',
}

// ── Orbitring (LineLoop) ──────────────────────────────────────────────────────

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

// ── Stern im Zentrum ──────────────────────────────────────────────────────────

function StarBody({ star }: { star: Star }) {
  const r = useMemo(() => {
    const type = star.star_type
    if (type === 'SMBH')               return 0.6
    if (type === 'StellarBH')          return 0.18
    if (type === 'Pulsar')             return 0.12
    const sr = star.radius_solar ?? 1
    return Math.min(Math.max(sr * 0.04, 0.12), 0.5)
  }, [star])

  return (
    <mesh>
      <sphereGeometry args={[r, 20, 10]} />
      <meshBasicMaterial color={star.color_hex ?? '#ffffff'} />
    </mesh>
  )
}

// ── Planet ────────────────────────────────────────────────────────────────────

function PlanetBody({
  planet, selected, onSelect,
}: {
  planet: Planet
  selected: boolean
  onSelect: () => void
}) {
  // Deterministisch über orbit_index verteilen, kein Magic durch rng
  const phi = planet.orbit_index * 1.2
  const d   = planet.orbit_distance_au
  const px  = Math.cos(phi) * d
  const pz  = Math.sin(phi) * d
  const r   = PLANET_VIS_R[planet.planet_type] ?? 0.08
  const col = selected ? '#ffffff' : (PLANET_COLORS[planet.planet_type] ?? '#888888')

  if (planet.planet_type === 'asteroid_belt') {
    return (
      <group>
        <OrbitRing radius={d}         color="#777777" opacity={0.55} />
        <OrbitRing radius={d + 0.4}   color="#555555" opacity={0.25} />
      </group>
    )
  }

  return (
    <group>
      <OrbitRing
        radius={d}
        color={selected ? '#4a7fa8' : '#1a3050'}
        opacity={selected ? 0.85 : 0.4}
      />

      {/* Planetkörper — klickbar */}
      <mesh position={[px, 0, pz]} onClick={(e) => { e.stopPropagation(); onSelect() }}>
        <sphereGeometry args={[r, 16, 8]} />
        <meshBasicMaterial color={col} />
      </mesh>

      {/* Auswahlring */}
      {selected && (
        <mesh position={[px, 0, pz]} rotation-x={Math.PI / 2}>
          <torusGeometry args={[r * 1.9, r * 0.14, 6, 48]} />
          <meshBasicMaterial color="#4af0ff" transparent opacity={0.55} depthWrite={false} />
        </mesh>
      )}
    </group>
  )
}

// ── Mondsystem (nur bei ausgewähltem Planeten) ────────────────────────────────

function MoonSystem({ planet }: { planet: Planet }) {
  const phi = planet.orbit_index * 1.2
  const px  = Math.cos(phi) * planet.orbit_distance_au
  const pz  = Math.sin(phi) * planet.orbit_distance_au

  const baseR   = (PLANET_VIS_R[planet.planet_type] ?? 0.08) * 3.5
  const stepR   = baseR * 0.6
  const moonDot = Math.max(0.02, (PLANET_VIS_R[planet.planet_type] ?? 0.08) * 0.32)

  return (
    <group position={[px, 0, pz]}>
      {planet.moons.map((moon, i) => {
        const mr   = baseR + i * stepR
        const mphi = i * 2.1
        const mx   = Math.cos(mphi) * mr
        const mz   = Math.sin(mphi) * mr
        return (
          <group key={moon.id}>
            <OrbitRing radius={mr} color="#304060" opacity={0.5} />
            <mesh position={[mx, 0, mz]}>
              <sphereGeometry args={[moonDot, 8, 5]} />
              <meshBasicMaterial color={MOON_COMP_COLORS[moon.composition_type] ?? '#909090'} />
            </mesh>
          </group>
        )
      })}
    </group>
  )
}

// ── Scene-Inhalt ──────────────────────────────────────────────────────────────

function SystemContent({
  star, planets, selectedPlanet, onSelectPlanet,
}: {
  star: Star
  planets: Planet[]
  selectedPlanet: Planet | null
  onSelectPlanet: (p: Planet | null) => void
}) {
  return (
    <>
      <StarBody star={star} />
      {planets.map(p => (
        <PlanetBody
          key={p.id}
          planet={p}
          selected={p.id === selectedPlanet?.id}
          onSelect={() => onSelectPlanet(p.id === selectedPlanet?.id ? null : p)}
        />
      ))}
      {selectedPlanet && selectedPlanet.moons.length > 0 && (
        <MoonSystem planet={selectedPlanet} />
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
}

export function SystemScene({ star, planets, selectedPlanet, onSelectPlanet }: Props) {
  const maxOrbit = useMemo(
    () => planets.length
      ? Math.max(...planets.map(p => p.orbit_distance_au)) * 1.5
      : 10,
    [planets],
  )

  return (
    <Canvas
      camera={{ position: [0, maxOrbit * 2.2, 0], up: [0, 0, -1], fov: 60 }}
      style={{ background: '#000008' }}
    >
      <SystemContent
        star={star}
        planets={planets}
        selectedPlanet={selectedPlanet}
        onSelectPlanet={onSelectPlanet}
      />
      <OrbitControls
        makeDefault
        enablePan
        enableZoom
        enableRotate={false}
        zoomSpeed={1.2}
      />
    </Canvas>
  )
}
