import { Suspense } from 'react'
import { Canvas, useThree } from '@react-three/fiber'
import { OrbitControls, Stars as BackgroundStars } from '@react-three/drei'
import { EffectComposer, Bloom, ChromaticAberration } from '@react-three/postprocessing'
import { BlendFunction } from 'postprocessing'
import * as THREE from 'three'
import { StarField } from './StarField'
import { NebulaLayer } from './NebulaLayer'
import { useVisualParams } from '../context/VisualParamsContext'
import type { Star, Nebula, StarFilter } from '../types/galaxy'

// Syncs gl.toneMappingExposure reactively (Canvas gl prop is mount-only)
function ExposureSync() {
  const { gl } = useThree()
  const { params } = useVisualParams()
  gl.toneMappingExposure = params.exposure
  return null
}

interface Props {
  stars: Star[]
  nebulae: Nebula[]
  filter: StarFilter
  onSelectStar: (star: Star) => void
}

export function GalaxyScene({ stars, nebulae, filter, onSelectStar }: Props) {
  const { params } = useVisualParams()

  return (
    <Canvas
      camera={{ position: [0, 800, 0], up: [0, 0, -1], fov: 60 }}
      gl={{ antialias: true, toneMapping: THREE.ACESFilmicToneMapping, toneMappingExposure: params.exposure }}
      style={{ background: '#000005' }}
    >
      <Suspense fallback={null}>
        <ExposureSync />
        <BackgroundStars radius={1500} depth={200} count={3000} factor={0.5} fade />
        <NebulaLayer nebulae={nebulae} filter={filter} />
        <StarField stars={stars} filter={filter} onSelect={onSelectStar} />

        <EffectComposer>
          <Bloom
            intensity={params.bloomIntensity}
            luminanceThreshold={params.bloomThreshold}
            luminanceSmoothing={params.bloomSmoothing}
            mipmapBlur
          />
          <ChromaticAberration
            blendFunction={BlendFunction.NORMAL}
            offset={new THREE.Vector2(0.0005, 0.0005)}
          />
        </EffectComposer>

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
