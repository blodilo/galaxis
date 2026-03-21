import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'
import {
  type VisualParams,
  DEFAULT_VISUAL_PARAMS,
  STORAGE_KEY,
} from '../config/visualParams'

function loadFromStorage(): VisualParams {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return DEFAULT_VISUAL_PARAMS
    const parsed = JSON.parse(raw)
    // Deep-merge so new defaults for added fields are respected
    return {
      ...DEFAULT_VISUAL_PARAMS,
      ...parsed,
      typeSizes: { ...DEFAULT_VISUAL_PARAMS.typeSizes, ...parsed.typeSizes },
    }
  } catch {
    return DEFAULT_VISUAL_PARAMS
  }
}

interface VisualParamsContextValue {
  params: VisualParams
  setParam: <K extends keyof VisualParams>(key: K, value: VisualParams[K]) => void
  setTypeSize: (type: string, value: number) => void
  resetSection: (section: 'postprocessing' | 'stars' | 'typesizes' | 'system' | 'layers') => void
  resetAll: () => void
}

const VisualParamsContext = createContext<VisualParamsContextValue | null>(null)

export function VisualParamsProvider({ children }: { children: ReactNode }) {
  const [params, setParams] = useState<VisualParams>(loadFromStorage)

  const persist = useCallback((next: VisualParams) => {
    setParams(next)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  }, [])

  const setParam = useCallback(<K extends keyof VisualParams>(key: K, value: VisualParams[K]) => {
    setParams(prev => {
      const next = { ...prev, [key]: value }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
      return next
    })
  }, [])

  const setTypeSize = useCallback((type: string, value: number) => {
    setParams(prev => {
      const next = { ...prev, typeSizes: { ...prev.typeSizes, [type]: value } }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
      return next
    })
  }, [])

  const resetSection = useCallback((section: 'postprocessing' | 'stars' | 'typesizes' | 'system' | 'layers') => {
    setParams(prev => {
      let next: VisualParams
      const d = DEFAULT_VISUAL_PARAMS
      switch (section) {
        case 'postprocessing':
          next = { ...prev, exposure: d.exposure, bloomIntensity: d.bloomIntensity, bloomThreshold: d.bloomThreshold, bloomSmoothing: d.bloomSmoothing }
          break
        case 'stars':
          next = { ...prev, starSizeScale: d.starSizeScale, starSizeCap: d.starSizeCap, starPointScale: d.starPointScale, starGaussian: d.starGaussian }
          break
        case 'typesizes':
          next = { ...prev, typeSizes: { ...d.typeSizes } }
          break
        case 'system':
          next = { ...prev, planetVisMax: d.planetVisMax, planetVisMin: d.planetVisMin, moonSizeFactor: d.moonSizeFactor, moonOrbitMin: d.moonOrbitMin, moonOrbitMax: d.moonOrbitMax }
          break
        case 'layers':
          next = { ...prev, layerOrbits: d.layerOrbits, layerAxisInfo: d.layerAxisInfo, layerOrbitalChevron: d.layerOrbitalChevron }
          break
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
      return next
    })
  }, [])

  const resetAll = useCallback(() => {
    persist({ ...DEFAULT_VISUAL_PARAMS, typeSizes: { ...DEFAULT_VISUAL_PARAMS.typeSizes } })
  }, [persist])

  return (
    <VisualParamsContext.Provider value={{ params, setParam, setTypeSize, resetSection, resetAll }}>
      {children}
    </VisualParamsContext.Provider>
  )
}

export function useVisualParams(): VisualParamsContextValue {
  const ctx = useContext(VisualParamsContext)
  if (!ctx) throw new Error('useVisualParams must be used within VisualParamsProvider')
  return ctx
}
