import { useEffect, useState, useCallback, useRef } from 'react'
import {
  fetchSystemState, fetchLog, buildFacility, assignRecipe,
  advanceTick, openTickStream,
} from '../api/economy'
import type { SystemState, Facility, LogRow, LogEvent, TickEvent } from '../api/economy'

interface Props {
  starId: string
}

// ── Helpers ──────────────────────────────────────────────────────────────────

const FACILITY_LABELS: Record<string, string> = {
  mine: 'Bergwerk',
  smelter: 'Schmelze',
  refinery: 'Raffinerie',
  bioreaktor: 'Bioreaktor',
  precision_factory: 'Präzisionsfabrik',
  assembler: 'Assembler',
  shipyard: 'Werft',
  elevator: 'Aufzug',
}

const STATUS_COLORS: Record<string, string> = {
  idle:            'text-slate-500',
  running:         'text-emerald-400',
  paused_input:    'text-yellow-400',
  paused_output:   'text-orange-400',
  paused_depleted: 'text-red-400',
}

const EVENT_COLORS: Record<string, string> = {
  mined:             'text-cyan-400',
  produced:          'text-emerald-400',
  paused_input:      'text-yellow-400',
  paused_output:     'text-orange-400',
  deposit_depleted:  'text-red-500',
  deposit_warning:   'text-yellow-500',
  deposit_critical:  'text-red-400',
}

function eventLabel(ev: LogEvent): string {
  switch (ev.type) {
    case 'mined':            return `Gefördert: ${ev.qty} × ${ev.good}`
    case 'produced':         return `Produziert: ${ev.qty} × ${ev.good}`
    case 'paused_input':     return `Pause (Input): ${ev.missing} fehlt`
    case 'paused_output':    return `Pause (Output): Lager voll`
    case 'deposit_depleted': return `Deposit erschöpft: ${ev.good}`
    case 'deposit_warning':  return `Deposit niedrig (20%): ${ev.good}`
    case 'deposit_critical': return `Deposit kritisch (5%): ${ev.good}`
    default:                 return ev.type
  }
}

// ── Sub-components ───────────────────────────────────────────────────────────

function TickControls({
  tickN,
  sseOk,
  onAdvance,
}: {
  tickN: number
  sseOk: boolean
  onAdvance: () => void
}) {
  return (
    <div className="flex items-center gap-4 px-4 py-2 bg-slate-900 border border-slate-700 rounded">
      <span className="text-xs text-slate-400">Tick</span>
      <span className="text-lg font-mono text-white">{tickN}</span>
      <button
        onClick={onAdvance}
        className="text-xs font-bold px-3 py-1 bg-slate-700 hover:bg-slate-600 text-white rounded transition-colors"
      >
        ▶ Advance
      </button>
      <span className={`text-[10px] font-mono ${sseOk ? 'text-emerald-400' : 'text-red-400'}`}>
        {sseOk ? '● SSE' : '○ SSE'}
      </span>
    </div>
  )
}

function OrbitalSlotsBar({ used, max }: { used: number; max: number }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-slate-400">Orbital-Slots</span>
      <div className="flex gap-0.5">
        {Array.from({ length: max }).map((_, i) => (
          <div
            key={i}
            className={`w-3 h-3 rounded-sm border ${
              i < used ? 'bg-cyan-500 border-cyan-400' : 'bg-slate-800 border-slate-600'
            }`}
          />
        ))}
      </div>
      <span className="text-xs text-slate-500">{used}/{max}</span>
    </div>
  )
}

