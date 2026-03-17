/**
 * @frozen Design-Snapshot 2026-03-16 — nicht verändern ohne explizite Freigabe.
 * Dokumentation: galaxis/cic-visual-design_v1.0.md · Option A
 *
 * SceneExpanse — Visuelle Option A: Hard Sci-Fi / Radar-Ästhetik
 *
 * Leitprinzip: Information vor Dekoration. Klassisches Radar-Display.
 * - Amber (#d4a017) auf fast-schwarzem Grün-Schwarz
 * - Rotierende Sweep-Line als Haupt-Rhythmus
 * - Schiffe als flache Radar-Blips auf der Referenzebene projiziert
 * - Keine 3D-Sphären für Sensor-Reichweite — nur Bodensringe
 */

import { useRef, useMemo } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import type { ShipPosition } from './DropLines'

const AMBER  = 0xd4a017
const BRIGHT = 0xffd04a

// ─── Rotierende Radar-Sweep-Linie ─────────────────────────────────────────────

function RadarSweep({ size = 92 }: { size?: number }) {
  const groupRef = useRef<THREE.Group>(null)

  useFrame((_, delta) => {
    if (groupRef.current) groupRef.current.rotation.y -= delta * 0.38
  })

  const geo = useMemo(() => {
    const pts = [new THREE.Vector3(0, 0, 0), new THREE.Vector3(size, 0, 0)]
    return new THREE.BufferGeometry().setFromPoints(pts)
  }, [size])

  return (
    <group ref={groupRef}>
      {/* @ts-expect-error */}
      <line geometry={geo}>
        <lineBasicMaterial color={BRIGHT} transparent opacity={0.75} depthWrite={false} />
      </line>
    </group>
  )
}

// ─── Konzentrische Distanzringe ────────────────────────────────────────────────

function RangeRings() {
  return (
    <>
      {[25, 50, 75, 100].map((r) => (
        <mesh key={r} rotation-x={Math.PI / 2}>
          <torusGeometry args={[r, 0.1, 4, 120]} />
          <meshBasicMaterial
            color={AMBER}
            transparent
            opacity={r === 50 ? 0.28 : 0.12}
            depthWrite={false}
          />
        </mesh>
      ))}
    </>
  )
}

// ─── Flaches Raster ────────────────────────────────────────────────────────────

function FlatGrid({ size = 100, divisions = 16 }: { size?: number; divisions?: number }) {
  const geo = useMemo(() => {
    const half = size
    const step = (half * 2) / divisions
    const pts: number[] = []
    for (let i = 0; i <= divisions; i++) {
      const v = -half + i * step
      pts.push(-half, 0, v,  half, 0, v)
      pts.push(v, 0, -half,  v, 0, half)
    }
    const g = new THREE.BufferGeometry()
    g.setAttribute('position', new THREE.Float32BufferAttribute(pts, 3))
    return g
  }, [size, divisions])

  return (
    <lineSegments geometry={geo}>
      <lineBasicMaterial color={AMBER} transparent opacity={0.09} depthWrite={false} />
    </lineSegments>
  )
}

// ─── Radar-Blip ────────────────────────────────────────────────────────────────
// Projektion auf Referenzebene + Höhenanzeige als dünne Vertikallinie

function RadarBlip({ ship }: { ship: ShipPosition }) {
  const lineGeo = useMemo(() => {
    const pts = [
      new THREE.Vector3(ship.x, ship.y, ship.z),
      new THREE.Vector3(ship.x, 0.1,   ship.z),
    ]
    return new THREE.BufferGeometry().setFromPoints(pts)
  }, [ship.x, ship.y, ship.z])

  return (
    <group>
      {/* Bodenring — Hauptindikator auf der Radar-Ebene */}
      <mesh position={[ship.x, 0.15, ship.z]} rotation-x={Math.PI / 2}>
        <torusGeometry args={[1.6, 0.28, 6, 40]} />
        <meshBasicMaterial color={BRIGHT} transparent opacity={0.95} depthWrite={false} />
      </mesh>

      {/* Höhenlinie — zeigt Z-Position des Schiffs */}
      {/* @ts-expect-error */}
      <line geometry={lineGeo}>
        <lineBasicMaterial color={AMBER} transparent opacity={0.38} depthWrite={false} />
      </line>

      {/* Kleines Kreuz am eigentlichen Schiff */}
      <mesh position={[ship.x, ship.y, ship.z]} rotation-x={Math.PI / 2}>
        <torusGeometry args={[0.6, 0.18, 4, 12]} />
        <meshBasicMaterial color={BRIGHT} transparent opacity={0.7} depthWrite={false} />
      </mesh>
    </group>
  )
}

// ─── Sensor-Reichweite (Radar-Stil: 2 Bodenringe) ─────────────────────────────

interface SensorProps { ship: ShipPosition; radius: number; scanning?: boolean }

function RadarSensor({ ship, radius, scanning }: SensorProps) {
  const matRef = useRef<THREE.MeshBasicMaterial>(null)

  useFrame(({ clock }) => {
    if (scanning && matRef.current) {
      matRef.current.opacity = 0.15 + 0.12 * Math.sin(clock.elapsedTime * 2.5)
    }
  })

  return (
    <group>
      <mesh position={[ship.x, 0.2, ship.z]} rotation-x={Math.PI / 2}>
        <torusGeometry args={[radius, 0.18, 6, 100]} />
        <meshBasicMaterial
          ref={matRef}
          color={scanning ? BRIGHT : AMBER}
          transparent
          opacity={scanning ? 0.22 : 0.14}
          depthWrite={false}
        />
      </mesh>
      {/* Innerer Dämpfungsring */}
      <mesh position={[ship.x, 0.1, ship.z]} rotation-x={Math.PI / 2}>
        <torusGeometry args={[radius * 0.6, 0.08, 4, 80]} />
        <meshBasicMaterial color={AMBER} transparent opacity={0.08} depthWrite={false} />
      </mesh>
    </group>
  )
}

// ─── Haupt-Export ──────────────────────────────────────────────────────────────

export function SceneExpanse({ ships }: { ships: ShipPosition[] }) {
  return (
    <>
      <color attach="background" args={['#000d05']} />
      <FlatGrid />
      <RangeRings />
      <RadarSweep />
      {ships.map(s => <RadarBlip key={s.id} ship={s} />)}
      <RadarSensor ship={ships[0]} radius={28} />
      <RadarSensor ship={ships[1]} radius={32} scanning />
      <RadarSensor ship={ships[3]} radius={24} />
    </>
  )
}
