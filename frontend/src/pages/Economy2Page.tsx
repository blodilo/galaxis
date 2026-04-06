import { useState, useEffect, useCallback, useRef } from 'react'
import { useNatsTick } from '../hooks/useNatsTick'
import type { Recipe, Route, AggregatedStock, Goal } from '../types/economy2'
import type { Facility, Order } from '../types/economy2'
import {
  listRecipes,
  listMyNodes,
  listGoals,
  stockAll as fetchStockAll,
  facilitiesAll,
  ordersAll,
  listRoutes,
  bootstrap,
  startFacility as startFacilityApi,
  stopFacility as stopFacilityApi,
} from '../api/economy2'
import type { MyNodeEntry } from '../api/economy2'
import { Spinner } from '../components/economy2/ui'
import LeftRail from '../components/economy2/LeftRail'
import PlanTab from '../components/economy2/PlanTab'
import FabrikenTab from '../components/economy2/FabrikenTab'
import NetzwerkTab from '../components/economy2/NetzwerkTab'

// ── Tick generator ────────────────────────────────────────────────────────────

const TICK_MIN = 0.1
const TICK_MAX = 100

interface TickGeneratorProps {
  speed: number
  onSetSpeed: (updater: (s: number) => number) => void
  running: boolean
  onToggle: () => void
  currentTick: number | null
}

function TickGenerator({ speed, onSetSpeed, running, onToggle, currentTick }: TickGeneratorProps) {
  function faster() { onSetSpeed(s => Math.min(TICK_MAX, +(s * 10).toPrecision(4))) }
  function slower() { onSetSpeed(s => Math.max(TICK_MIN, +(s / 10).toPrecision(4))) }
  const speedLabel = speed < 1 ? speed.toFixed(1) : speed >= 10 ? speed.toFixed(0) : speed.toString()

  return (
    <div className="flex items-center gap-1.5 text-xs font-mono select-none">
      <span className="text-slate-600">Tick</span>
      <span className="text-slate-400 w-8 text-right">
        {currentTick !== null ? `#${currentTick}` : '—'}
      </span>
      <span className="text-slate-700 mx-0.5">|</span>
      <button
        onClick={slower}
        disabled={speed <= TICK_MIN}
        className="px-1.5 py-0.5 rounded border border-slate-700 text-slate-400
                   hover:border-slate-500 disabled:opacity-30 transition-colors"
        title="Langsamer"
      >/10</button>
      <span className="text-slate-300 w-12 text-center">{speedLabel}/s</span>
      <button
        onClick={faster}
        disabled={speed >= TICK_MAX}
        className="px-1.5 py-0.5 rounded border border-slate-700 text-slate-400
                   hover:border-slate-500 disabled:opacity-30 transition-colors"
        title="Schneller"
      >×10</button>
      <button
        onClick={onToggle}
        className={`px-2 py-0.5 rounded border font-bold transition-colors ${
          running
            ? 'border-orange-700 text-orange-400 hover:bg-orange-900/30'
            : 'border-emerald-700 text-emerald-400 hover:bg-emerald-900/30'
        }`}
      >{running ? '⏹' : '▶'}</button>
    </div>
  )
}

// ── Star type labels ──────────────────────────────────────────────────────────

const STAR_TYPE_LABELS: Record<string, string> = {
  O:'O-Stern', B:'B-Stern', A:'A-Stern', F:'F-Stern', G:'G-Stern', K:'K-Stern',
  M:'M-Stern', WR:'Wolf-Rayet', RStar:'Roter Überriese', SStar:'S-Stern',
  Pulsar:'Pulsar', StellarBH:'Schwarzes Loch', SMBH:'SMBH',
}

// ── Assets Overview ───────────────────────────────────────────────────────────

