/**
 * TacticalGrid — perspektivisches Referenzgitter für die CIC-Taktikansicht
 *
 * Features:
 *  - Referenzebene (Y=0) mit gleichmäßigem Raster
 *  - Gravity-Well-Verzerrung: Gitterlinien biegen sich am Zentrum nach unten
 *    (visuelles Modell der Treibstoffkosten-Topologie, ADR-012 / ui-analyse 3.6)
 *  - Vertex-Farben: Übergang von grid-line (Außen) → grid-line-warped (Zentrum)
 *  - transparent, depthWrite: false → liegt unter Schiffen und Overlays
 *
 * Token-Referenz:
 *  --color-galaxis-grid-line        rgba(30,  80, 180, 0.35)
 *  --color-galaxis-grid-line-warped rgba(80,  30, 255, 0.50)
 */

import { useMemo } from 'react'
import * as THREE from 'three'

interface TacticalGridProps {
  /** Halbe Kantenlänge des Gitters in World-Units (Default: 100 → 200×200) */
  size?: number
  /** Anzahl Zellen pro Achse (Default: 20) */
  divisions?: number
  /** Maximale Y-Absenkung des Gravity Wells im Zentrum (Default: 10) */
  wellDepth?: number
  /** "Radius" des Well-Effekts — Abstand, bei dem die Absenkung auf ~50% fällt (Default: 28) */
  wellScale?: number
  /** Anzahl Unter-Segmente pro Gitterlinie für eine glatte Kurve (Default: 32) */
  subdivisions?: number
  /** Globale Opacity des Gitters (Default: 0.45) */
  opacity?: number
}

/** Berechnet die Y-Absenkung durch das Gravitationsfeld am Punkt (x, z). */
function gravityDisplacement(x: number, z: number, depth: number, scale: number): number {
  const r = Math.sqrt(x * x + z * z)
  return -depth / (r / scale + 1)
}

export function TacticalGrid({
  size        = 100,
  divisions   = 20,
  wellDepth   = 10,
  wellScale   = 28,
  subdivisions = 32,
  opacity     = 0.45,
}: TacticalGridProps) {

  const geometry = useMemo(() => {
    const half = size
    const cellStep = (half * 2) / divisions
    const subStep  = (half * 2) / subdivisions

    // Positionen der Gitterlinien auf der orthogonalen Achse
    const linePositions: number[] = []
    for (let i = 0; i <= divisions; i++) {
      linePositions.push(-half + i * cellStep)
    }

    // Farb-Referenzen (Three.js Color, ohne Alpha — alpha via Material.opacity)
    const colorNormal = new THREE.Color(0x1e50b4) // grid-line (30, 80, 180)
    const colorWarped = new THREE.Color(0x501eff) // grid-line-warped (80, 30, 255)
    const tmp = new THREE.Color()

    // Maximale Absenkung für Normierung der Farbinterpolation
    const maxDisp = Math.abs(gravityDisplacement(0, 0, wellDepth, wellScale))

    const positions: number[] = []
    const colors:    number[] = []

    const pushVertex = (x: number, z: number) => {
      const y = gravityDisplacement(x, z, wellDepth, wellScale)
      positions.push(x, y, z)

      // Farbinterpolation: 0 = Rand (normal), 1 = Zentrum (warped)
      const t = maxDisp > 0 ? Math.min(Math.abs(y) / maxDisp, 1) : 0
      // Quadratisch abfallen lassen — Effekt erst nahe dem Zentrum sichtbar
      tmp.lerpColors(colorNormal, colorWarped, t * t)
      colors.push(tmp.r, tmp.g, tmp.b)
    }

    // Linien parallel zur X-Achse (konstantes Z)
    for (const z of linePositions) {
      for (let j = 0; j < subdivisions; j++) {
        const x0 = -half + j       * subStep
        const x1 = -half + (j + 1) * subStep
        pushVertex(x0, z)
        pushVertex(x1, z)
      }
    }

    // Linien parallel zur Z-Achse (konstantes X)
    for (const x of linePositions) {
      for (let j = 0; j < subdivisions; j++) {
        const z0 = -half + j       * subStep
        const z1 = -half + (j + 1) * subStep
        pushVertex(x, z0)
        pushVertex(x, z1)
      }
    }

    const geo = new THREE.BufferGeometry()
    geo.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3))
    geo.setAttribute('color',    new THREE.Float32BufferAttribute(colors,    3))
    return geo
  }, [size, divisions, wellDepth, wellScale, subdivisions])

  return (
    <lineSegments geometry={geometry}>
      <lineBasicMaterial
        vertexColors
        transparent
        opacity={opacity}
        depthWrite={false}
      />
    </lineSegments>
  )
}