function DepositsSection({ surveys }: { surveys: SystemState['surveys'] }) {
  if (!surveys || surveys.length === 0) {
    return (
      <div className="text-xs text-slate-600 italic">
        Keine Survey-Daten — führe zuerst einen Survey durch.
      </div>
    )
  }

  const allDeposits: { planetId: string; good: string; snap: NonNullable<typeof surveys[0]>['snapshot'][string]; stale: boolean }[] = []
  for (const survey of surveys) {
    for (const [good, snap] of Object.entries(survey.snapshot)) {
      allDeposits.push({ planetId: survey.planet_id, good, snap, stale: survey.stale })
    }
  }

  return (
    <div className="grid grid-cols-1 gap-2">
      {allDeposits.map(({ planetId, good, snap, stale }) => (
        <div
          key={`${planetId}-${good}`}
          className="flex items-center gap-3 px-3 py-2 bg-slate-900 border border-slate-700 rounded text-xs"
        >
          <span className="text-slate-300 font-mono w-28 truncate">{good}</span>
          {snap.remaining_exact !== undefined ? (
            <span className="text-white">{snap.remaining_exact.toLocaleString('de-DE')} E</span>
          ) : snap.remaining_approx !== undefined ? (
            <span className="text-slate-400">≈ {snap.remaining_approx} E</span>
          ) : (
            <span className="text-slate-600">vorhanden</span>
          )}
          {snap.max_rate !== undefined && (
            <span className="text-slate-500">max {snap.max_rate}/Tick</span>
          )}
          {snap.slots !== undefined && (
            <span className="text-slate-500">{snap.slots} Slots</span>
          )}
          {stale && (
            <span className="ml-auto px-1.5 py-0.5 bg-yellow-900/50 border border-yellow-700 text-yellow-400 rounded text-[10px]">
              veraltet
            </span>
          )}
        </div>
      ))}
    </div>
  )
}

function FacilityCard({ facility, starId, onRefresh }: {
  facility: Facility
  starId: string
  onRefresh: () => void
}) {
  const [recipeInput, setRecipeInput] = useState('')
  const [assigning, setAssigning] = useState(false)

  const handleAssign = async () => {
    if (!recipeInput.trim()) return
    setAssigning(true)
    try {
      await assignRecipe(starId, facility.id, recipeInput.trim())
      setRecipeInput('')
      onRefresh()
    } catch (e) {
      console.error(e)
    } finally {
      setAssigning(false)
    }
  }

  const eta = facility.config.ticks_remaining > 0
    ? `${facility.config.ticks_remaining} Tick${facility.config.ticks_remaining !== 1 ? 's' : ''}`
    : '—'

  return (
    <div className="px-3 py-2 bg-slate-900 border border-slate-700 rounded text-xs space-y-1.5">
      <div className="flex items-center gap-2">
        <span className="font-semibold text-slate-200">
          {FACILITY_LABELS[facility.facility_type] ?? facility.facility_type} Lv{facility.config.level}
        </span>
        <span className={`${STATUS_COLORS[facility.status] ?? 'text-slate-400'}`}>
          {facility.status}
        </span>
        {facility.planet_id && (
          <span className="text-slate-600 ml-auto font-mono text-[10px]">
            {facility.planet_id.slice(0, 8)}…
          </span>
        )}
      </div>

      <div className="flex gap-4 text-slate-500">
        {facility.config.recipe_id && (
          <span>Rezept: <span className="text-slate-300">{facility.config.recipe_id}</span></span>
        )}
        {facility.status === 'running' && (
          <>
            <span>ETA: <span className="text-slate-300">{eta}</span></span>
            <span>η-Acc: <span className="text-slate-300">{facility.config.efficiency_acc.toFixed(3)}</span></span>
          </>
        )}
      </div>

      {(facility.status === 'idle' || facility.status === 'paused_input') && (
        <div className="flex gap-1 mt-1">
          <input
            value={recipeInput}
            onChange={e => setRecipeInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleAssign()}
            placeholder="recipe_id…"
            className="flex-1 bg-slate-800 border border-slate-600 rounded px-2 py-0.5 text-slate-200 text-xs focus:outline-none focus:border-slate-400"
          />
          <button
            onClick={handleAssign}
            disabled={assigning}
            className="px-2 py-0.5 bg-slate-700 hover:bg-slate-600 text-white rounded disabled:opacity-40 transition-colors"
          >
            Start
          </button>
        </div>
      )}
    </div>
  )
}

