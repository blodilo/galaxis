/**
 * @frozen Design-Snapshot 2026-03-16 — nicht verändern ohne explizite Freigabe.
 * Dokumentation: galaxis/cic-visual-design_v1.0.md · Option C
 *
 * SceneGalaxis — Visuelle Option C: Eigene Galaxis-Sprache
 *
 * Kernthese: Das Gravity-Well-Grid IST die spielentscheidende Information —
 * es macht Treibstoffkosten sichtbar. Schiffe sind sekundäre Marker.
 *
 * Unterschiede zu Homeworld:
 *  - Marker: "Taktische Nadel" (dünner Stab + Kopfring) statt Oktaeder
 *  - Lot: pulsierende Boden-Ringe statt gestrichelte Vertikallinie
 *  - Sensor: Zwei schiefe Ringe (Armillarsphären-Skizze) statt Kugelvolumen
 *  - Farben: Token-Palette, aber Cyan nur für eigene Einheiten; feindlich: Rot
 */

import { useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import { TacticalGrid } from './TacticalGrid'
import type { ShipPosition } from './DropLines'

const CYAN    = 0x00d4ff
const RED     = 0xff2222
const BLUE    = 0x1e64ff

function gravityDisplacement(x: number, z: number, depth: number, scale: number): number {
  const r = Math.sqrt(x * x + z * z)
  return -depth / (r / scale + 1)
}

// ─── Taktische Nadel ──────────────────────────────────────────────────────────
// Dünner Stab von Boden zum Schiff + flacher Kopfring an Schiffsposition

interface PinProps { ship: ShipPosition; color: number }

function TacticalPin({ ship, color }: PinProps) {
  const groundY = gravityDisplacement(ship.x, ship.z, 10, 28)

  return (
    <group>
      {/* Stab */}
      <mesh position={[ship.x, groundY + (ship.y - groundY) / 2, ship.z]}>
        <cylinderGeometry args={[0.07, 0.07, ship.y - groundY, 6]} />
        <meshBasicMaterial color={color} transparent opacity={0.55} depthWrite={false} />
      </mesh>

      {/* Kopfring — liegt in der XZ-Ebene am Schiff */}
      <mesh position={[ship.x, ship.y, ship.z]} rotation-x={Math.PI / 2}>
        <torusGeometry args={[1.4, 0.22, 8, 36]} />
        <meshBasicMaterial color={color} transparent opacity={0.90} depthWrite={false} />
      </mesh>

      {/* Kleiner Kern-Punkt */}
      <mesh position={[ship.x, ship.y, ship.z]}>
        <sphereGeometry args={[0.35, 8, 6]} />
        <meshBasicMaterial color={color} depthWrite={false} />
      </mesh>
    </group>
  )
}

// ─── Pulsierender Boden-Ring ───────────────────────────────────────────────────
// Ersetzt die gestrichelten Lot-Linien — kommuniziert Position auf Referenzebene

function GroundPulse({ ship, color }: PinProps) {
  const meshRef = useRef<THREE.Mesh>(null)
  const matRef  = useRef<THREE.MeshBasicMaterial>(null)
  const phase   = useRef(Math.random() * Math.PI * 2)
  const groundY = gravityDisplacement(ship.x, ship.z, 10, 28)

  useFrame(({ clock }) => {
    const t = clock.elapsedTime * 1.1 + phase.current
    const pulse = 0.5 + 0.5 * Math.sin(t)
    if (meshRef.current) meshRef.current.scale.setScalar(1 + pulse * 0.25)
    if (matRef.current)  matRef.current.opacity = 0.12 + pulse * 0.28
  })

  return (
    <mesh ref={meshRef} position={[ship.x, groundY + 0.15, ship.z]} rotation-x={Math.PI / 2}>
      <torusGeometry args={[2.4, 0.18, 6, 48]} />
      <meshBasicMaterial
        ref={matRef}
        color={color}
        transparent
        opacity={0.25}
        depthWrite={false}
      />
    </mesh>
  )
}

// ─── Armillarsphären-Sensor ───────────────────────────────────────────────────
// Zwei schiefe Ringe skizzieren den Erfassungsradius — kein Sphären-Volumen

interface ArmillaryProps { ship: ShipPosition; radius: number; color: number; scanning?: boolean }

function ArmillarySensor({ ship, radius, color, scanning }: ArmillaryProps) {
  const matRef = useRef<THREE.MeshBasicMaterial>(null)
  const matRef2 = useRef<THREE.MeshBasicMaterial>(null)

  useFrame(({ clock }) => {
    if (!scanning) return
    const ping = (clock.elapsedTime * 0.8) % 1
    if (matRef.current)  matRef.current.opacity  = (1 - ping) * 0.32
    if (matRef2.current) matRef2.current.opacity = (1 - ping) * 0.20
  })

  const pos: [number, number, number] = [ship.x, ship.y, ship.z]

  return (
    <group position={pos}>
      {/* Horizontaler Äquatorring */}
      <mesh rotation-x={Math.PI / 2}>
        <torusGeometry args={[radius, 0.20, 8, 100]} />
        <meshBasicMaterial ref={matRef} color={color} transparent opacity={0.28} depthWrite={false} />
      </mesh>

      {/* Geneigter Ring (30°) — Armillar-Effekt */}
      <mesh rotation-z={Math.PI / 6}>
        <torusGeometry args={[radius, 0.12, 6, 100]} />
        <meshBasicMaterial ref={matRef2} color={color} transparent opacity={0.14} depthWrite={false} />
      </mesh>
    </group>
  )
}

// ─── Haupt-Export ──────────────────────────────────────────────────────────────

export function SceneGalaxis({ ships }: { ships: ShipPosition[] }) {
  return (
    <>
      <color attach="background" args={['#060a12']} />

      {/* Gravity-Well-Grid — Herzstück der Taktikansicht */}
      <TacticalGrid />

      {/* Schiffs-Marker */}
      {ships.map((s) => {
        const color = s.id === 's4' ? RED : CYAN
        return (
          <group key={s.id}>
            <TacticalPin  ship={s} color={color} />
            <GroundPulse  ship={s} color={color} />
          </group>
        )
      })}

      {/* Sensor-Reichweiten (Armillar-Stil) */}
      <ArmillarySensor ship={ships[0]} radius={28} color={BLUE} />
      <ArmillarySensor ship={ships[1]} radius={32} color={BLUE} scanning />
      <ArmillarySensor ship={ships[3]} radius={24} color={RED} />
    </>
  )
}
