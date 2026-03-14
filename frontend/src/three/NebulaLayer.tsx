import { useMemo } from 'react'
import * as THREE from 'three'
import type { Nebula, StarFilter } from '../types/galaxy'

const LY = 100

const NEBULA_COLORS: Record<string, string> = {
  HII:      '#ff2244',  // red hydrogen
  SNR:      '#22aaff',  // blue supernova remnant
  Globular: '#ffcc44',  // golden globular cluster
}

interface Props {
  nebulae: Nebula[]
  filter: StarFilter
}

function NebulaMesh({ nebula }: { nebula: Nebula }) {
  const color = NEBULA_COLORS[nebula.type] ?? '#ffffff'
  const r = nebula.radius_ly / LY

  const material = useMemo(() => new THREE.MeshBasicMaterial({
    color: new THREE.Color(color),
    transparent: true,
    opacity: nebula.density * 0.15,
    side: THREE.DoubleSide,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
  }), [color, nebula.density])

  return (
    <mesh
      position={[nebula.center_x / LY, nebula.center_y / LY, nebula.center_z / LY]}
      material={material}
    >
      <sphereGeometry args={[r, 16, 12]} />
    </mesh>
  )
}

export function NebulaLayer({ nebulae, filter }: Props) {
  const visible = nebulae.filter(n => {
    if (n.type === 'HII'      && !filter.HII)      return false
    if (n.type === 'SNR'      && !filter.SNR)      return false
    if (n.type === 'Globular' && !filter.Globular) return false
    return true
  })

  return (
    <group>
      {visible.map(n => <NebulaMesh key={n.id} nebula={n} />)}
    </group>
  )
}