function MyAssetsView({ onSelect }: { onSelect: (node: MyNodeEntry) => void }) {
  const [nodes, setNodes] = useState<MyNodeEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    listMyNodes()
      .then(setNodes)
      .catch(e => setError(e instanceof Error ? e.message : 'Ladefehler'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return (
    <div className="flex-1 flex items-center justify-center text-slate-500 text-sm">
      Lade Assets…
    </div>
  )

  if (error) return (
    <div className="flex-1 flex items-center justify-center text-red-400 text-sm">{error}</div>
  )

  if (nodes.length === 0) return (
    <div className="flex-1 flex flex-col items-center justify-center gap-3 text-slate-500">
      <span className="text-2xl">⬡</span>
      <p className="text-sm">Keine Kolonien vorhanden.</p>
      <p className="text-xs text-slate-600">
        God Mode → Stern auswählen → Planet → "Heimatplaneten anlegen"
      </p>
    </div>
  )

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <h2 className="text-xs font-bold tracking-widest text-slate-500 uppercase mb-4">
        Meine Assets — {nodes.length} Kolonie{nodes.length !== 1 ? 'n' : ''}
      </h2>
      <div className="grid gap-2" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))' }}>
        {nodes.map(n => (
          <button
            key={n.node_id}
            onClick={() => onSelect(n)}
            className="text-left p-3 rounded border border-slate-800 bg-black/40
                       hover:border-emerald-700 hover:bg-emerald-900/10 transition-colors group"
          >
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm font-bold text-slate-200 group-hover:text-emerald-400 transition-colors">
                {STAR_TYPE_LABELS[n.star_type] ?? n.star_type}
              </span>
              <span className="text-xs text-slate-500 uppercase">{n.level}</span>
            </div>
            <div className="text-xs text-slate-600 font-mono mb-2">
              {n.star_id.slice(0, 8)}…
            </div>
            <div className="flex items-center gap-3 text-xs">
              <span className={n.facility_count > 0 ? 'text-emerald-500' : 'text-slate-600'}>
                {n.facility_count} Anlage{n.facility_count !== 1 ? 'n' : ''}
              </span>
              <span className="text-slate-700">
                ({Math.round(n.x / 1000)}k, {Math.round(n.y / 1000)}k ly)
              </span>
            </div>
          </button>
        ))}
      </div>
    </div>
  )
}

// ── Main Economy2Page ─────────────────────────────────────────────────────────

type TabId = 'plan' | 'fabriken' | 'netzwerk'

interface PageData {
  nodes: MyNodeEntry[]
  stock: AggregatedStock[]
  facilities: Facility[]
  orders: Order[]
  recipes: Recipe[]
  goals: Goal[]
  routes: Route[]
}

