/**
 * DropLines — Lot-Linien von Schiffen zur taktischen Referenzebene
 *
 * Zweck (ui-analyse 3.5):
 *  Ohne Lot-Linien fehlt dem Spieler jede Tiefenreferenz im 3D-Raum.
 *  Jede Linie verbindet die Schiffsposition (x, y, z) mit dem Punkt
 *  auf der Referenzfläche direkt darunter — inklusive Gravity-Well-Verzerrung,
 *  d.h. die Linie endet auf der tatsächlich angezeigten Gitterfläche.
 *
 * Visuell (Homeworld-Referenz, homeworld_positioning.jpg):
 *  - Gestrichelte orange Linie (LineDashedMaterial)
 *  - Kleiner Wireframe-Marker an der Schiffsposition
 *  - Der Endpunkt der Linie wandert mit dem Gravity-Well-Grid mit
 *
 * Token-Referenz:
 *  --color-galaxis-orange  #ff8c00  (Bewegung, Drop-Lines)
 *  --color-galaxis-cyan    #00d4ff  (eigene Einheiten — Marker-Farbe)
 */

import { useMemo, useRef, useEffect } from 'react'
import * as THREE from 'three'

// ─── Typen ────────────────────────────────────────────────────────────────────

export interface ShipPosition {
  id:   string
  x:    number
  /** Höhe über der (flachen) Referenzebene in World-Units */
  y:    number
  z:    number
}

interface DropLinesProps {
  ships:      ShipPosition[]
  /** Gravity-Well-Parameter — müssen mit TacticalGrid übereinstimmen */
  wellDepth?: number
  wellScale?: number
}

// ─── Hilfsfunktion (identisch zu TacticalGrid) ────────────────────────────────

function gravityDisplacement(x: number, z: number, depth: number, scale: number): number {
  const r = Math.sqrt(x * x + z * z)
  return -depth / (r / scale + 1)
}

// ─── Einzelne Lot-Linie ───────────────────────────────────────────────────────

interface DropLineProps {
  ship:      ShipPosition
  wellDepth: number
  wellScale: number
}

function DropLine({ ship, wellDepth, wellScale }: DropLineProps) {
  const lineRef = useRef<THREE.Line>(null)

  const geometry = useMemo(() => {
    const groundY = gravityDisplacement(ship.x, ship.z, wellDepth, wellScale)
    const points = [
      new THREE.Vector3(ship.x, ship.y, ship.z),
      new THREE.Vector3(ship.x, groundY, ship.z),
    ]
    return new THREE.BufferGeometry().setFromPoints(points)
  }, [ship.x, ship.y, ship.z, wellDepth, wellScale])

  // computeLineDistances() muss auf dem Line-Objekt aufgerufen werden (Three.js r125+)
  useEffect(() => {
    lineRef.current?.computeLineDistances()
  }, [geometry])

  return (
    // @ts-expect-error — R3F lowercase 'line' ist die Three.js Line-Primitive
    <line ref={lineRef} geometry={geometry}>
      <lineDashedMaterial
        color={0xff8c00}
        dashSize={1.8}
        gapSize={1.0}
        transparent
        opacity={0.65}
        depthWrite={false}
      />
    </line>
  )
}

// ─── Schiffs-Marker ───────────────────────────────────────────────────────────

function ShipMarker({ ship }: { ship: ShipPosition }) {
  return (
    <mesh position={[ship.x, ship.y, ship.z]}>
      <octahedronGeometry args={[1.2, 0]} />
      <meshBasicMaterial
        color={0x00d4ff}
        wireframe
        transparent
        opacity={0.85}
      />
    </mesh>
  )
}

// ─── Bodenpunkt-Marker ────────────────────────────────────────────────────────
// Kleines Kreuz auf der Gitterfläche — zeigt exakt wo die Lot-Linie landet.

function GroundMark({ ship, wellDepth, wellScale }: DropLineProps) {
  const geometry = useMemo(() => {
    const y = gravityDisplacement(ship.x, ship.z, wellDepth, wellScale)
    const s = 1.5
    const pts = [
      new THREE.Vector3(ship.x - s, y, ship.z),
      new THREE.Vector3(ship.x + s, y, ship.z),
      new THREE.Vector3(ship.x, y, ship.z - s),
      new THREE.Vector3(ship.x, y, ship.z + s),
    ]
    // Zwei separate Segmente als LineSegments
    return new THREE.BufferGeometry().setFromPoints(pts)
  }, [ship.x, ship.z, wellDepth, wellScale])

  return (
    <lineSegments geometry={geometry}>
      <lineBasicMaterial color={0xff8c00} transparent opacity={0.5} depthWrite={false} />
    </lineSegments>
  )
}

// ─── Haupt-Export ─────────────────────────────────────────────────────────────

export function DropLines({ ships, wellDepth = 10, wellScale = 28 }: DropLinesProps) {
  return (
    <group>
      {ships.map(ship => (
        <group key={ship.id}>
          <ShipMarker  ship={ship} />
          <DropLine    ship={ship} wellDepth={wellDepth} wellScale={wellScale} />
          <GroundMark  ship={ship} wellDepth={wellDepth} wellScale={wellScale} />
        </group>
      ))}
    </group>
  )
}
