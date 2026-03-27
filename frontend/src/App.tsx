import { useEffect, useState, useCallback, useRef } from 'react'
import { GalaxyScene } from './three/GalaxyScene'
import { SystemScene } from './three/SystemScene'
import { MoonSystemScene } from './three/MoonSystemScene'
import { FilterPanel } from './components/FilterPanel'
import { Inspector } from './components/Inspector'
import { PlanetInspector } from './components/PlanetInspector'
import { StatsPanel } from './components/StatsPanel'
import { GalaxyPicker } from './components/GalaxyPicker'
import { SystemTree } from './components/SystemTree'
import { VisualTuner } from './components/VisualTuner'
import { GeneratorPage } from './pages/GeneratorPage'
import { HudDevPage } from './pages/HudDevPage'
import { Economy2Page } from './pages/Economy2Page'
import { VisualParamsProvider } from './context/VisualParamsContext'
import { fetchGalaxies, fetchAllStars, fetchNebulae, fetchSystem } from './api/galaxy'
import type { Galaxy, Star, Nebula, StarFilter, Planet } from './types/galaxy'
import { DEFAULT_FILTER } from './types/galaxy'
import './index.css'

type View = 'viewer' | 'generator' | 'hud-dev' | 'system' | 'moon' | 'economy' | 'start-conditions'
type LoadState = 'idle' | 'loading' | 'ready' | 'error'

const STAR_TYPE_LABELS: Record<string, string> = {
  O:'O-Stern', B:'B-Stern', A:'A-Stern', F:'F-Stern', G:'G-Stern', K:'K-Stern',
  M:'M-Stern', WR:'Wolf-Rayet', RStar:'Roter Überriese', SStar:'S-Stern',
  Pulsar:'Pulsar', StellarBH:'Schwarzes Loch', SMBH:'SMBH',
}

