/**
 * SensorHalo — Sensor-Erfassungsradius als halbtransparente Sphäre
 *
 * Konzept (ui-analyse 3.3 + 3.4):
 *  - Der Halo gehört dem BEOBACHTER, nicht dem Beobachteten.
 *  - Passiver Halo (immer aktiv): zeigt Reichweite des Passiv-Sensors.
 *  - Aktiver Scan (scanning=true): Ping-Ring expandiert nach außen und verblasst.
 *    Dieser Ring ist für alle Parteien sichtbar — Leuchtturm-Effekt.
 *
 * Varianten:
 *  - 'friendly' → blau  (--color-galaxis-range-friendly)
 *  - 'enemy'    → rot   (--color-galaxis-range-enemy)
 *
 * Token-Referenz:
 *  --color-galaxis-range-friendly   rgba(30, 100, 255, 0.20)
 *  --color-galaxis-range-enemy      rgba(255, 30, 30, 0.30)
 *  --color-galaxis-range-scan-ping  rgba(0, 212, 255, 0.15)
 */

import { useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'

// ─── Typen ────────────────────────────────────────────────────────────────────

export interface SensorHaloProps {
  position:  [number, number, number]
  /** Sensor-Erfassungsradius in World-Units */
  radius?:   number
  variant?:  'friendly' | 'enemy'
  /** Aktiver Scan aktiv → Ping-Ring animiert sich nach außen (Leuchtturm-Effekt) */
  scanning?: boolean
}

// ─── Farb-Konfiguration ───────────────────────────────────────────────────────

const VARIANT = {
  friendly: {
    fillColor:  0x1e64ff,
    fillOpacity: 0.05,
    ringColor:  0x1e64ff,
    ringOpacity: 0.22,
  },
  enemy: {
    fillColor:  0xff1e1e,
    fillOpacity: 0.08,
    ringColor:  0xff1e1e,
    ringOpacity: 0.32,
  },
}

// ─── Ping-Ring (animiert) ─────────────────────────────────────────────────────

interface PingProps {
  radius: number
}

function ScanPing({ radius }: PingProps) {
  const meshRef  = useRef<THREE.Mesh>(null)
  const matRef   = useRef<THREE.MeshBasicMaterial>(null)
  const progress = useRef(0)

  useFrame((_, delta) => {
    progress.current = (progress.current + delta * 0.35) % 1

    if (meshRef.current) {
      // Ring expandiert von 1× auf 2× des Halo-Radius
      const scale = 1 + progress.current
      meshRef.current.scale.setScalar(scale)
    }
    if (matRef.current) {
      // Opacity sinkt von 0.5 → 0 während des Expansions
      matRef.current.opacity = (1 - progress.current) * 0.5
    }
  })

  return (
    <mesh ref={meshRef} rotation-x={Math.PI / 2}>
      <torusGeometry args={[radius, 0.4, 8, 80]} />
      <meshBasicMaterial
        ref={matRef}
        color={0x00d4ff}
        transparent
        opacity={0.5}
        depthWrite={false}
        side={THREE.DoubleSide}
      />
    </mesh>
  )
}

// ─── Haupt-Export ─────────────────────────────────────────────────────────────

export function SensorHalo({
  position,
  radius   = 25,
  variant  = 'friendly',
  scanning = false,
}: SensorHaloProps) {
  const cfg = VARIANT[variant]

  return (
    <group position={position}>

      {/* Halbtransparente Sphärenfüllung — zeigt das Volumen */}
      <mesh>
        <sphereGeometry args={[radius, 32, 16]} />
        <meshBasicMaterial
          color={cfg.fillColor}
          transparent
          opacity={cfg.fillOpacity}
          depthWrite={false}
          side={THREE.FrontSide}
        />
      </mesh>

      {/* Horizontaler Äquator-Ring — markiert die Reichweiten-Grenze */}
      <mesh rotation-x={Math.PI / 2}>
        <torusGeometry args={[radius, 0.3, 8, 80]} />
        <meshBasicMaterial
          color={cfg.ringColor}
          transparent
          opacity={cfg.ringOpacity}
          depthWrite={false}
          side={THREE.DoubleSide}
        />
      </mesh>

      {/* Aktiver Scan: Ping-Ring (Leuchtturm-Effekt) */}
      {scanning && <ScanPing radius={radius} />}

    </group>
  )
}
