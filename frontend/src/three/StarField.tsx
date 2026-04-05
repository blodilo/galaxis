import { useMemo, useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import type { Star, StarFilter } from '../types/galaxy'
import { useVisualParams } from '../context/VisualParamsContext'
import { DEFAULT_VISUAL_PARAMS } from '../config/visualParams'

// Scale factor: 1 Three.js unit = 100 ly
const LY = 100

interface Props {
  stars: Star[]
  filter: StarFilter
  onSelect: (star: Star) => void
}

export function StarField({ stars, filter, onSelect }: Props) {
  const meshRef = useRef<THREE.Points>(null!)
  const pulsarPhase = useRef(0)
  const { params } = useVisualParams()

  const { geometry, starIndex } = useMemo(() => {
    const visible = stars.filter(s =>
      filter[s.star_type as keyof typeof filter] !== false &&
      (!filter.onlyWithPlanets || s.planets_generated)
    )

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

      const baseSize = DEFAULT_VISUAL_PARAMS.typeSizes[s.star_type] ?? 1.0
      sizes[i] = baseSize
    })

    const geo = new THREE.BufferGeometry()
    geo.setAttribute('position', new THREE.BufferAttribute(positions, 3))
    geo.setAttribute('color',    new THREE.BufferAttribute(colors, 3))
    geo.setAttribute('size',     new THREE.BufferAttribute(sizes, 1))

    const idx = new Map<number, Star>()
    visible.forEach((s, i) => idx.set(i, s))

    return { geometry: geo, starIndex: idx }
  }, [stars, filter])

  // Shader material with uniforms — no rebuild on param change
  const material = useMemo(() => new THREE.ShaderMaterial({
    vertexColors: true,
    transparent: true,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
    uniforms: {
      uSizeScale:   { value: params.starSizeScale },
      uSizeCap:     { value: params.starSizeCap },
      uPointScale:  { value: params.starPointScale },
      uGaussian:    { value: params.starGaussian },
    },
    vertexShader: `
      attribute float size;
      varying vec3 vColor;
      uniform float uSizeScale;
      uniform float uSizeCap;
      uniform float uPointScale;
      void main() {
        vColor = color;
        vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
        float s = size * uSizeScale * (uPointScale / -mvPosition.z);
        gl_PointSize = min(s, uSizeCap);
        gl_Position = projectionMatrix * mvPosition;
      }
    `,
    fragmentShader: `
      varying vec3 vColor;
      uniform float uGaussian;
      void main() {
        vec2 uv = gl_PointCoord - 0.5;
        float d = length(uv);
        if (d > 0.5) discard;
        float alpha = exp(-d * d * uGaussian);
        gl_FragColor = vec4(vColor, alpha);
      }
    `,
  }), []) // created once

  // Sync uniforms every frame — zero allocation, no material rebuild
  useFrame((_, delta) => {
    if (!material) return
    material.uniforms.uSizeScale.value  = params.starSizeScale
    material.uniforms.uSizeCap.value    = params.starSizeCap
    material.uniforms.uPointScale.value = params.starPointScale
    material.uniforms.uGaussian.value   = params.starGaussian

    // Per-type sizes: update size attribute when typeSizes change
    const sizeAttr = geometry.getAttribute('size') as THREE.BufferAttribute
    let needsUpdate = false

    pulsarPhase.current += delta * 2
    stars.forEach((s, i) => {
      const baseSize = params.typeSizes[s.star_type] ?? 1.0
      if (s.star_type === 'Pulsar') {
        const pulse = baseSize + Math.sin(pulsarPhase.current * 3 + i) * baseSize * 0.4
        sizeAttr.setX(i, pulse)
        needsUpdate = true
      } else {
        const cur = sizeAttr.getX(i)
        if (Math.abs(cur - baseSize) > 0.001) {
          sizeAttr.setX(i, baseSize)
          needsUpdate = true
        }
      }
    })
    if (needsUpdate) sizeAttr.needsUpdate = true
  })

  const handleClick = (e: any) => {
    e.stopPropagation()
    const star = starIndex.get(e.index)
    if (star) onSelect(star)
  }

  return (
    <points ref={meshRef} geometry={geometry} material={material} onClick={handleClick} />
  )
}