function StorageTable({ storage }: { storage: Record<string, number> }) {
  const entries = Object.entries(storage).filter(([, qty]) => qty > 0)
  if (entries.length === 0) {
    return <div className="text-xs text-slate-600 italic">Lager leer.</div>
  }
  return (
    <table className="w-full text-xs">
      <thead>
        <tr className="text-slate-500 border-b border-slate-800">
          <th className="text-left py-1 font-normal">Gut</th>
          <th className="text-right py-1 font-normal">Menge</th>
        </tr>
      </thead>
      <tbody>
        {entries.sort((a, b) => b[1] - a[1]).map(([good, qty]) => (
          <tr key={good} className="border-b border-slate-800/50">
            <td className="py-1 text-slate-300 font-mono">{good}</td>
            <td className="py-1 text-right text-white">{qty.toLocaleString('de-DE', { maximumFractionDigits: 2 })}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function BuildPanel({ starId, onBuilt }: { starId: string; onBuilt: () => void }) {
  const [facilityType, setFacilityType] = useState('mine')
  const [planetId, setPlanetId] = useState('')
  const [depositId, setDepositId] = useState('')
  const [building, setBuilding] = useState(false)
  const [error, setError] = useState('')

  const handleBuild = async () => {
    setError('')
    setBuilding(true)
    try {
      await buildFacility(
        starId,
        facilityType,
        planetId.trim() || null,
        1,
        depositId.trim() || undefined,
      )
      onBuilt()
    } catch (e) {
      setError(String(e))
    } finally {
      setBuilding(false)
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex gap-2 flex-wrap">
        <select
          value={facilityType}
          onChange={e => setFacilityType(e.target.value)}
          className="bg-slate-800 border border-slate-600 rounded px-2 py-1 text-xs text-slate-200 focus:outline-none focus:border-slate-400"
        >
          {Object.entries(FACILITY_LABELS).map(([k, v]) => (
            <option key={k} value={k}>{v}</option>
          ))}
        </select>

        <input
          value={planetId}
          onChange={e => setPlanetId(e.target.value)}
          placeholder="Planet-UUID (leer = orbital)"
          className="flex-1 min-w-0 bg-slate-800 border border-slate-600 rounded px-2 py-1 text-xs text-slate-200 focus:outline-none focus:border-slate-400"
        />

        {facilityType === 'mine' && (
          <input
            value={depositId}
            onChange={e => setDepositId(e.target.value)}
            placeholder="deposit_id (z.B. iron_ore)"
            className="flex-1 min-w-0 bg-slate-800 border border-slate-600 rounded px-2 py-1 text-xs text-slate-200 focus:outline-none focus:border-slate-400"
          />
        )}

        <button
          onClick={handleBuild}
          disabled={building}
          className="px-3 py-1 bg-cyan-700 hover:bg-cyan-600 text-white text-xs font-bold rounded disabled:opacity-40 transition-colors"
        >
          {building ? '…' : 'Bauen'}
        </button>
      </div>
      {error && <div className="text-xs text-red-400">{error}</div>}
    </div>
  )
}

function EventLog({ rows, liveEvents }: { rows: LogRow[]; liveEvents: string[] }) {
  const flat: { tickN: number; label: string; type: string }[] = []

  for (const row of rows) {
    for (const ev of row.Events ?? []) {
      flat.push({ tickN: row.TickN, label: eventLabel(ev), type: ev.type })
    }
  }

  return (
    <div className="space-y-0.5 max-h-48 overflow-y-auto font-mono text-[11px]">
      {liveEvents.map((msg, i) => (
        <div key={`live-${i}`} className="text-emerald-300">▶ {msg}</div>
      ))}
      {flat.length === 0 && liveEvents.length === 0 && (
        <div className="text-slate-600 italic">Noch keine Ereignisse.</div>
      )}
      {flat.map((e, i) => (
        <div key={i} className="flex gap-2">
          <span className="text-slate-600 w-8 shrink-0">#{e.tickN}</span>
          <span className={EVENT_COLORS[e.type] ?? 'text-slate-400'}>{e.label}</span>
        </div>
      ))}
    </div>
  )
}

// ── Main Page ────────────────────────────────────────────────────────────────

export function EconomyPage({ starId }: Props) {
  const [state, setState]         = useState<SystemState | null>(null)
  const [log, setLog]             = useState<LogRow[]>([])
  const [liveEvents, setLiveEvents] = useState<string[]>([])
  const [sseOk, setSseOk]         = useState(false)
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState('')
  const liveRef = useRef<string[]>([])

  const load = useCallback(async () => {
    try {
      const [s, l] = await Promise.all([
        fetchSystemState(starId),
        fetchLog(starId, 20),
      ])
      setState(s)
      setLog(l)
      setError('')
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }, [starId])

  // Initial load
  useEffect(() => { load() }, [load])

  // SSE tick stream
  useEffect(() => {
    const close = openTickStream(
      starId,
      (ev: TickEvent) => {
        setSseOk(true)
        const msg = ev.message ?? `Tick #${ev.tick}`
        liveRef.current = [msg, ...liveRef.current].slice(0, 10)
        setLiveEvents([...liveRef.current])
        // Refresh state after each tick
        load()
      },
      () => setSseOk(false),
    )
    setSseOk(true)
    return close
  }, [starId, load])

  const handleAdvance = async () => {
    await advanceTick()
    await load()
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-400 gap-3">
        <div className="w-6 h-6 border-2 border-slate-600 border-t-cyan-400 rounded-full animate-spin" />
        <span className="text-sm">Lade Wirtschaftsdaten…</span>
      </div>
    )
  }

  if (error || !state) {
    return (
      <div className="flex items-center justify-center h-full text-red-400 text-sm">
        {error || 'Kein System gewählt.'}
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto p-4 space-y-5 text-slate-200">

      {/* Header row */}
      <div className="flex items-center gap-4 flex-wrap">
        <TickControls tickN={state.last_tick_n} sseOk={sseOk} onAdvance={handleAdvance} />
        <OrbitalSlotsBar used={state.orbital_slots_used} max={state.orbital_slots_max} />
        <span className="text-xs text-slate-600 font-mono ml-auto">{starId.slice(0, 8)}…</span>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">

        {/* Left column */}
        <div className="space-y-5">

          {/* Deposits */}
          <section>
            <h2 className="text-xs font-bold tracking-widest text-slate-400 uppercase mb-2">
              Deposits (Survey-Daten)
            </h2>
            <DepositsSection surveys={state.surveys} />
          </section>

          {/* Facilities */}
          <section>
            <h2 className="text-xs font-bold tracking-widest text-slate-400 uppercase mb-2">
              Anlagen ({state.facilities.length})
            </h2>
            {state.facilities.length === 0 ? (
              <div className="text-xs text-slate-600 italic">Keine Anlagen.</div>
            ) : (
              <div className="space-y-2">
                {state.facilities.map(f => (
                  <FacilityCard key={f.id} facility={f} starId={starId} onRefresh={load} />
                ))}
              </div>
            )}
          </section>

          {/* Build Panel */}
          <section>
            <h2 className="text-xs font-bold tracking-widest text-slate-400 uppercase mb-2">
              Anlage bauen
            </h2>
            <BuildPanel starId={starId} onBuilt={load} />
          </section>

        </div>

        {/* Right column */}
        <div className="space-y-5">

          {/* Storage */}
          <section>
            <h2 className="text-xs font-bold tracking-widest text-slate-400 uppercase mb-2">
              Systemlager
            </h2>
            <StorageTable storage={state.storage} />
          </section>

          {/* Event Log */}
          <section>
            <h2 className="text-xs font-bold tracking-widest text-slate-400 uppercase mb-2">
              Ereignis-Log
            </h2>
            <EventLog rows={log} liveEvents={liveEvents} />
          </section>

        </div>
      </div>
    </div>
  )
}