function AppInner() {
  const [view, setView]           = useState<View>('viewer')
  const [galaxies, setGalaxies]   = useState<Galaxy[]>([])
  const [galaxy, setGalaxy]       = useState<Galaxy | null>(null)
  const [stars, setStars]         = useState<Star[]>([])
  const [nebulae, setNebulae]     = useState<Nebula[]>([])
  const [selected, setSelected]         = useState<Star | null>(null)
  const [filter, setFilter]             = useState<StarFilter>(DEFAULT_FILTER)
  const [loadState, setLoadState]       = useState<LoadState>('idle')
  const [progress, setProgress]         = useState('')
  // System view
  const [systemStar, setSystemStar]     = useState<Star | null>(null)
  const [systemPlanets, setSystemPlanets] = useState<Planet[]>([])
  const [systemLoading, setSystemLoading] = useState(false)
  const [selectedPlanet, setSelectedPlanet] = useState<Planet | null>(null)
  // Galaxy picker
  const [pickerOpen, setPickerOpen]     = useState(false)
  const pickerAnchorRef = useRef<HTMLDivElement>(null)
  // Galaxy to resume in generator
  const [resumeGalaxy, setResumeGalaxy] = useState<Galaxy | null>(null)
  // Moon system view (BL-24: Doppelklick Planet → Mondsystem)
  const [moonPlanet, setMoonPlanet] = useState<Planet | null>(null)
  // Visual tuner panel
  const [tunerOpen, setTunerOpen] = useState(false)

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

  const refreshGalaxies = useCallback((completedId?: string) => {
    fetchGalaxies().then(gs => {
      setGalaxies(gs)
      if (completedId) {
        const found = gs.find(g => g.id === completedId)
        if (found && galaxy?.id === completedId) loadGalaxy(found)
      }
    }).catch(() => {})
  }, [galaxy?.id, loadGalaxy])

  // Load galaxy list on mount
  useEffect(() => {
    fetchGalaxies()
      .then(gs => {
        setGalaxies(gs)
        const ready = gs.find(g => g.status === 'ready')
        if (ready) loadGalaxy(ready)
      })
      .catch(() => setLoadState('error'))
  }, [])

  const handleViewSystem = useCallback(async (star: Star) => {
    if (!galaxy) return
    setSystemStar(star)
    setSystemPlanets([])
    setSelectedPlanet(null)
    setMoonPlanet(null)
    setSystemLoading(true)
    setView('system')
    try {
      const data = await fetchSystem(galaxy.id, star.id)
      setSystemPlanets(data.planets ?? [])
    } finally {
      setSystemLoading(false)
    }
  }, [galaxy])

  const handleDoubleClickPlanet = useCallback((p: Planet) => {
    if (p.planet_type === 'asteroid_belt') return
    setMoonPlanet(p)
    setView('moon')
  }, [])

  // Called by GeneratorPage when user clicks "Im Viewer anzeigen" → switch to viewer
  const handleViewGalaxy = useCallback((galaxyId: string) => {
    fetchGalaxies().then(gs => {
      setGalaxies(gs)
      const found = gs.find(g => g.id === galaxyId)
      if (found) {
        loadGalaxy(found)
        setView('viewer')
      }
    })
  }, [loadGalaxy])

  // Open picker → resume an in-progress galaxy in the generator
  const handleResume = useCallback((g: Galaxy) => {
    setResumeGalaxy(g)
    setView('generator')
  }, [])

  // Open generator for a new galaxy (clear any resume state)
  const handleNewGalaxy = useCallback(() => {
    setResumeGalaxy(null)
    setView('generator')
  }, [])

  // Switch to generator tab — auto-resume the first in-progress galaxy if not already resuming
  const handleSwitchToGenerator = useCallback(() => {
    if (!resumeGalaxy) {
      const inProgress = galaxies.find(g =>
        (g.status === 'morphology' || g.status === 'spectral' || g.status === 'objects')
      )
      if (inProgress) setResumeGalaxy(inProgress)
    }
    setView('generator')
  }, [galaxies, resumeGalaxy])

  return (
    <div className="relative w-screen h-screen bg-black overflow-hidden">

      {/* ── Header bar ── */}
      <div className="absolute top-0 left-0 right-0 h-10 z-10 flex items-center px-4 gap-3
                      bg-black/60 border-b border-slate-800 backdrop-blur-sm">
        <span className="text-xs tracking-[0.3em] text-slate-400 uppercase font-semibold">
          Galaxis
        </span>
        <span className="text-slate-700">|</span>

        {/* Tab buttons */}
        <button
          onClick={() => setView('viewer')}
          className={`text-xs font-bold tracking-widest px-2 py-0.5 rounded transition-colors
            ${view === 'viewer' || view === 'system' || view === 'moon' ? 'text-red-500' : 'text-slate-600 hover:text-slate-400'}`}
        >
          GOD MODE
        </button>
        <button
          onClick={handleSwitchToGenerator}
          className={`text-xs font-bold tracking-widest px-2 py-0.5 rounded transition-colors
            ${view === 'generator' ? 'text-blue-400' : 'text-slate-600 hover:text-slate-400'}`}
        >
          GENERATOR
        </button>
        <button
          onClick={() => setView('hud-dev')}
          className={`text-xs font-bold tracking-widest px-2 py-0.5 rounded transition-colors
            ${view === 'hud-dev' ? 'text-[var(--color-galaxis-cyan)]' : 'text-slate-600 hover:text-slate-400'}`}
        >
          HUD DEV
        </button>
        <button
          onClick={() => setView('economy')}
          className={`text-xs font-bold tracking-widest px-2 py-0.5 rounded transition-colors
            ${view === 'economy' ? 'text-emerald-400' : 'text-slate-600 hover:text-slate-400'}`}
        >
          ECONOMY
        </button>
        <button
          onClick={() => setView('start-conditions')}
          className={`text-xs font-bold tracking-widest px-2 py-0.5 rounded transition-colors
            ${view === 'start-conditions' ? 'text-orange-400' : 'text-slate-600 hover:text-slate-400'}`}
        >
          STARTBEDINGUNGEN
        </button>

        {(view === 'viewer' || view === 'system' || view === 'moon') && (
          <button
            onClick={() => setTunerOpen(o => !o)}
            className={`text-xs font-bold tracking-widest px-2 py-0.5 rounded transition-colors
              ${tunerOpen ? 'text-yellow-400' : 'text-slate-600 hover:text-slate-400'}`}
          >
            TUNING
          </button>
        )}

        <span className="text-slate-700">|</span>

        {/* ── Galaxy picker trigger ── */}
        <div className="relative" ref={pickerAnchorRef}>
          <button
            onClick={() => setPickerOpen(p => !p)}
            className={`flex items-center gap-1.5 text-xs px-2 py-0.5 rounded border transition-colors
              ${pickerOpen
                ? 'border-slate-500 text-slate-200 bg-slate-800'
                : 'border-slate-700 text-slate-400 hover:border-slate-500 hover:text-slate-200'}`}
          >
            <span>
              {galaxy ? galaxy.name : 'Galaxie wählen'}
            </span>
            <span className="text-[9px] text-slate-500">▾</span>
          </button>

          {pickerOpen && (
            <GalaxyPicker
              galaxies={galaxies}
              currentId={galaxy?.id ?? null}
              onSelect={(g) => { loadGalaxy(g); setView('viewer') }}
              onResume={handleResume}
              onNew={handleNewGalaxy}
              onClose={() => setPickerOpen(false)}
            />
          )}
        </div>

        {/* Breadcrumbs: Galaxie › Stern › Planet */}
        {(view === 'viewer' || view === 'system' || view === 'moon') && galaxy && (
          <div className="flex items-center gap-1 text-xs">
            <button
              onClick={() => setView('viewer')}
              className={`transition-colors ${
                view === 'viewer'
                  ? 'text-slate-300 cursor-default'
                  : 'text-slate-500 hover:text-slate-300'
              }`}
            >
              {galaxy.name}
            </button>
            {view === 'viewer' && (
              <span className="text-slate-700 ml-1">
                ({stars.length.toLocaleString('de-DE')} Sterne)
              </span>
            )}
            {(view === 'system' || view === 'moon') && systemStar && (
              <>
                <span className="text-slate-700">›</span>
                <button
                  onClick={() => setView('system')}
                  className={`transition-colors ${view === 'system' ? 'text-cyan-400 cursor-default' : 'text-slate-500 hover:text-cyan-400'}`}
                >
                  {STAR_TYPE_LABELS[systemStar.star_type] ?? systemStar.star_type}
                </button>
                {view === 'system' && selectedPlanet && (
                  <>
                    <span className="text-slate-700">›</span>
                    <span className="text-slate-300">
                      Planet {selectedPlanet.orbit_index + 1}
                    </span>
                  </>
                )}
                {view === 'moon' && moonPlanet && (
                  <>
                    <span className="text-slate-700">›</span>
                    <span className="text-teal-400">
                      Planet {moonPlanet.orbit_index + 1} · Mondsystem
                    </span>
                  </>
                )}
              </>
            )}
          </div>
        )}

        <div className="ml-auto text-xs text-slate-600">
          {view === 'viewer' && 'Drag: Orbit · Scroll: Zoom · Klick: Inspektor'}
          {view === 'system' && 'Drag: Orbit · Rechtsklick: Pan · Scroll: Zoom · Klick: Planet · Doppelklick: Mondsystem'}
          {view === 'moon'   && 'Drag: Orbit · Scroll: Zoom · ← Stern-Breadcrumb: zurück'}
        </div>
      </div>

      {/* ── VIEWER ── */}
      {view === 'viewer' && (
        <>
          {loadState === 'ready' && (
            <GalaxyScene
              stars={stars}
              nebulae={nebulae}
              filter={filter}
              onSelectStar={setSelected}
            />
          )}

          {loadState === 'loading' && (
            <div className="absolute inset-0 flex flex-col items-center justify-center bg-black/80 text-slate-400 gap-3">
              <div className="w-8 h-8 border-2 border-slate-600 border-t-blue-400 rounded-full animate-spin" />
              <span className="text-sm">{progress}</span>
            </div>
          )}

          {loadState === 'idle' && galaxies.length === 0 && (
            <div className="absolute inset-0 flex flex-col items-center justify-center text-slate-500">
              <p className="text-lg">Keine Galaxie gefunden.</p>
              <p className="text-sm mt-2">
                Wechsle in den{' '}
                <button onClick={() => setView('generator')} className="text-blue-400 underline">
                  Generator
                </button>
                , um eine zu erstellen.
              </p>
            </div>
          )}

          {/* Left sidebar: Filter + Stats */}
          <div className="absolute top-10 left-0 bottom-0 w-52 z-10
                          bg-black/70 border-r border-slate-800 backdrop-blur-sm
                          overflow-y-auto p-3 flex flex-col gap-5">
            <StatsPanel galaxy={galaxy} stars={stars} />
            <div className="border-t border-slate-800" />
            <FilterPanel filter={filter} onChange={setFilter} />
          </div>

          {/* Right sidebar: Inspector or Tuner */}
          {tunerOpen ? (
            <VisualTuner />
          ) : (
            <div className="absolute top-10 right-0 bottom-0 w-60 z-10
                            bg-black/70 border-l border-slate-800 backdrop-blur-sm
                            overflow-y-auto p-3">
              <Inspector star={selected} onViewSystem={handleViewSystem} />
            </div>
          )}
        </>
      )}

      {/* ── SYSTEM VIEW ── */}
      {view === 'system' && systemStar && (
        <>
          {systemLoading ? (
            <div className="absolute inset-0 flex items-center justify-center bg-black/80 text-slate-400 gap-3">
              <div className="w-8 h-8 border-2 border-slate-600 border-t-cyan-400 rounded-full animate-spin" />
              <span className="text-sm">Lade Planetensystem…</span>
            </div>
          ) : (
            <SystemScene
              star={systemStar}
              planets={systemPlanets}
              selectedPlanet={selectedPlanet}
              onSelectPlanet={setSelectedPlanet}
              onDoubleClickPlanet={handleDoubleClickPlanet}
            />
          )}

          {/* Left: Systembaum (BL-15) */}
          <div className="absolute top-10 left-0 bottom-0 w-52 z-10
                          bg-black/70 border-r border-slate-800 backdrop-blur-sm
                          overflow-y-auto p-3">
            <SystemTree
              star={systemStar}
              planets={systemPlanets}
              selectedPlanet={selectedPlanet}
              onSelectPlanet={setSelectedPlanet}
            />
          </div>

          {/* Right: Planetinspektor or Tuner */}
          {tunerOpen ? (
            <VisualTuner />
          ) : (
            <div className="absolute top-10 right-0 bottom-0 w-60 z-10
                            bg-black/70 border-l border-slate-800 backdrop-blur-sm
                            overflow-y-auto p-3">
              <PlanetInspector planet={selectedPlanet} starId={systemStar?.id} />
            </div>
          )}
        </>
      )}

      {/* ── MOON SYSTEM VIEW (BL-24) ── */}
      {view === 'moon' && moonPlanet && (
        <>
          <MoonSystemScene planet={moonPlanet} />

          {/* Right: Planetinspektor */}
          <div className="absolute top-10 right-0 bottom-0 w-60 z-10
                          bg-black/70 border-l border-slate-800 backdrop-blur-sm
                          overflow-y-auto p-3">
            <PlanetInspector planet={moonPlanet} />
          </div>
        </>
      )}

      {/* ── GENERATOR ── */}
      <div className={`absolute inset-0 top-10 bg-slate-950 ${view === 'generator' ? '' : 'hidden'}`}>
        <GeneratorPage
          resumeGalaxy={resumeGalaxy}
          onViewGalaxy={handleViewGalaxy}
          onGalaxiesChanged={refreshGalaxies}
        />
      </div>

      {/* ── HUD DEV TESTBED ── */}
      {view === 'hud-dev' && (
        <div className="absolute inset-0 top-10">
          <HudDevPage />
        </div>
      )}

      {/* ── ECONOMY ── */}
      {view === 'economy' && (
        <div className="absolute inset-0 top-10 bg-slate-950">
          <Economy2Page />
        </div>
      )}

      {/* ── STARTBEDINGUNGEN ── */}
      {view === 'start-conditions' && (
        <div className="absolute inset-0 top-10 bg-slate-950 flex flex-col items-center justify-center gap-4 text-slate-400">
          <span className="text-xs font-bold tracking-widest text-orange-400 uppercase">Startbedingungen editieren</span>
          <p className="text-sm text-slate-500 max-w-sm text-center">
            Hier werden alle Wirtschafts-Parameter konfigurierbar sein:
            Startressourcen, Anlagenausstattung, Lagerbestände, Survey-Qualität.
          </p>
          <p className="text-xs text-slate-700">(Post-MVP — noch nicht implementiert)</p>
          <p className="text-xs text-slate-600 mt-4">
            Heimatplanet per God Mode einrichten:{' '}
            <button onClick={() => setView('system')} className="text-cyan-400 underline">
              GOD MODE → Planet auswählen → "Heimatplaneten anlegen"
            </button>
          </p>
        </div>
      )}


    </div>
  )
}

export default function App() {
  return (
    <VisualParamsProvider>
      <AppInner />
    </VisualParamsProvider>
  )
}
