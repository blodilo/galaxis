/**
 * @frozen Design-Snapshot 2026-03-16 — nicht verändern ohne explizite Freigabe.
 * Dokumentation: galaxis/cic-visual-design_v1.0.md · Option B · Refinement v5
 *
 * SceneMilitary — Visuelle Option B: Modernes Militär-HUD
 *
 * Marker-System pro Einheit:
 *  1. Kreuz auf der Referenzebene
 *  2. Dünne Vertikallinie (Höhe)
 *  3. NATO-Schild (Billboard, Anker linke obere Ecke)
 *     Farben: blau = Freund · weiß = Neutral · rot = Feind
 *  4. Verblassende Trajektorie (gleiche Dicke, zunehmend transparent nach hinten)
 *  5. Flugvektor-Pfeil — tangential zur Trajektorie, doppelte Dicke (Zylinder-Schaft)
 *  6. Unsicherheitskonus — Spitze an Schiffsposition, Basis offen vorwärts
 */

import { useRef, useMemo, useEffect } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import type { ShipPosition } from './DropLines'

type Variant  = 'friendly' | 'neutral' | 'enemy'
type UnitType = 'fighter' | 'destroyer' | 'cruiser' | 'battleship'

// Farben je Partie
const COLOR: Record<Variant, number> = {
  friendly: 0x2266ff,
  neutral:  0xccdde8,
  enemy:    0xff1122,
}

// ─── Manöver-Daten pro Schiff (Index = DEMO_SHIPS-Index) ─────────────────────

interface MilData {
  velocity:       THREE.Vector3
  posUncertainty: number
  dirUncertainty: number
  unitType:       UnitType
  variant:        Variant
}

const MIL_DATA: MilData[] = [
  { velocity: new THREE.Vector3( 8,  0,  4), posUncertainty: 12, dirUncertainty: 0.17, unitType: 'fighter',    variant: 'friendly' },
  { velocity: new THREE.Vector3(-12,-2,  5), posUncertainty: 10, dirUncertainty: 0.14, unitType: 'destroyer',  variant: 'friendly' },
  { velocity: new THREE.Vector3( 6,  2, -8), posUncertainty: 15, dirUncertainty: 0.21, unitType: 'cruiser',    variant: 'neutral'  },
  { velocity: new THREE.Vector3(-9, -1,  6), posUncertainty: 20, dirUncertainty: 0.30, unitType: 'battleship', variant: 'enemy'    },
  { velocity: new THREE.Vector3( 4,  1,  9), posUncertainty:  9, dirUncertainty: 0.15, unitType: 'fighter',    variant: 'friendly' },
  { velocity: new THREE.Vector3(11,  3, -3), posUncertainty: 11, dirUncertainty: 0.19, unitType: 'cruiser',    variant: 'neutral'  },
  { velocity: new THREE.Vector3(-7, -4,-10), posUncertainty: 13, dirUncertainty: 0.24, unitType: 'destroyer',  variant: 'neutral'  },
]

// ─── Trajektorie — elliptischer Bogen ─────────────────────────────────────────

function buildTrail(ship: ShipPosition, vel: THREE.Vector3, steps = 18): THREE.Vector3[] {
  const velDir = vel.clone().normalize()
  const ref    = Math.abs(velDir.y) > 0.9
    ? new THREE.Vector3(1, 0, 0)
    : new THREE.Vector3(0, 1, 0)
  const perp   = new THREE.Vector3().crossVectors(velDir, ref).normalize()

  const totalBack = vel.length() * 3.2
  const arcDepth  = totalBack * 0.30
  const sign      = ship.id.charCodeAt(1) % 2 === 0 ? 1 : -1
  const origin    = new THREE.Vector3(ship.x, ship.y, ship.z)
  const points: THREE.Vector3[] = []

  for (let i = 0; i <= steps; i++) {
    const t = i / steps
    const pos = origin.clone().addScaledVector(velDir.clone().negate(), t * totalBack)
    pos.addScaledVector(perp, arcDepth * Math.sin(t * Math.PI) * sign)
    points.push(pos)
  }
  return points  // [0] = Schiff (hell), [N] = älteste Position (transparent)
}

