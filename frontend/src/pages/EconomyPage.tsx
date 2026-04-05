import { useEffect, useState, useCallback, useRef, useMemo } from 'react'
import {
  fetchSystemState, fetchLog, fetchRecipes, buildFacility, assignRecipe,
  advanceTick, openTickStream, createOrder, cancelOrder,
} from '../api/economy'
import type {
  SystemState, StorageNode, Facility, LogRow, LogEvent, TickEvent, RecipeInfo,
  ProductionOrder,
} from '../api/economy'

interface Props {
  starId: string
  onBack?: () => void
}

// ── Helpers ──────────────────────────────────────────────────────────────────

const FACILITY_LABELS: Record<string, string> = {
  // Rohstoff-Infrastruktur
  mine:                 'Bergwerk',
  elevator:             'Aufzug',
  // Tier-2 Metalle
  steel_mill:           'Stahlwerk',
  titansteel_forge:     'Titanstahl-Schmiede',
  chrom_alloy_plant:    'Chrom-Legierungswerk',
  keramik_plant:        'Keramikwerk',
  // Tier-2 Chemie / Elektronik / Bio
  semiconductor_plant:  'Halbleiterwerk',
  fuel_processor:       'Treibstoffanlage',
  reprocessing_plant:   'Wiederaufbereitung',
  coolant_plant:        'Kühlmittelwerk',
  biosynth_lab:         'Biosynth-Labor',
  // Tier-3/4
  precision_factory:    'Präzisionsfabrik',
  assembler:            'Assembler',
  shipyard:             'Werft',
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
  build_complete:    'text-sky-400',
  paused_input:      'text-yellow-400',
  paused_output:     'text-orange-400',
  deposit_depleted:  'text-red-500',
  deposit_warning:   'text-yellow-500',
  deposit_critical:  'text-red-400',
}

