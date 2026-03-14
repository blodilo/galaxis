import { Suspense } from 'react'
import { Canvas } from '@react-three/fiber'
import { OrbitControls, Stars as BackgroundStars } from '@react-three/drei'
import { EffectComposer, Bloom, ChromaticAberration } from '@react-three/postprocessing'
import { BlendFunction } from 'postprocessing'
import * as THREE from 'three'
import { StarField } from './StarField'
import { NebulaLayer } from './NebulaLayer'
import type { Star, Nebula, StarFilter } from '../types/galaxy'

interface Props {
  stars: Star[]
  nebulae: Nebula[]
  filter: StarFilter
  onSelectStar: (star: Star) => void
}

export function GalaxyScene({ stars, nebulae, filter, onSelectStar }: Props) {
  return (
    <Canvas
      camera={{ position: [0, 800, 0], up: [0, 0, -1], fov: 60 }}
      gl={{ antialias: true, toneMapping: THREE.ACESFilmicToneMapping, toneMappingExposure: 1.2 }}
      style={{ background: '#000005' }}
    >
      <Suspense fallback={null}>
        {/* Deep-space background star haze */}
        <BackgroundStars radius={1500} depth={200} count={3000} factor={0.5} fade />

        {/* Nebulae (rendered first — additive blending, no depth write) */}
        <NebulaLayer nebulae={nebulae} filter={filter} />

        {/* Main star field */}
        <StarField stars={stars} filter={filter} onSelect={onSelectStar} />

        {/* Post-processing */}
        <EffectComposer>
          <Bloom
            intensity={1.4}
            luminanceThreshold={0.05}
            luminanceSmoothing={0.9}
            mipmapBlur
          />
          <ChromaticAberration
            blendFunction={BlendFunction.NORMAL}
            offset={new THREE.Vector2(0.0005, 0.0005)}
          />
        </EffectComposer>

        {/* Camera controls */}
        <OrbitControls
          makeDefault
          enablePan
          enableZoom
          enableRotate
          minDistance={10}
          maxDistance={2000}
          zoomSpeed={1.5}
        />
      </Suspense>
    </Canvas>
  )
}