// ─── Raster ───────────────────────────────────────────────────────────────────

function MilGrid() {
  const geo = useMemo(() => {
    const half = 100, step = 10
    const pts: number[] = []
    for (let v = -half; v <= half; v += step) {
      pts.push(-half, 0, v,  half, 0, v)
      pts.push(v, 0, -half,  v, 0, half)
    }
    const g = new THREE.BufferGeometry()
    g.setAttribute('position', new THREE.Float32BufferAttribute(pts, 3))
    return g
  }, [])
  return (
    <lineSegments geometry={geo}>
      <lineBasicMaterial color={0x5577aa} transparent opacity={0.09} depthWrite={false} />
    </lineSegments>
  )
}

// ─── Bodenkreuz ───────────────────────────────────────────────────────────────

function GroundCross({ ship, color }: { ship: ShipPosition; color: number }) {
  const geo = useMemo(() => {
    const s = 1.8
    return new THREE.BufferGeometry().setFromPoints([
      new THREE.Vector3(ship.x - s, 0.08, ship.z),
      new THREE.Vector3(ship.x + s, 0.08, ship.z),
      new THREE.Vector3(ship.x,     0.08, ship.z - s),
      new THREE.Vector3(ship.x,     0.08, ship.z + s),
    ])
  }, [ship.x, ship.z])
  return (
    <lineSegments geometry={geo}>
      <lineBasicMaterial color={color} transparent opacity={0.50} depthWrite={false} />
    </lineSegments>
  )
}

// ─── Vertikale Lot-Linie ───────────────────────────────────────────────────────

function VerticalLine({ ship, color }: { ship: ShipPosition; color: number }) {
  const geo = useMemo(() => new THREE.BufferGeometry().setFromPoints([
    new THREE.Vector3(ship.x, 0.08, ship.z),
    new THREE.Vector3(ship.x, ship.y,  ship.z),
  ]), [ship.x, ship.y, ship.z])
  return (
    // @ts-expect-error
    <line geometry={geo}>
      <lineBasicMaterial color={color} transparent opacity={0.28} depthWrite={false} />
    </line>
  )
}

// ─── Einheitensymbole ─────────────────────────────────────────────────────────

function SymbolFighter({ cx, cy, color }: { cx: number; cy: number; color: number }) {
  const geo = useMemo(() => {
    const b = 0.28, h = 0.33, gap = 0.58
    const pts: THREE.Vector3[] = []
    for (const dx of [-gap, 0, gap]) {
      const bl = new THREE.Vector3(cx + dx - b, cy - h * 0.5, 0)
      const br = new THREE.Vector3(cx + dx + b, cy - h * 0.5, 0)
      const tp = new THREE.Vector3(cx + dx,     cy + h * 0.5, 0)
      pts.push(bl, br, br, tp, tp, bl)
    }
    return new THREE.BufferGeometry().setFromPoints(pts)
  }, [cx, cy])
  return (
    <lineSegments geometry={geo}>
      <lineBasicMaterial color={color} transparent opacity={0.65} depthWrite={false} />
    </lineSegments>
  )
}

function SymbolDestroyer({ cx, cy, color }: { cx: number; cy: number; color: number }) {
  const geo = useMemo(() => {
    const b = 0.55, h = 0.65
    const pos = new Float32Array([cx, cy + h * 0.5, 0, cx - b, cy - h * 0.5, 0, cx + b, cy - h * 0.5, 0])
    const g = new THREE.BufferGeometry()
    g.setAttribute('position', new THREE.Float32BufferAttribute(pos, 3))
    return g
  }, [cx, cy])
  return (
    <mesh geometry={geo}>
      <meshBasicMaterial color={color} transparent opacity={0.60} depthWrite={false} side={THREE.DoubleSide} />
    </mesh>
  )
}