function eventLabel(ev: LogEvent): string {
  switch (ev.type) {
    case 'build_complete':   return `Bau abgeschlossen: ${ev.facility_id.slice(0, 8)}…`
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
    for (const [good, snap] of Object.entries(survey.snapshot ?? {})) {
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

function RecipeDetail({ recipe }: { recipe: RecipeInfo }) {
  return (
    <div className="mt-1.5 px-2 py-1.5 bg-slate-800/70 border border-slate-700/60 rounded space-y-1">
      <div className="flex gap-4">
        <span className="text-slate-500">T{recipe.tier}</span>
        <span className="text-slate-500">{recipe.ticks} Tick{recipe.ticks !== 1 ? 's' : ''}</span>
      </div>
      <div className="flex gap-4 flex-wrap">
        {Object.entries(recipe.inputs).map(([good, qty]) => (
          <span key={good} className="text-red-400">
            -{qty} <span className="text-red-300/80">{good}</span>
          </span>
        ))}
        <span className="text-slate-600">→</span>
        {Object.entries(recipe.outputs).map(([good, qty]) => (
          <span key={good} className="text-emerald-400">
            +{qty} <span className="text-emerald-300/80">{good}</span>
          </span>
        ))}
      </div>
    </div>
  )
}

function FacilityCard({ facility, starId, recipes, onRefresh }: {
  facility: Facility
  starId: string
  recipes: RecipeInfo[]
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

  // Recipe to show detail for: selected in dropdown > currently assigned
  const previewRecipeId = recipeInput || facility.config.recipe_id || ''
  const detailRecipe = previewRecipeId
    ? recipes.find(r => r.id === previewRecipeId)
    : undefined

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

      {detailRecipe && <RecipeDetail recipe={detailRecipe} />}

      {(facility.status === 'idle' || facility.status === 'paused_input') && recipes.length > 0 && (
        <div className="flex gap-1 mt-1">
          <select
            value={recipeInput}
            onChange={e => setRecipeInput(e.target.value)}
            className="flex-1 bg-slate-800 border border-slate-600 rounded px-2 py-0.5 text-slate-200 text-xs focus:outline-none focus:border-slate-400"
          >
            <option value="">— Rezept wählen —</option>
            {recipes.map(r => (
              <option key={r.id} value={r.id}>
                {r.name} (T{r.tier}, {r.ticks} Tick{r.ticks !== 1 ? 's' : ''})
              </option>
            ))}
          </select>
          <button
            onClick={handleAssign}
            disabled={assigning || !recipeInput}
            className="px-2 py-0.5 bg-slate-700 hover:bg-slate-600 text-white rounded disabled:opacity-40 transition-colors"
          >
            Start
          </button>
        </div>
      )}
    </div>
  )
}

function NodeHeader({ node }: { node: StorageNode }) {
  const label = node.level === 'orbital'
    ? '◈ Orbital'
    : node.planet_id
      ? `▣ Planet ${node.planet_id.slice(0, 8)}…`
      : node.level
  const capacityInfo = node.capacity != null
    ? ` · max ${node.capacity.toLocaleString('de-DE')} E`
    : ' · unbegrenzt'
  return (
    <div className="flex items-center gap-2 mb-1.5">
      <span className={`text-[10px] font-bold tracking-widest uppercase ${
        node.level === 'orbital' ? 'text-cyan-600' : 'text-slate-500'
      }`}>
        {label}
      </span>
      <span className="text-[10px] text-slate-600">{capacityInfo}</span>
    </div>
  )
}

const TIER_LABELS: Record<number, string> = {
  0: 'Rohstoffe',
  1: 'T1 – Basisverarbeitung',
  2: 'T2 – Halbzeug',
  3: 'T3 – Komponenten',
  4: 'T4 – Endprodukte',
}

function StorageTable({
  storage,
  goodTier,
}: {
  storage: Record<string, number>
  goodTier: Record<string, number>
}) {
  const entries = Object.entries(storage).filter(([, qty]) => qty > 0)
  if (entries.length === 0) {
    return <div className="text-xs text-slate-600 italic">Lager leer.</div>
  }

  // Group by tier (goods not in goodTier map → tier 0 = raw)
  const groups = new Map<number, [string, number][]>()
  for (const entry of entries) {
    const tier = goodTier[entry[0]] ?? 0
    if (!groups.has(tier)) groups.set(tier, [])
    groups.get(tier)!.push(entry)
  }
  // Sort groups by tier, entries within group by qty desc
  const sortedGroups = [...groups.entries()].sort((a, b) => a[0] - b[0])
  for (const [, list] of sortedGroups) list.sort((a, b) => b[1] - a[1])

  return (
    <div className="space-y-3">
      {sortedGroups.map(([tier, list]) => (
        <div key={tier}>
          <div className="text-[10px] font-bold tracking-widest uppercase text-slate-500 mb-1">
            {TIER_LABELS[tier] ?? `T${tier}`}
          </div>
          <table className="w-full text-xs">
            <tbody>
              {list.map(([good, qty]) => (
                <tr key={good} className="border-b border-slate-800/50">
                  <td className="py-1 text-slate-300 font-mono">{good}</td>
                  <td className="py-1 text-right text-white">{qty.toLocaleString('de-DE', { maximumFractionDigits: 2 })}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ))}
    </div>
  )
}

const _MODE_LABELS: Record<string, string> = {
  continuous_full:   'Dauerlauf (voll)',
  continuous_demand: 'Bedarfsgesteuert',
  batch:             'Batch',
}

function OrderRow({
  order,
  starId,
  assignedCount,
  onCancel,
}: {
  order: ProductionOrder
  starId: string
  assignedCount: number
  onCancel: () => void
}) {
  const [cancelling, setCancelling] = useState(false)

  const handleCancel = async () => {
    setCancelling(true)
    try {
      await cancelOrder(starId, order.id)
      onCancel()
    } catch (e) {
      console.error(e)
    } finally {
      setCancelling(false)
    }
  }

  return (
    <div className={`px-3 py-2 rounded border text-xs space-y-1 ${
      order.active ? 'bg-slate-900 border-slate-700' : 'bg-slate-900/40 border-slate-800 opacity-60'
    }`}>
      <div className="flex items-center gap-2">
        <span className="font-semibold text-slate-200">
          {FACILITY_LABELS[order.facility_type] ?? order.facility_type}
        </span>
        <span className="text-slate-500">·</span>
        <span className="text-slate-400">{order.recipe_id}</span>
        <span className={`ml-auto text-[10px] px-1.5 py-0.5 rounded font-bold ${
          order.active ? 'bg-emerald-900/50 text-emerald-400 border border-emerald-800' : 'bg-slate-800 text-slate-500'
        }`}>
          {order.active ? 'aktiv' : 'inaktiv'}
        </span>
      </div>
      <div className="flex items-center gap-3 text-slate-500">
        <span className={`text-[10px] px-1.5 py-0.5 rounded border font-bold ${
          order.mode === 'batch'
            ? 'bg-amber-900/40 border-amber-700 text-amber-400'
            : 'bg-slate-800 border-slate-700 text-slate-500'
        }`}>
          {order.mode === 'batch' ? '⚡ Batch' : order.mode === 'continuous_demand' ? '⟳ Bedarf' : '⟳ Dauerlauf'}
        </span>
        {order.mode === 'batch' && order.batch_remaining != null && (
          <span>noch {order.batch_remaining}×</span>
        )}
        {order.mode === 'continuous_demand' && order.target_stock != null && (
          <span>Ziel: {order.target_stock.toLocaleString('de-DE')} {order.good_id}</span>
        )}
        <span className="font-mono text-slate-600">P{order.priority}</span>
        {assignedCount > 0 && (
          <span className="text-cyan-500">{assignedCount} Anlage{assignedCount !== 1 ? 'n' : ''} aktiv</span>
        )}
        {order.active && (
          <button
            onClick={handleCancel}
            disabled={cancelling}
            className="ml-auto text-red-500 hover:text-red-400 disabled:opacity-40 transition-colors"
            title="Auftrag abbrechen"
          >
            ✕
          </button>
        )}
      </div>
    </div>
  )
}

function NewOrderForm({
  starId,
  recipes,
  onCreated,
}: {
  starId: string
  recipes: RecipeInfo[]
  onCreated: () => void
}) {
  const [facilityType, setFacilityType] = useState('')
  const [recipeId, setRecipeId] = useState('')
  const [mode, setMode] = useState<'continuous_full' | 'continuous_demand' | 'batch'>('continuous_full')
  const [batchCount, setBatchCount] = useState(1)
  const [goodId, setGoodId] = useState('')
  const [targetStock, setTargetStock] = useState(100)
  const [priority, setPriority] = useState(0)
  const [submitting, setSubmitting] = useState(false)

  // Derive available facility types from recipes
  const facilityTypes = useMemo(() => {
    const types = new Set(recipes.map(r => r.facility_type))
    return [...types].sort()
  }, [recipes])

  const filteredRecipes = useMemo(
    () => facilityType ? recipes.filter(r => r.facility_type === facilityType) : [],
    [recipes, facilityType],
  )

  // Reset recipe when facility type changes
  const handleFacilityTypeChange = (ft: string) => {
    setFacilityType(ft)
    setRecipeId('')
  }

  const canSubmit = facilityType && recipeId

  const handleCreate = async () => {
    if (!canSubmit) return
    setSubmitting(true)
    try {
      await createOrder(starId, {
        facility_type: facilityType,
        recipe_id: recipeId,
        mode,
        batch_remaining: mode === 'batch' ? batchCount : undefined,
        good_id: mode === 'continuous_demand' ? goodId || undefined : undefined,
        target_stock: mode === 'continuous_demand' ? targetStock : undefined,
        priority,
      })
      onCreated()
    } catch (e) {
      console.error(e)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-2 px-3 py-2 bg-slate-900/50 border border-slate-700 rounded">
      <div className="text-[10px] font-bold tracking-widest uppercase text-slate-500">Neuer Auftrag</div>
      <div className="flex gap-2 flex-wrap">
        <select
          value={facilityType}
          onChange={e => handleFacilityTypeChange(e.target.value)}
          className="bg-slate-800 border border-slate-600 rounded px-2 py-0.5 text-xs text-slate-200 focus:outline-none focus:border-slate-400"
        >
          <option value="">— Anlage —</option>
          {facilityTypes.map(ft => (
            <option key={ft} value={ft}>{FACILITY_LABELS[ft] ?? ft}</option>
          ))}
        </select>

        <select
          value={recipeId}
          onChange={e => setRecipeId(e.target.value)}
          disabled={!facilityType}
          className="flex-1 bg-slate-800 border border-slate-600 rounded px-2 py-0.5 text-xs text-slate-200 focus:outline-none focus:border-slate-400 disabled:opacity-40"
        >
          <option value="">— Rezept —</option>
          {filteredRecipes.map(r => (
            <option key={r.id} value={r.id}>{r.name} (T{r.tier})</option>
          ))}
        </select>

        <select
          value={mode}
          onChange={e => setMode(e.target.value as typeof mode)}
          className="bg-slate-800 border border-slate-600 rounded px-2 py-0.5 text-xs text-slate-200 focus:outline-none focus:border-slate-400"
        >
          <option value="continuous_full">Dauerlauf</option>
          <option value="continuous_demand">Bedarfsgest.</option>
          <option value="batch">Batch</option>
        </select>
      </div>

      {mode === 'batch' && (
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-500">Anzahl Batches:</span>
          <input
            type="number"
            min={1}
            value={batchCount}
            onChange={e => setBatchCount(Math.max(1, Number(e.target.value)))}
            className="w-20 bg-slate-800 border border-slate-600 rounded px-2 py-0.5 text-xs text-slate-200 focus:outline-none"
          />
        </div>
      )}

      {mode === 'continuous_demand' && (
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs text-slate-500">Gut:</span>
          <input
            value={goodId}
            onChange={e => setGoodId(e.target.value)}
            placeholder="z.B. steel"
            className="w-28 bg-slate-800 border border-slate-600 rounded px-2 py-0.5 text-xs text-slate-200 focus:outline-none"
          />
          <span className="text-xs text-slate-500">Ziel-Lager:</span>
          <input
            type="number"
            min={0}
            value={targetStock}
            onChange={e => setTargetStock(Number(e.target.value))}
            className="w-24 bg-slate-800 border border-slate-600 rounded px-2 py-0.5 text-xs text-slate-200 focus:outline-none"
          />
        </div>
      )}

      <div className="flex items-center gap-2">
        <span className="text-xs text-slate-500">Priorität:</span>
        <input
          type="number"
          value={priority}
          onChange={e => setPriority(Number(e.target.value))}
          className="w-16 bg-slate-800 border border-slate-600 rounded px-2 py-0.5 text-xs text-slate-200 focus:outline-none font-mono"
          title="Höhere Zahl = höhere Priorität. Batch-Aufträge haben immer Vorrang vor kontinuierlichen."
        />
        <span className="text-[10px] text-slate-600">(Batch hat immer Vorrang)</span>
        <button
          onClick={handleCreate}
          disabled={!canSubmit || submitting}
          className="ml-auto text-xs px-3 py-1 bg-cyan-800 hover:bg-cyan-700 text-white rounded disabled:opacity-40 transition-colors"
        >
          {submitting ? '…' : '+ Auftrag anlegen'}
        </button>
      </div>
    </div>
  )
}

function ProductionOrdersSection({
  orders,
  starId,
  facilities,
  recipes,
  onRefresh,
}: {
  orders: ProductionOrder[]
  starId: string
  facilities: Facility[]
  recipes: RecipeInfo[]
  onRefresh: () => void
}) {
  // Count how many facilities are currently assigned to each order
  const assignedPerOrder = useMemo(() => {
    const map: Record<string, number> = {}
    for (const f of facilities) {
      if (f.current_order_id) {
        map[f.current_order_id] = (map[f.current_order_id] ?? 0) + 1
      }
    }
    return map
  }, [facilities])

  const activeOrders = orders.filter(o => o.active)
  const inactiveOrders = orders.filter(o => !o.active)

  return (
    <div className="space-y-2">
      {activeOrders.length === 0 && (
        <div className="text-xs text-slate-600 italic">Keine aktiven Aufträge.</div>
      )}
      {activeOrders.map(order => (
        <OrderRow
          key={order.id}
          order={order}
          starId={starId}
          assignedCount={assignedPerOrder[order.id] ?? 0}
          onCancel={onRefresh}
        />
      ))}
      {inactiveOrders.length > 0 && (
        <details className="text-xs">
          <summary className="cursor-pointer text-slate-600 hover:text-slate-400">
            {inactiveOrders.length} abgeschlossene / inaktive Aufträge
          </summary>
          <div className="mt-1 space-y-1">
            {inactiveOrders.map(order => (
              <OrderRow
                key={order.id}
                order={order}
                starId={starId}
                assignedCount={0}
                onCancel={onRefresh}
              />
            ))}
          </div>
        </details>
      )}
      <NewOrderForm starId={starId} recipes={recipes} onCreated={onRefresh} />
    </div>
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

export function EconomyPage({ starId, onBack }: Props) {
  const [state, setState]         = useState<SystemState | null>(null)
  const [log, setLog]             = useState<LogRow[]>([])
  const [recipes, setRecipes]     = useState<RecipeInfo[]>([])
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

  // Load recipes once (static data)
  useEffect(() => {
    fetchRecipes().then(setRecipes).catch(() => {})
  }, [])

  // Map good → highest recipe output tier (0 = raw/unmapped)
  const goodTierMap = useMemo<Record<string, number>>(() => {
    const map: Record<string, number> = {}
    for (const recipe of recipes) {
      for (const goodId of Object.keys(recipe.outputs)) {
        map[goodId] = Math.max(map[goodId] ?? 0, recipe.tier)
      }
    }
    return map
  }, [recipes])

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
        {onBack && (
          <button
            onClick={onBack}
            className="text-xs text-slate-500 hover:text-slate-300 transition-colors flex items-center gap-1"
          >
            ← Systeme
          </button>
        )}
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

          {/* Facilities — grouped by planet / orbital */}
          <section>
            <h2 className="text-xs font-bold tracking-widest text-slate-400 uppercase mb-2">
              Anlagen ({state.facilities.length})
            </h2>
            {state.facilities.length === 0 ? (
              <div className="text-xs text-slate-600 italic">Keine Anlagen.</div>
            ) : (() => {
              // Group by planet_id (null = orbital)
              const groups = new Map<string, Facility[]>()
              for (const f of state.facilities) {
                const key = f.planet_id ?? '__orbital__'
                if (!groups.has(key)) groups.set(key, [])
                groups.get(key)!.push(f)
              }
              return (
                <div className="space-y-4">
                  {[...groups.entries()].map(([key, facilities]) => (
                    <div key={key}>
                      <div className="text-[10px] font-bold tracking-widest uppercase mb-1.5 flex items-center gap-2">
                        {key === '__orbital__' ? (
                          <span className="text-cyan-600">◈ Orbital</span>
                        ) : (
                          <span className="text-slate-500">▣ Planet <span className="font-mono">{key.slice(0, 8)}…</span></span>
                        )}
                      </div>
                      <div className="space-y-2 pl-3 border-l border-slate-800">
                        {facilities.map(f => (
                          <FacilityCard
                            key={f.id}
                            facility={f}
                            starId={starId}
                            recipes={recipes.filter(r => r.facility_type === f.facility_type)}
                            onRefresh={load}
                          />
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              )
            })()}
          </section>

          {/* Build Queue */}
          {state.facilities.some(f => f.status === 'building') && (
            <section>
              <h2 className="text-xs font-bold tracking-widest text-sky-600 uppercase mb-2">
                Bauliste
              </h2>
              <div className="space-y-1.5">
                {state.facilities.filter(f => f.status === 'building').map(f => (
                  <div key={f.id} className="flex items-center gap-3 px-3 py-2 bg-slate-900 border border-sky-900/50 rounded text-xs">
                    <span className="text-slate-300">{FACILITY_LABELS[f.facility_type] ?? f.facility_type} Lv{f.config.level}</span>
                    <span className="ml-auto text-sky-400 font-mono">
                      ETA {f.config.ticks_remaining} Tick{f.config.ticks_remaining !== 1 ? 's' : ''}
                    </span>
                  </div>
                ))}
              </div>
            </section>
          )}

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

          {/* Production orders */}
          <section>
            <h2 className="text-xs font-bold tracking-widest text-slate-400 uppercase mb-2">
              Fertigungsaufträge ({(state.orders ?? []).filter(o => o.active).length} aktiv)
            </h2>
            <ProductionOrdersSection
              orders={state.orders ?? []}
              starId={starId}
              facilities={state.facilities}
              recipes={recipes}
              onRefresh={load}
            />
          </section>

          {/* Storage nodes */}
          <section>
            <h2 className="text-xs font-bold tracking-widest text-slate-400 uppercase mb-2">
              Systemlager
            </h2>
            {state.storage_nodes.length === 0 ? (
              <div className="text-xs text-slate-600 italic">Keine Lagerknoten.</div>
            ) : (
              <div className="space-y-4">
                {state.storage_nodes.map(node => (
                  <div key={node.id}>
                    <NodeHeader node={node} />
                    <StorageTable storage={node.storage} goodTier={goodTierMap} />
                  </div>
                ))}
              </div>
            )}
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