export function Economy2Page() {
  const [activeNode, setActiveNode] = useState<MyNodeEntry | null>(null)
  const [activeTab, setActiveTab] = useState<TabId>('plan')
  const [activeGoalId, setActiveGoalId] = useState<string | null>(null)
  const [data, setData] = useState<PageData | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [bootstrapping, setBootstrapping] = useState(false)
  const [bootstrapMsg, setBootstrapMsg] = useState('')
  const [currentTick, setCurrentTick] = useState<number | null>(null)
  const [tickSpeed, setTickSpeed] = useState(1)
  const [tickRunning, setTickRunning] = useState(false)

  const activeNodeRef = useRef(activeNode)
  activeNodeRef.current = activeNode

  const loadData = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [nodes, stock, facilities, orders, recipes, goals, routes] = await Promise.all([
        listMyNodes(),
        fetchStockAll(),
        facilitiesAll(),
        ordersAll(),
        listRecipes(),
        listGoals(),
        listRoutes(),
      ])
      setData({ nodes, stock, facilities, orders, recipes, goals, routes })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ladefehler')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (activeNode) {
      loadData()
    }
  }, [activeNode, loadData])

  // NATS live updates
  const natsStatus = useNatsTick((tickN) => {
    setCurrentTick(tickN)
    if (activeNodeRef.current) {
      loadData()
    }
  })

  // Tick generator — POST advance on interval when running
  const tickSpeedRef = useRef(tickSpeed)
  tickSpeedRef.current = tickSpeed
  const tickRunningRef = useRef(tickRunning)
  tickRunningRef.current = tickRunning

  useEffect(() => {
    if (!tickRunning) return
    const ms = Math.round(1000 / tickSpeed)
    const id = setInterval(async () => {
      try {
        await fetch('/api/v2/admin/tick/advance', { method: 'POST' })
      } catch {
        // ignore — server not reachable
      }
    }, ms)
    return () => clearInterval(id)
  }, [tickRunning, tickSpeed])

  async function handleBootstrap() {
    if (!activeNode) return
    setBootstrapping(true)
    setBootstrapMsg('')
    setError('')
    try {
      const result = await bootstrap(activeNode.star_id)
      setBootstrapMsg(`Kit gesetzt: ${result.seeded_facilities} Anlagen, ${Object.keys(result.seeded_stock).length} Güter`)
      await loadData()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Bootstrap fehlgeschlagen')
    } finally {
      setBootstrapping(false)
    }
  }

  async function handleStartFacility(facility: Facility) {
    try {
      await startFacilityApi(facility.id)
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Fehler'
      setError(msg.includes('409') ? 'Kein passender Auftrag verfügbar — erstelle einen im PLAN-Tab.' : msg)
    }
    loadData()
  }

  async function handleStopFacility(facility: Facility) {
    try {
      await stopFacilityApi(facility.id)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fehler')
    }
    loadData()
  }

  return (
    <div className="absolute inset-0 top-0 flex flex-col font-mono bg-slate-950">
      {/* Topbar */}
      <div className="flex items-center gap-3 px-4 py-2 bg-black/70 border-b border-slate-800 backdrop-blur-sm flex-shrink-0">
        {!activeNode ? (
          <span className="text-xs font-bold tracking-widest text-emerald-500 uppercase">Meine Assets</span>
        ) : (
          <>
            <button
              onClick={() => { setActiveNode(null); setData(null) }}
              className="text-xs text-slate-500 hover:text-slate-300 transition-colors mr-1"
              title="Zurück zur Übersicht"
            >← Assets</button>
            <span className="text-xs text-slate-700">|</span>
            <span className="text-xs text-slate-400 font-mono">
              {STAR_TYPE_LABELS[activeNode.star_type] ?? activeNode.star_type} · {activeNode.star_id.slice(0, 8)}…
            </span>
            <span className="text-xs text-slate-700">|</span>
            {/* Tab nav */}
            {(['plan', 'fabriken', 'netzwerk'] as TabId[]).map(tab => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`text-xs px-2 py-0.5 rounded transition-colors ${
                  activeTab === tab
                    ? 'text-emerald-400 bg-emerald-900/30 border border-emerald-800'
                    : 'text-slate-500 hover:text-slate-300'
                }`}
              >
                {tab.toUpperCase()}
              </button>
            ))}
          </>
        )}

        <div className="ml-auto flex items-center gap-3">
          <TickGenerator
            speed={tickSpeed}
            onSetSpeed={setTickSpeed}
            running={tickRunning}
            onToggle={() => setTickRunning(r => !r)}
            currentTick={currentTick}
          />
          {activeNode && (
            <>
              <button
                onClick={handleBootstrap}
                disabled={bootstrapping}
                className="text-xs px-2 py-0.5 rounded border border-blue-700 text-blue-400
                           hover:bg-blue-900/30 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
              >
                {bootstrapping ? '…' : '⬡ Spielstart-Kit'}
              </button>
              {natsStatus === 'live'
                ? <span className="text-xs text-emerald-600">● Live</span>
                : natsStatus === 'connecting'
                  ? <span className="text-xs text-slate-600">○ Verbinde…</span>
                  : <span className="text-xs text-amber-600" title="NATS nicht erreichbar — kein Auto-Update">⚠ Offline</span>
              }
            </>
          )}
          {loading && <Spinner />}
        </div>

        {bootstrapMsg && <span className="text-xs text-blue-400 ml-2">{bootstrapMsg}</span>}
        {error && <span className="text-xs text-red-400 ml-2">{error}</span>}
      </div>

      {/* Content */}
      {!activeNode ? (
        <MyAssetsView onSelect={n => { setActiveNode(n); setActiveTab('plan') }} />
      ) : !data ? (
        <div className="flex-1 flex items-center justify-center">
          <Spinner />
        </div>
      ) : (
        <div className="flex flex-1 overflow-hidden">
          <LeftRail
            goals={data.goals}
            stockAll={data.stock}
            activeGoalId={activeGoalId}
            onSelectGoal={id => { setActiveGoalId(id); setActiveTab('plan') }}
            onRefresh={loadData}
          />

          <main className="flex-1 overflow-auto">
            {activeTab === 'plan' && (
              <PlanTab
                goals={data.goals}
                recipes={data.recipes}
                stockAll={data.stock}
                facilities={data.facilities}
                orders={data.orders}
                routes={data.routes}
                nodes={data.nodes}
                onRefresh={loadData}
              />
            )}
            {activeTab === 'fabriken' && (
              <FabrikenTab
                nodes={data.nodes}
                facilities={data.facilities}
                orders={data.orders}
                recipes={data.recipes}
                stockAll={data.stock}
                onStartFacility={handleStartFacility}
                onStopFacility={handleStopFacility}
                onRefresh={loadData}
              />
            )}
            {activeTab === 'netzwerk' && (
              <NetzwerkTab
                nodes={data.nodes}
                routes={data.routes}
                stockAll={data.stock}
                onRefresh={loadData}
              />
            )}
          </main>
        </div>
      )}
    </div>
  )
}