function SymbolCruiser({ cx, cy, color }: { cx: number; cy: number; color: number }) {
  const geo = useMemo(() => {
    const hw = 0.55, hh = 0.38
    const TL = new THREE.Vector3(cx - hw, cy + hh, 0)
    const TR = new THREE.Vector3(cx + hw, cy + hh, 0)
    const BR = new THREE.Vector3(cx + hw, cy - hh, 0)
    const BL = new THREE.Vector3(cx - hw, cy - hh, 0)
    return new THREE.BufferGeometry().setFromPoints([TL, TR, TR, BR, BR, BL, BL, TL])
  }, [cx, cy])
  return (
    <lineSegments geometry={geo}>
      <lineBasicMaterial color={color} transparent opacity={0.65} depthWrite={false} />
    </lineSegments>
  )
}

function SymbolBattleship({ cx, cy, color }: { cx: number; cy: number; color: number }) {
  return (
    <mesh position={[cx, cy, 0]}>
      <planeGeometry args={[1.1, 0.76]} />
      <meshBasicMaterial color={color} transparent opacity={0.55} depthWrite={false} side={THREE.DoubleSide} />
    </mesh>
  )
}

// ─── NATO-Schild (Billboard) ──────────────────────────────────────────────────
// Anker: linke obere Ecke. Farbcodierung: blau/weiß/rot per Partie.

const BW = 1.8, BH = 1.05

function NatoBillboard({ ship, variant, unitType }: {
  ship: ShipPosition; variant: Variant; unitType: UnitType
}) {
  const groupRef = useRef<THREE.Group>(null)
  useFrame(({ camera }) => { groupRef.current?.quaternion.copy(camera.quaternion) })

  const color   = COLOR[variant]
  const opacity = variant === 'enemy' ? 0.72 : 0.52

  const frameGeo = useMemo(() => {
    const W = BW * 2, H = BH * 2
    const TL = new THREE.Vector3(0, 0,  0)
    const TR = new THREE.Vector3(W, 0,  0)
    const BR = new THREE.Vector3(W, -H, 0)
    const BL = new THREE.Vector3(0, -H, 0)
    return new THREE.BufferGeometry().setFromPoints([TL, TR, TR, BR, BR, BL, BL, TL])
  }, [])

  const cx = BW, cy = -BH

  return (
    <group ref={groupRef} position={[ship.x, ship.y, ship.z]}>
      <lineSegments geometry={frameGeo}>
        <lineBasicMaterial color={color} transparent opacity={opacity} depthWrite={false} />
      </lineSegments>
      {unitType === 'fighter'    && <SymbolFighter   cx={cx} cy={cy} color={color} />}
      {unitType === 'destroyer'  && <SymbolDestroyer cx={cx} cy={cy} color={color} />}
      {unitType === 'cruiser'    && <SymbolCruiser   cx={cx} cy={cy} color={color} />}
      {unitType === 'battleship' && <SymbolBattleship cx={cx} cy={cy} color={color} />}
    </group>
  )
}

// ─── Trajektorie (einfache Linien, gleiche Dicke, zunehmend transparent) ──────

function TrailLine({ from, to, opacity, color }: {
  from: THREE.Vector3; to: THREE.Vector3; opacity: number; color: number
}) {
  const geo = useMemo(
    () => new THREE.BufferGeometry().setFromPoints([from, to]),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [from.x, from.y, from.z, to.x, to.y, to.z]
  )
  return (
    // @ts-expect-error
    <line geometry={geo}>
      <lineBasicMaterial color={color} transparent opacity={opacity} depthWrite={false} />
    </line>
  )
}

function Trail({ trail, color }: { trail: THREE.Vector3[]; color: number }) {
  const N = trail.length - 1
  return (
    <>
      {trail.slice(0, N).map((p, i) => (
        <TrailLine
          key={i}
          from={p}
          to={trail[i + 1]}
          opacity={(1 - i / N) * 0.62}
          color={color}
        />
      ))}
    </>
  )
}

// ─── Flugvektor-Pfeil (tangential zur Trajektorie, Zylinder-Schaft) ───────────

