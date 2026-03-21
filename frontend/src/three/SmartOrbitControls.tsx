import { useRef, useMemo } from 'react'
import { useFrame } from '@react-three/fiber'
import { OrbitControls } from '@react-three/drei'
import * as THREE from 'three'

// Typ der drei OrbitControls-Instanz (duck-typed, kein three-stdlib-Import nötig)
type OC = {
  getDistance(): number
  target: THREE.Vector3
  maxDistance: number
  update(): void
}

interface Props {
  maxDistance: number    // Maximale Zoom-out-Distanz (in Scene-Einheiten)
  center?: THREE.Vector3 // Ziel beim Zentrieren (Standard: Ursprung)
  zoomSpeed?: number
  rotateSpeed?: number
}

/**
 * OrbitControls mit zwei Zusatz-Verhaltensweisen:
 * 1. Zoom-out stoppt bei maxDistance — alle Objekte bleiben im Viewport.
 * 2. Sobald die Kamera nahe an maxDistance ist, gleitet das Orbit-Target
 *    sanft zurück zum Ursprung (Sonne / Planet), statt weiter hinauszuzoomen.
 */
export function SmartOrbitControls({
  maxDistance,
  center,
  zoomSpeed  = 1.2,
  rotateSpeed = 0.6,
}: Props) {
  const ref    = useRef<OC | null>(null)
  const origin = useMemo(() => center ?? new THREE.Vector3(0, 0, 0), [center])

  useFrame((_, delta) => {
    const c = ref.current
    if (!c) return
    // Wenn Kamera 97 % des Limits erreicht → Target Richtung Ursprung ziehen
    if (c.getDistance() > maxDistance * 0.97) {
      // Frame-rate-unabhängiges Lerp: Halbwertszeit ~0.8 s
      const t = 1 - Math.pow(0.04, delta)
      c.target.lerp(origin, t)
      c.update()
    }
  })

  return (
    <OrbitControls
      ref={ref as React.RefObject<any>}
      makeDefault
      enablePan
      enableZoom
      enableRotate
      maxDistance={maxDistance}
      zoomSpeed={zoomSpeed}
      rotateSpeed={rotateSpeed}
    />
  )
}
