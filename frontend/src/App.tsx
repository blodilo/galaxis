import { useEffect, useState, useCallback } from 'react'
import { GalaxyScene } from './three/GalaxyScene'
import { FilterPanel } from './components/FilterPanel'
import { Inspector } from './components/Inspector'
import { StatsPanel } from './components/StatsPanel'
import { fetchGalaxies, fetchAllStars, fetchNebulae } from './api/galaxy'
import type { Galaxy, Star, Nebula, StarFilter } from './types/galaxy'
import { DEFAULT_FILTER } from './types/galaxy'
import './index.css'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

export default function App() {
  const [galaxies, setGalaxies]   = useState<Galaxy[]>([])
  const [galaxy, setGalaxy]       = useState<Galaxy | null>(null)
  const [stars, setStars]         = useState<Star[]>([])
  const [nebulae, setNebulae]     = useState<Nebula[]>([])
  const [selected, setSelected]   = useState<Star | null>(null)
  const [filter, setFilter]       = useState<StarFilter>(DEFAULT_FILTER)
  const [loadState, setLoadState] = useState<LoadState>('idle')
  const [progress, setProgress]   = useState('')

  // Load galaxy list on mount
  useEffect(() => {
    fetchGalaxies()
      .then(gs => {
        setGalaxies(gs)
        if (gs.length > 0 && gs[0].status === 'ready') {
          loadGalaxy(gs[0])
        }
      })
      .catch(() => setLoadState('error'))
  }, [])

  const loadGalaxy = useCallback(async (g: Galaxy) => {
    setGalaxy(g)
    setLoadState('loading')
    setProgress('Lade Sterne…')
    try {
      const [s, n] = await Promise.all([
        fetchAllStars(g.id),
        fetchNebulae(g.id),
      ])
      setStars(s)
      setNebulae(n)
      setProgress('')
      setLoadState('ready')
    } catch {
      setLoadState('error')
    }
  }, [])

  return (
    <div className="relative w-screen h-screen bg-black overflow-hidden">

      {/* ── 3D Scene ── */}
      {loadState === 'ready' && (
        <GalaxyScene
          stars={stars}
          nebulae={nebulae}
          filter={filter}
          onSelectStar={setSelected}
        />
      )}

      {/* ── Loading overlay ── */}
      {loadState === 'loading' && (
        <div className="absolute inset-0 flex flex-col items-center justify-center bg-black/80 text-slate-400 gap-3">
          <div className="w-8 h-8 border-2 border-slate-600 border-t-blue-400 rounded-full animate-spin" />
          <span className="text-sm">{progress}</span>
        </div>
      )}

      {/* ── No galaxy ── */}
      {loadState === 'idle' && galaxies.length === 0 && (
        <div className="absolute inset-0 flex flex-col items-center justify-center text-slate-500">
          <p className="text-lg">Keine Galaxie gefunden.</p>
          <p className="text-sm mt-2 font-mono">make db-up && DATABASE_URL=… make run-gen</p>
        </div>
      )}

      {/* ── Galaxy selector (if multiple) ── */}
      {galaxies.length > 1 && (
        <div className="absolute top-3 left-1/2 -translate-x-1/2 z-20 flex gap-2">
          {galaxies.map(g => (
            <button
              key={g.id}
              onClick={() => loadGalaxy(g)}
              className={`px-3 py-1 text-xs rounded border transition-colors
                ${g.id === galaxy?.id
                  ? 'border-blue-500 text-blue-300 bg-blue-900/30'
                  : 'border-slate-700 text-slate-400 hover:border-slate-500'}`}
            >
              {g.name}
            </button>
          ))}
        </div>
      )}

      {/* ── Header bar ── */}
      <div className="absolute top-0 left-0 right-0 h-10 z-10 flex items-center px-4 gap-3
                      bg-black/60 border-b border-slate-800 backdrop-blur-sm">
        <span className="text-xs tracking-[0.3em] text-slate-400 uppercase font-semibold">
          Galaxis
        </span>
        <span className="text-slate-700">|</span>
        <span className="text-xs text-red-500 font-bold tracking-widest">GOD MODE</span>
        {galaxy && (
          <>
            <span className="text-slate-700">|</span>
            <span className="text-xs text-slate-400">{galaxy.name}</span>
            <span className="text-xs text-slate-600">
              {stars.length.toLocaleString('de-DE')} Sterne
            </span>
          </>
        )}
        <div className="ml-auto text-xs text-slate-600">
          Drag: Orbit · Scroll: Zoom · Klick: Inspektor
        </div>
      </div>

      {/* ── Left sidebar: Filter + Stats ── */}
      <div className="absolute top-10 left-0 bottom-0 w-52 z-10
                      bg-black/70 border-r border-slate-800 backdrop-blur-sm
                      overflow-y-auto p-3 flex flex-col gap-5">
        <StatsPanel galaxy={galaxy} stars={stars} />
        <div className="border-t border-slate-800" />
        <FilterPanel filter={filter} onChange={setFilter} />
      </div>

      {/* ── Right sidebar: Inspector ── */}
      <div className="absolute top-10 right-0 bottom-0 w-60 z-10
                      bg-black/70 border-l border-slate-800 backdrop-blur-sm
                      overflow-y-auto p-3">
        <Inspector star={selected} />
      </div>

    </div>
  )
}
