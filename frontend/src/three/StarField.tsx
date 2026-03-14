import { useMemo, useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import type { Star, StarFilter } from '../types/galaxy'

// Scale factor: 1 Three.js unit = 100 ly
const LY = 100

interface Props {
  stars: Star[]
  filter: StarFilter
  onSelect: (star: Star) => void
}

// Point sizes by star type (Three.js units at base scale)
const TYPE_SIZE: Record<string, number> = {
  SMBH: 8, StellarBH: 4, WR: 3,
  O: 2.5, B: 2.0, A: 1.8, F: 1.5,
  Pulsar: 2, RStar: 2.5, SStar: 2.2,
  G: 1.2, K: 1.0, M: 0.8,
}

export function StarField({ stars, filter, onSelect }: Props) {
  const meshRef = useRef<THREE.Points>(null!)
  const pulsarPhase = useRef(0)

  const { geometry, starIndex } = useMemo(() => {
    const visible = stars.filter(s => filter[s.star_type as keyof typeof filter] !== false)

    const positions = new Float32Array(visible.length * 3)
    const colors    = new Float32Array(visible.length * 3)
    const sizes     = new Float32Array(visible.length)

    visible.forEach((s, i) => {
      positions[i * 3]     = s.x / LY
      positions[i * 3 + 1] = s.y / LY
      positions[i * 3 + 2] = s.z / LY

      const col = new THREE.Color(s.color_hex || '#ffffff')
      colors[i * 3]     = col.r
      colors[i * 3 + 1] = col.g
      colors[i * 3 + 2] = col.b

      sizes[i] = (TYPE_SIZE[s.star_type] ?? 1.0) * 1.5
    })

    const geo = new THREE.BufferGeometry()
    geo.setAttribute('position', new THREE.BufferAttribute(positions, 3))
    geo.setAttribute('color',    new THREE.BufferAttribute(colors, 3))
    geo.setAttribute('size',     new THREE.BufferAttribute(sizes, 1))

    // Map from geometry index → original star for raycasting
    const idx = new Map<number, Star>()
    visible.forEach((s, i) => idx.set(i, s))

    return { geometry: geo, starIndex: idx }
  }, [stars, filter])

  // Pulsate pulsars by modulating their size each frame
  useFrame((_, delta) => {
    pulsarPhase.current += delta * 2
    const sizeAttr = geometry.getAttribute('size') as THREE.BufferAttribute
    stars.forEach((s, i) => {
      if (s.star_type === 'Pulsar') {
        const base = TYPE_SIZE['Pulsar'] * 1.5
        sizeAttr.setX(i, base + Math.sin(pulsarPhase.current * 3 + i) * base * 0.4)
      }
    })
    sizeAttr.needsUpdate = true
  })

  const material = useMemo(() => new THREE.ShaderMaterial({
    vertexColors: true,
    transparent: true,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
    vertexShader: `
      attribute float size;
      varying vec3 vColor;
      void main() {
        vColor = color;
        vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
        gl_PointSize = size * (600.0 / -mvPosition.z);
        gl_Position = projectionMatrix * mvPosition;
      }
    `,
    fragmentShader: `
      varying vec3 vColor;
      void main() {
        // Circular soft disc with glow falloff
        vec2 uv = gl_PointCoord - 0.5;
        float d = length(uv);
        if (d > 0.5) discard;
        float alpha = 1.0 - smoothstep(0.1, 0.5, d);
        gl_FragColor = vec4(vColor, alpha);
      }
    `,
  }), [])

  const handleClick = (e: any) => {
    e.stopPropagation()
    const idx = e.index
    const star = starIndex.get(idx)
    if (star) onSelect(star)
  }

  return (
    <points ref={meshRef} geometry={geometry} material={material} onClick={handleClick} />
  )
}