function VelocityArrow({ trail, speed, color }: {
  trail: THREE.Vector3[]; speed: number; color: number
}) {
  // Richtung = Tangente an trail[0]: vorwärts = trail[0] - trail[1]
  const origin = trail[0]
  const dir    = trail[0].clone().sub(trail[1]).normalize()
  const tip    = origin.clone().addScaledVector(dir, speed)
  const _mid   = origin.clone().lerp(tip, 0.5)
  const headH  = Math.max(1.6, speed * 0.13)
  const shaftH = Math.max(0, speed - headH)

  const shaftRef = useRef<THREE.Mesh>(null)
  const headRef  = useRef<THREE.Group>(null)

  useEffect(() => {
    const q = new THREE.Quaternion().setFromUnitVectors(new THREE.Vector3(0, 1, 0), dir)
    shaftRef.current?.quaternion.copy(q)
    headRef.current?.quaternion.copy(q)
  })

  const shaftMid = origin.clone().addScaledVector(dir, shaftH / 2)

  return (
    <group>
      {/* Schaft — Zylinder für doppelte visuelle Dicke */}
      <mesh ref={shaftRef} position={shaftMid.toArray()}>
        <cylinderGeometry args={[0.22, 0.22, shaftH, 8, 1]} />
        <meshBasicMaterial color={color} transparent opacity={0.80} depthWrite={false} />
      </mesh>
      {/* Pfeilkopf */}
      <group ref={headRef} position={tip.toArray()}>
        <mesh position={[0, headH / 2, 0]}>
          <coneGeometry args={[headH * 0.40, headH, 8, 1]} />
          <meshBasicMaterial color={color} transparent opacity={0.82} depthWrite={false} />
        </mesh>
      </group>
    </group>
  )
}

// ─── Unsicherheitskonus (Spitze an Schiffsposition, Basis vorwärts) ───────────

function UncertaintyCone({ ship, trail, posUncertainty, dirUncertainty, color }: {
  ship: ShipPosition; trail: THREE.Vector3[]
  posUncertainty: number; dirUncertainty: number; color: number
}) {
  const dir        = trail[0].clone().sub(trail[1]).normalize()
  const baseRadius = posUncertainty * Math.tan(dirUncertainty)
  const coneRef    = useRef<THREE.Group>(null)

  useEffect(() => {
    coneRef.current?.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), dir)
  })

  return (
    <group ref={coneRef} position={[ship.x, ship.y, ship.z]}>
      {/* rotation-x={Math.PI}: Spitze am Ursprung (Schiff), Basis öffnet vorwärts */}
      <mesh position={[0, posUncertainty / 2, 0]} rotation-x={Math.PI}>
        <coneGeometry args={[baseRadius, posUncertainty, 20, 1, true]} />
        <meshBasicMaterial color={color} transparent opacity={0.05} side={THREE.DoubleSide} depthWrite={false} />
      </mesh>
      <mesh position={[0, posUncertainty, 0]} rotation-x={Math.PI / 2}>
        <torusGeometry args={[baseRadius, 0.10, 5, 72]} />
        <meshBasicMaterial color={color} transparent opacity={0.20} depthWrite={false} />
      </mesh>
    </group>
  )
}

// ─── Haupt-Export ──────────────────────────────────────────────────────────────

export function SceneMilitary({ ships }: { ships: ShipPosition[] }) {
  return (
    <>
      <color attach="background" args={['#020810']} />
      <MilGrid />
      {ships.map((s, i) => {
        const mil   = MIL_DATA[i]
        const color = COLOR[mil.variant]
        const trail = buildTrail(s, mil.velocity)
        return (
          <group key={s.id}>
            <GroundCross     ship={s} color={color} />
            <VerticalLine    ship={s} color={color} />
            <NatoBillboard   ship={s} variant={mil.variant} unitType={mil.unitType} />
            <Trail           trail={trail} color={color} />
            <VelocityArrow   trail={trail} speed={mil.velocity.length()} color={0xdcecf8} />
            <UncertaintyCone
              ship={s}
              trail={trail}
              posUncertainty={mil.posUncertainty}
              dirUncertainty={mil.dirUncertainty}
              color={color}
            />
          </group>
        )
      })}
    </>
  )
}
