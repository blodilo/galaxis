import { useState, useEffect, useCallback, useRef } from 'react'
import { useNatsTick } from '../hooks/useNatsTick'
import type { ItemStock, Facility, Order, Route, Recipe, MyNodeEntry, DepositEntry } from '../types/economy2'
import {
  getStock,
  listFacilities,
  destroyFacility,
  createOrder,
  listOrders,
  cancelOrder,
  createRoute,
  listRoutes,
  listRecipes,
  bootstrap,
  listMyNodes,
  getDeposits,
} from '../api/economy2'

// ── Labels ────────────────────────────────────────────────────────────────────

const ITEM_LABELS: Record<string, string> = {
  steel: 'Stahl', titansteel: 'Titanstahl',
  semiconductor_wafer: 'Halbleiter-Wafer', fusion_fuel: 'Fusionskraftstoff',
  base_component: 'Basisbauteil', nav_computer: 'Navigationscomputer',
  iron_ore: 'Eisenerz', silicates: 'Silikate', titan: 'Titan',
  rare_earths: 'Seltene Erden', he3: 'Helium-3', hydrogen: 'Wasserstoff',
}

const FACILITY_TYPE_LABELS: Record<string, string> = {
  mine: 'Mine', smelter: 'Schmelze', refinery: 'Raffinerie',
  precision_fab: 'Präzisionsfertigung', construction: 'Bau',
}

const PRODUCT_LABELS: Record<string, string> = {
  ...ITEM_LABELS,
  facility_mine_iron_ore: 'Mine: Eisenerz',
  facility_mine_silicates: 'Mine: Silikate',
  facility_mine_titan: 'Mine: Titan',
  facility_mine_rare_earths: 'Mine: Seltene Erden',
  facility_mine_he3: 'Mine: Helium-3',
  facility_mine_hydrogen: 'Mine: Wasserstoff',
  facility_smelter: 'Schmelze',
  facility_refinery: 'Raffinerie',
  facility_precision_fab: 'Präzisionsfertigung',
}

function recipeLabel(r: Recipe): string {
  const base = PRODUCT_LABELS[r.product_id] ?? r.product_id
  if (r.factory_type === 'construction') return `${base} bauen`
  const ft = FACILITY_TYPE_LABELS[r.factory_type] ?? r.factory_type
  return `${base} (${ft})`
}

function itemLabel(id: string): string {
  return ITEM_LABELS[id] ?? id
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function short(uuid: string): string {
  return uuid.slice(0, 8)
}

function StatusBadge({ status, colors }: { status: string; colors: Record<string, string> }) {
  const cls = colors[status] ?? 'bg-slate-800 text-slate-400'
  return (
    <span className={`text-xs px-1.5 py-0.5 rounded font-bold ${cls}`}>
      {status.toUpperCase()}
    </span>
  )
}

const FACILITY_STATUS_COLORS: Record<string, string> = {
  idle:            'bg-slate-800 text-slate-400',
  running:         'bg-emerald-900/60 text-emerald-400',
  building:        'bg-blue-900/60 text-blue-400',
  paused_input:    'bg-orange-900/60 text-orange-400',
  paused_depleted: 'bg-orange-900/60 text-orange-400',
  destroyed:       'bg-red-900/60 text-red-400',
}

const ORDER_STATUS_COLORS: Record<string, string> = {
  pending:         'bg-slate-800 text-slate-400',
  waiting:         'bg-slate-800 text-slate-400',
  ready:           'bg-blue-900/60 text-blue-400',
  running:         'bg-emerald-900/60 text-emerald-400',
  completed:       'bg-slate-700 text-slate-500',
  cancelled:       'bg-slate-700 text-slate-500',
  paused_depleted: 'bg-orange-900/60 text-orange-400',
}

const TERMINAL_ORDER_STATUSES = new Set(['completed', 'cancelled'])

// ── Sub-components ────────────────────────────────────────────────────────────

function Spinner() {
  return (
    <div className="w-4 h-4 border-2 border-slate-700 border-t-emerald-400 rounded-full animate-spin" />
  )
}

function StatusLamp({ status }: { status: string }) {
  const cls: Record<string, string> = {
    running:         'bg-emerald-500',
    paused_depleted: 'bg-orange-500',
    paused_input:    'bg-yellow-500',
    idle:            'bg-slate-600',
  }
  return (
    <span
      className={`inline-block w-2 h-2 rounded-full flex-shrink-0 ${cls[status] ?? 'bg-slate-700'}`}
      title={status}
    />
  )
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="text-xs tracking-widest text-slate-500 uppercase mb-2">{children}</h3>
  )
}

function Card({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={`bg-slate-900/50 border border-slate-800 rounded px-3 py-2 mb-1.5 ${className}`}>
      {children}
    </div>
  )
}

function PrimaryButton({ onClick, children, disabled }: {
  onClick: () => void
  children: React.ReactNode
  disabled?: boolean
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="text-xs px-2 py-0.5 rounded border border-emerald-700 text-emerald-400
                 hover:bg-emerald-900/30 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
    >
      {children}
    </button>
  )
}

function DangerButton({ onClick, children, disabled }: {
  onClick: () => void
  children: React.ReactNode
  disabled?: boolean
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="text-xs px-2 py-0.5 rounded border border-red-800 text-red-400
                 hover:bg-red-900/30 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
    >
      {children}
    </button>
  )
}

function CancelButton({ onClick, children }: { onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className="text-xs px-2 py-0.5 rounded border border-slate-700 text-slate-500
                 hover:border-slate-500 transition-colors"
    >
      {children}
    </button>
  )
}

function TextInput({
  value,
  onChange,
  placeholder,
  className = '',
}: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  className?: string
}) {
  return (
    <input
      type="text"
      value={value}
      onChange={e => onChange(e.target.value)}
      placeholder={placeholder}
      className={`bg-black border border-slate-700 rounded px-2 py-1 text-xs text-slate-300 w-full ${className}`}
    />
  )
}

function NumberInput({
  value,
  onChange,
  placeholder,
  min,
  max,
  step,
}: {
  value: number | string
  onChange: (v: string) => void
  placeholder?: string
  min?: number
  max?: number
  step?: number
}) {
  return (
    <input
      type="number"
      value={value}
      onChange={e => onChange(e.target.value)}
      placeholder={placeholder}
      min={min}
      max={max}
      step={step}
      className="bg-black border border-slate-700 rounded px-2 py-1 text-xs text-slate-300 w-full"
    />
  )
}

function FormWrapper({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-slate-900 border border-slate-700 rounded p-3 mt-2 flex flex-col gap-2">
      {children}
    </div>
  )
}

function Select({
  value, onChange, children,
}: {
  value: string
  onChange: (v: string) => void
  children: React.ReactNode
}) {
  return (
    <select
      value={value}
      onChange={e => onChange(e.target.value)}
      className="bg-black border border-slate-700 rounded px-2 py-1 text-xs text-slate-300 w-full
                 focus:outline-none focus:border-emerald-600"
    >
      {children}
    </select>
  )
}

// ── Left column: LAGER ────────────────────────────────────────────────────────

function LagerPanel({ stock, loading }: { stock: ItemStock[]; loading: boolean }) {
  return (
    <div className="flex flex-col h-full overflow-hidden">
      <SectionTitle>Lager</SectionTitle>
      {loading && (
        <div className="flex justify-center py-4"><Spinner /></div>
      )}
      {!loading && stock.length === 0 && (
        <p className="text-xs text-slate-600 italic">Kein Lager verfügbar</p>
      )}
      {!loading && stock.length > 0 && (
        <>
          <div className="grid grid-cols-4 gap-1 text-xs text-slate-600 mb-1 px-1">
            <span>GUT</span>
            <span className="text-right">TOTAL</span>
            <span className="text-right">GEB.</span>
            <span className="text-right">FREI</span>
          </div>
          <div className="overflow-y-auto flex-1">
            {stock.map(item => {
              const ratio = item.total > 0 ? item.available / item.total : 1
              const valueColor =
                item.available === 0
                  ? 'text-red-400'
                  : ratio < 0.1
                  ? 'text-orange-400'
                  : 'text-slate-300'
              return (
                <div
                  key={item.item_id}
                  className="grid grid-cols-4 gap-1 text-xs px-1 py-0.5 hover:bg-slate-800/30 rounded"
                >
                  <span className="text-slate-400 truncate" title={item.item_id}>
                    {item.item_id}
                  </span>
                  <span className={`text-right ${valueColor}`}>{item.total.toFixed(1)}</span>
                  <span className={`text-right ${valueColor}`}>{item.allocated.toFixed(1)}</span>
                  <span className={`text-right ${valueColor}`}>{item.available.toFixed(1)}</span>
                </div>
              )
            })}
          </div>
        </>
      )}
    </div>
  )
}

// ── Minen-Übersicht ───────────────────────────────────────────────────────────

interface MinenPanelProps {
  facilities: Facility[]
  orders: Order[]
  deposits: Record<string, DepositEntry>
  mineRateLv1: number
  nodeId: string
  starId: string
  loading: boolean
  onRefresh: () => void
}

function MinenPanel({
  facilities, orders, deposits, mineRateLv1,
  nodeId, starId, loading, onRefresh,
}: MinenPanelProps) {
  const [activating, setActivating] = useState<Record<string, boolean>>({})
  const [building, setBuilding]     = useState<Record<string, boolean>>({})
  const [errors, setErrors]         = useState<Record<string, string>>({})
  const [buildRecipes, setBuildRecipes] = useState<Recipe[]>([])

  useEffect(() => {
    listRecipes()
      .then(rs => setBuildRecipes(rs.filter(r => r.factory_type === 'construction')))
      .catch(() => {})
  }, [])

  const mines = facilities.filter(f => f.factory_type === 'mine' && f.status !== 'destroyed')
  const mineOrders = orders.filter(o =>
    o.factory_type === 'mine' && !TERMINAL_ORDER_STATUSES.has(o.status)
  )
  const mineBuildOrders = orders.filter(o =>
    o.factory_type === 'construction' &&
    o.product_id.startsWith('facility_mine_') &&
    !TERMINAL_ORDER_STATUSES.has(o.status)
  )

  const goodIds = Array.from(new Set([
    ...Object.keys(deposits).filter(g => (deposits[g]?.remaining ?? 0) > 0),
    ...mines.map(f => f.config.deposit_good_id ?? '').filter(Boolean),
  ])).sort()

  const minesByGood = new Map<string, Facility[]>()
  for (const f of mines) {
    const g = f.config.deposit_good_id ?? ''
    if (!g) continue
    if (!minesByGood.has(g)) minesByGood.set(g, [])
    minesByGood.get(g)!.push(f)
  }

  const buildsByGood = new Map<string, Order[]>()
  for (const o of mineBuildOrders) {
    const g = o.product_id.replace('facility_mine_', '')
    if (!buildsByGood.has(g)) buildsByGood.set(g, [])
    buildsByGood.get(g)!.push(o)
  }

  const ordersByGood = new Map<string, Order[]>()
  for (const o of mineOrders) {
    if (!ordersByGood.has(o.product_id)) ordersByGood.set(o.product_id, [])
    ordersByGood.get(o.product_id)!.push(o)
  }

  async function handleActivate(goodId: string) {
    setActivating(a => ({ ...a, [goodId]: true }))
    setErrors(e => ({ ...e, [goodId]: '' }))
    try {
      await createOrder({
        node_id: nodeId, star_id: starId,
        factory_type: 'mine', product_id: goodId,
        order_type: 'continuous', target_qty: 999_999, priority: 8,
      })
      onRefresh()
    } catch (err) {
      setErrors(e => ({ ...e, [goodId]: err instanceof Error ? err.message : 'Fehler' }))
    } finally {
      setActivating(a => ({ ...a, [goodId]: false }))
    }
  }

  async function handleBuild(goodId: string) {
    const recipe = buildRecipes.find(r => r.product_id === `facility_mine_${goodId}`)
    if (!recipe) return
    setBuilding(b => ({ ...b, [goodId]: true }))
    setErrors(e => ({ ...e, [goodId]: '' }))
    try {
      await createOrder({
        node_id: nodeId, star_id: starId,
        factory_type: 'construction', product_id: recipe.product_id,
        order_type: 'build', target_qty: 1,
      })
      onRefresh()
    } catch (err) {
      setErrors(e => ({ ...e, [goodId]: err instanceof Error ? err.message : 'Fehler' }))
    } finally {
      setBuilding(b => ({ ...b, [goodId]: false }))
    }
  }

  return (
    <div>
      <SectionTitle>Minen & Vorkommen</SectionTitle>
      {loading && <div className="flex justify-center py-2"><Spinner /></div>}
      {!loading && goodIds.length === 0 && (
        <p className="text-xs text-slate-600 italic mb-2">Keine Vorkommen</p>
      )}
      {!loading && goodIds.map(goodId => {
        const mineFacs  = minesByGood.get(goodId) ?? []
        const deposit   = deposits[goodId]
        const builds    = buildsByGood.get(goodId) ?? []
        const activeOrders = ordersByGood.get(goodId) ?? []

        const runningCount  = mineFacs.filter(m => m.status === 'running').length
        const idleNoOrder   = mineFacs.filter(m => m.status === 'idle' && m.current_order_id === null)
        const needsActivate = idleNoOrder.length > activeOrders.filter(o => o.status !== 'running').length
        const totalRate     = runningCount * mineRateLv1
        const maxSlots      = deposit?.max_rate ?? 0

        const initial  = deposit ? Math.max(deposit.max_rate * 10_000, deposit.remaining) : 0
        const pct      = initial > 0 && deposit ? Math.min(100, (deposit.remaining / initial) * 100) : 0
        const barColor = pct > 50 ? 'bg-amber-700/70' : pct > 20 ? 'bg-orange-700/70' : 'bg-red-700/70'
        const hasBuildRecipe = buildRecipes.some(r => r.product_id === `facility_mine_${goodId}`)

        return (
          <Card key={goodId}>
            {/* Header: lamps + name + capacity + rate */}
            <div className="flex items-center gap-1.5 flex-wrap mb-1">
              <div className="flex gap-1 items-center">
                {mineFacs.map(m => <StatusLamp key={m.id} status={m.status} />)}
                {builds.map(o => (
                  <span key={o.id} className="inline-block w-2 h-2 rounded-full bg-blue-500 flex-shrink-0" title="Im Bau" />
                ))}
              </div>
              <span className="text-xs font-bold text-slate-200">{itemLabel(goodId)}</span>
              {mineFacs.length > 0 && maxSlots > 0 && (
                <span className="text-xs text-slate-600">
                  {mineFacs.length}/{Math.floor(maxSlots)} Minen
                </span>
              )}
              {totalRate > 0 && (
                <span className="ml-auto text-xs font-mono text-emerald-500">
                  {totalRate.toFixed(1)} u/Tick
                </span>
              )}
            </div>

            {/* Deposit bar */}
            {deposit && (
              <div className="mb-1.5">
                <div className="flex justify-between text-xs text-slate-600 mb-0.5">
                  <span>Vorkommen</span>
                  <span className="tabular-nums">{deposit.remaining.toFixed(0)}</span>
                </div>
                <div className="h-1 bg-slate-800 rounded-full overflow-hidden">
                  <div className={`h-full rounded-full ${barColor}`} style={{ width: `${pct}%` }} />
                </div>
              </div>
            )}

            {/* In-progress build orders */}
            {builds.map(o => {
              const bPct = o.recipe_ticks > 0 ? Math.min((o.produced_qty / o.recipe_ticks) * 100, 100) : 0
              return (
                <div key={o.id} className="mb-1.5">
                  <div className="flex justify-between text-xs text-blue-400 mb-0.5">
                    <span>+1 Mine im Bau</span>
                    <span>{Math.round(o.produced_qty)}/{o.recipe_ticks} Ticks</span>
                  </div>
                  <div className="h-1 bg-slate-800 rounded overflow-hidden">
                    <div className="h-full bg-blue-700 rounded" style={{ width: `${bPct}%` }} />
                  </div>
                </div>
              )
            })}

            {/* Actions */}
            <div className="flex gap-1.5 mt-1 flex-wrap">
              {needsActivate && (
                <PrimaryButton onClick={() => handleActivate(goodId)} disabled={activating[goodId]}>
                  {activating[goodId] ? '…' : 'Mine starten'}
                </PrimaryButton>
              )}
              {hasBuildRecipe && (
                <button
                  onClick={() => handleBuild(goodId)}
                  disabled={building[goodId]}
                  className="text-xs px-2 py-0.5 rounded border border-slate-700 text-slate-400
                             hover:border-slate-500 disabled:opacity-40 transition-colors"
                >
                  {building[goodId] ? '…' : '+ Mine bauen'}
                </button>
              )}
            </div>

            {errors[goodId] && <p className="text-xs text-red-400 mt-0.5">{errors[goodId]}</p>}
          </Card>
        )
      })}
    </div>
  )
}

// ── Center column: ANLAGEN + AUFTRÄGE ─────────────────────────────────────────

interface AnlagenPanelProps {
  facilities: Facility[]
  orders: Order[]
  stock: ItemStock[]
  loading: boolean
  starId: string
  nodeId: string
  onRefresh: () => void
}

function AnlagenPanel({ facilities, orders, stock, loading, starId, nodeId, onRefresh }: AnlagenPanelProps) {
  const [showForm, setShowForm]     = useState(false)
  const [buildRecipes, setBuildRecipes] = useState<Recipe[]>([])
  const [selectedRecipeId, setSelectedRecipeId] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError]           = useState('')

  useEffect(() => {
    listRecipes()
      .then(rs => {
        const brs = rs.filter(r => r.factory_type === 'construction')
        setBuildRecipes(brs)
        if (brs.length > 0) setSelectedRecipeId(brs[0].recipe_id)
      })
      .catch(() => {})
  }, [])

  const orderByFacility = orders.reduce<Record<string, Order>>((acc, o) => {
    if (o.facility_id) acc[o.facility_id] = o
    return acc
  }, {})

  const buildOrders = orders.filter(o =>
    o.factory_type === 'construction' &&
    !o.product_id.startsWith('facility_mine_') &&
    !['completed', 'cancelled'].includes(o.status)
  )

  const stockMap = Object.fromEntries(stock.map(s => [s.item_id, s.available]))
  const selectedRecipe = buildRecipes.find(r => r.recipe_id === selectedRecipeId)

  async function handleBuild() {
    if (!selectedRecipe) return
    setSubmitting(true)
    setError('')
    try {
      await createOrder({
        node_id: nodeId,
        star_id: starId,
        factory_type: 'construction',
        product_id: selectedRecipe.product_id,
        order_type: 'build',
        target_qty: 1,
      })
      setShowForm(false)
      onRefresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fehler beim Anlegen')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDestroy(id: string) {
    try {
      await destroyFacility(id)
      onRefresh()
    } catch {
      // silently ignore
    }
  }

  return (
    <div>
      <SectionTitle>Anlagen</SectionTitle>
      {loading && <div className="flex justify-center py-4"><Spinner /></div>}
      {!loading && facilities.length === 0 && buildOrders.length === 0 && (
        <p className="text-xs text-slate-600 italic mb-2">Keine Anlagen</p>
      )}

      {/* Existing facilities (mines handled by MinenPanel) */}
      {!loading && facilities.filter(f => f.factory_type !== 'mine').map(f => {
        const activeOrder = f.current_order_id ? orderByFacility[f.id] ?? null : null
        const label = FACILITY_TYPE_LABELS[f.factory_type] ?? f.factory_type
        const depositLabel = f.config.deposit_good_id ? ` — ${itemLabel(f.config.deposit_good_id)}` : ''
        return (
          <Card key={f.id}>
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-xs font-mono font-bold text-slate-300 uppercase truncate">
                  {label}{depositLabel}
                </span>
                <StatusBadge status={f.status} colors={FACILITY_STATUS_COLORS} />
              </div>
              <DangerButton onClick={() => handleDestroy(f.id)}>Zerstören</DangerButton>
            </div>
            {f.status === 'running' && f.current_order_id && (
              <p className="text-xs text-slate-500 mt-0.5">
                ↳ {activeOrder ? (PRODUCT_LABELS[activeOrder.product_id] ?? activeOrder.product_id) : short(f.current_order_id)}
              </p>
            )}
          </Card>
        )
      })}

      {/* In-progress build orders */}
      {!loading && buildOrders.map(o => {
        const pct = o.recipe_ticks > 0 ? Math.min(o.produced_qty / o.recipe_ticks, 1) : 0
        return (
          <Card key={o.id} className="border-blue-900/50">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-xs text-blue-400 font-bold">
                {PRODUCT_LABELS[o.product_id] ?? o.product_id} — im Bau
              </span>
              <StatusBadge status={o.status} colors={ORDER_STATUS_COLORS} />
            </div>
            <div className="flex justify-between text-xs text-slate-600 mb-0.5">
              <span>Fortschritt</span>
              <span>{Math.round(o.produced_qty)}/{o.recipe_ticks} Ticks</span>
            </div>
            <div className="h-1 bg-slate-800 rounded overflow-hidden">
              <div
                className="h-full bg-blue-600 rounded transition-all"
                style={{ width: `${pct * 100}%` }}
              />
            </div>
          </Card>
        )
      })}

      {/* Build form */}
      {!showForm && (
        <PrimaryButton onClick={() => setShowForm(true)}>+ Anlage bauen</PrimaryButton>
      )}
      {showForm && (
        <FormWrapper>
          <label className="text-xs text-slate-500">Anlage</label>
          <Select value={selectedRecipeId} onChange={setSelectedRecipeId}>
            {buildRecipes.map(r => (
              <option key={r.recipe_id} value={r.recipe_id}>{recipeLabel(r)}</option>
            ))}
          </Select>

          {selectedRecipe && selectedRecipe.inputs.length > 0 && (
            <div className="mt-1">
              <div className="grid grid-cols-3 gap-x-2 text-xs text-slate-600 mb-1">
                <span>Ressource</span>
                <span className="text-right">Menge</span>
                <span className="text-right">Lager</span>
              </div>
              {selectedRecipe.inputs.map(inp => {
                const have = stockMap[inp.item_id] ?? 0
                const ok = have >= inp.amount
                return (
                  <div key={inp.item_id} className="grid grid-cols-3 gap-x-2 text-xs py-0.5">
                    <span className="text-slate-400 truncate">{itemLabel(inp.item_id)}</span>
                    <span className="text-right text-slate-500">{inp.amount}</span>
                    <span className={`text-right font-mono ${ok ? 'text-emerald-400' : 'text-red-400'}`}>
                      {have.toFixed(0)}
                      {!ok && <span className="text-red-600 ml-1">−{(inp.amount - have).toFixed(0)}</span>}
                    </span>
                  </div>
                )
              })}
              <p className="text-xs text-slate-600 mt-1">
                Baudauer: {selectedRecipe.ticks} Tick{selectedRecipe.ticks !== 1 ? 's' : ''}
              </p>
            </div>
          )}

          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2 mt-1">
            <PrimaryButton onClick={handleBuild} disabled={submitting || !selectedRecipe}>
              {submitting ? '…' : 'Bau starten'}
            </PrimaryButton>
            <CancelButton onClick={() => setShowForm(false)}>Abbrechen</CancelButton>
          </div>
        </FormWrapper>
      )}
    </div>
  )
}

interface AuftraegePanelProps {
  orders: Order[]
  loading: boolean
  nodeId: string
  starId: string
  onRefresh: () => void
}

function AuftraegePanel({ orders, loading, nodeId, starId, onRefresh }: AuftraegePanelProps) {
  const [showForm, setShowForm]         = useState(false)
  const [prodRecipes, setProdRecipes]   = useState<Recipe[]>([])
  const [selectedRecipeId, setSelectedRecipeId] = useState('')
  const [orderType, setOrderType]       = useState<'batch' | 'continuous'>('batch')
  const [targetQty, setTargetQty]       = useState('100')
  const [priority, setPriority]         = useState('5')
  const [submitting, setSubmitting]     = useState(false)
  const [error, setError]               = useState('')

  useEffect(() => {
    listRecipes()
      .then(rs => {
        const prs = rs.filter(r => r.factory_type !== 'construction' && r.factory_type !== 'mine')
        setProdRecipes(prs)
        if (prs.length > 0) setSelectedRecipeId(prs[0].recipe_id)
      })
      .catch(() => {})
  }, [])

  const selectedRecipe = prodRecipes.find(r => r.recipe_id === selectedRecipeId)

  // Show only non-construction, non-mine production orders
  const prodOrders = orders.filter(o => o.factory_type !== 'construction' && o.factory_type !== 'mine')

  async function handleCreate() {
    if (!selectedRecipe) return
    setSubmitting(true)
    setError('')
    try {
      await createOrder({
        node_id: nodeId,
        star_id: starId,
        factory_type: selectedRecipe.factory_type,
        product_id: selectedRecipe.product_id,
        order_type: orderType,
        target_qty: parseFloat(targetQty) || 100,
        priority: parseInt(priority) || 5,
      })
      setOrderType('batch')
      setTargetQty('100')
      setPriority('5')
      setShowForm(false)
      onRefresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fehler beim Erteilen')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleCancel(id: string) {
    try {
      await cancelOrder(id)
      onRefresh()
    } catch {
      // silently ignore
    }
  }

  return (
    <div>
      <SectionTitle>Aufträge</SectionTitle>
      {loading && <div className="flex justify-center py-4"><Spinner /></div>}
      {!loading && prodOrders.length === 0 && (
        <p className="text-xs text-slate-600 italic mb-2">Keine Aufträge</p>
      )}
      {!loading && prodOrders.map(o => {
        const progress = o.target_qty > 0 ? Math.min(o.produced_qty / o.target_qty, 1) : 0
        const isTerminal = TERMINAL_ORDER_STATUSES.has(o.status)
        return (
          <Card key={o.id}>
            <div className="flex items-center justify-between gap-2 flex-wrap">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-xs font-mono text-slate-200 truncate">
                  {PRODUCT_LABELS[o.product_id] ?? o.product_id}
                </span>
                <span className="text-xs px-1 py-0.5 rounded border border-slate-700 text-slate-500 font-bold">
                  {o.order_type === 'batch' ? 'BATCH' : 'CONT'}
                </span>
                <StatusBadge status={o.status} colors={ORDER_STATUS_COLORS} />
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs bg-slate-800 text-slate-500 px-1.5 py-0.5 rounded">
                  P{o.priority}
                </span>
                {!isTerminal && (
                  <DangerButton onClick={() => handleCancel(o.id)}>Abbrechen</DangerButton>
                )}
              </div>
            </div>

            {o.order_type === 'batch' && (
              <div className="mt-1.5">
                <div className="flex justify-between text-xs text-slate-600 mb-0.5">
                  <span>Fortschritt</span>
                  <span>{o.produced_qty} / {o.target_qty}</span>
                </div>
                <div className="h-1 bg-slate-800 rounded overflow-hidden">
                  <div
                    className="h-full bg-emerald-600 rounded transition-all"
                    style={{ width: `${progress * 100}%` }}
                  />
                </div>
              </div>
            )}

            {o.inputs && o.inputs.length > 0 && (
              <div className="flex flex-wrap gap-2 mt-1.5">
                {o.inputs.map(inp => {
                  const allocated = o.allocated_inputs?.[inp.item_id] ?? 0
                  const ok = allocated > 0
                  return (
                    <span
                      key={inp.item_id}
                      className={`text-xs ${ok ? 'text-emerald-400' : 'text-red-400'}`}
                    >
                      {itemLabel(inp.item_id)}: {ok ? '✓' : '✗'}
                    </span>
                  )
                })}
              </div>
            )}
          </Card>
        )
      })}

      {!showForm && (
        <PrimaryButton onClick={() => setShowForm(true)}>+ Auftrag erteilen</PrimaryButton>
      )}
      {showForm && (
        <FormWrapper>
          <label className="text-xs text-slate-500">Rezept</label>
          <Select value={selectedRecipeId} onChange={setSelectedRecipeId}>
            {prodRecipes.map(r => (
              <option key={r.recipe_id} value={r.recipe_id}>{recipeLabel(r)}</option>
            ))}
          </Select>

          {selectedRecipe && (
            <p className="text-xs text-slate-600 -mt-1">
              {selectedRecipe.inputs.map(i => `${i.amount}× ${itemLabel(i.item_id)}`).join(', ')}
              {' → '}{selectedRecipe.ticks} Tick{selectedRecipe.ticks !== 1 ? 's' : ''}
            </p>
          )}

          <label className="text-xs text-slate-500">Auftragstyp</label>
          <div className="flex gap-3">
            {(['batch', 'continuous'] as const).map(t => (
              <label key={t} className="flex items-center gap-1 text-xs text-slate-400 cursor-pointer">
                <input
                  type="radio"
                  value={t}
                  checked={orderType === t}
                  onChange={() => setOrderType(t)}
                  className="accent-emerald-400"
                />
                {t === 'batch' ? 'Batch' : 'Kontinuierlich'}
              </label>
            ))}
          </div>
          {orderType === 'batch' && (
            <>
              <label className="text-xs text-slate-500">Zielmenge</label>
              <NumberInput value={targetQty} onChange={setTargetQty} placeholder="100" min={1} />
            </>
          )}
          <label className="text-xs text-slate-500">Priorität (1–10)</label>
          <NumberInput value={priority} onChange={setPriority} min={1} max={10} />
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <PrimaryButton onClick={handleCreate} disabled={submitting || !selectedRecipe}>
              {submitting ? '…' : 'Erteilen'}
            </PrimaryButton>
            <CancelButton onClick={() => setShowForm(false)}>Abbrechen</CancelButton>
          </div>
        </FormWrapper>
      )}
    </div>
  )
}

// ── Right column: TRANSPORT ───────────────────────────────────────────────────

interface TransportPanelProps {
  routes: Route[]
  loading: boolean
  onRefresh: () => void
}

function TransportPanel({ routes, loading, onRefresh }: TransportPanelProps) {
  const [showForm, setShowForm] = useState(false)
  const [fromNodeId, setFromNodeId] = useState('')
  const [toNodeId, setToNodeId] = useState('')
  const [capacity, setCapacity] = useState('10')
  const [minShare, setMinShare] = useState('0.2')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleCreate() {
    if (!fromNodeId.trim() || !toNodeId.trim()) return
    setSubmitting(true)
    setError('')
    try {
      await createRoute({
        from_node_id: fromNodeId.trim(),
        to_node_id: toNodeId.trim(),
        capacity_per_tick: parseFloat(capacity) || 10,
        min_continuous_share: parseFloat(minShare) || 0.2,
      })
      setFromNodeId('')
      setToNodeId('')
      setCapacity('10')
      setMinShare('0.2')
      setShowForm(false)
      onRefresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fehler beim Anlegen')
    } finally {
      setSubmitting(false)
    }
  }

  const ROUTE_STATUS_COLORS: Record<string, string> = {
    active:    'bg-emerald-900/60 text-emerald-400',
    suspended: 'bg-orange-900/60 text-orange-400',
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <SectionTitle>Transport</SectionTitle>
      {loading && <div className="flex justify-center py-4"><Spinner /></div>}
      {!loading && routes.length === 0 && (
        <p className="text-xs text-slate-600 italic mb-2">Keine Routen</p>
      )}
      {!loading && (
        <div className="overflow-y-auto flex-1">
          {routes.map(r => (
            <Card key={r.id}>
              <div className="flex items-center justify-between gap-1 mb-1">
                <span className="text-xs font-mono text-slate-300">
                  {short(r.from_node_id)} → {short(r.to_node_id)}
                </span>
                <StatusBadge status={r.status} colors={ROUTE_STATUS_COLORS} />
              </div>
              <div className="text-xs text-slate-500">
                {r.capacity_per_tick}/tick · min {Math.round(r.min_continuous_share * 100)}% kont.
              </div>
            </Card>
          ))}
        </div>
      )}

      {!showForm && (
        <PrimaryButton onClick={() => setShowForm(true)}>+ Route anlegen</PrimaryButton>
      )}
      {showForm && (
        <FormWrapper>
          <label className="text-xs text-slate-500">Von Node ID</label>
          <TextInput value={fromNodeId} onChange={setFromNodeId} placeholder="UUID …" />
          <label className="text-xs text-slate-500">Zu Node ID</label>
          <TextInput value={toNodeId} onChange={setToNodeId} placeholder="UUID …" />
          <label className="text-xs text-slate-500">Kapazität/Tick</label>
          <NumberInput value={capacity} onChange={setCapacity} min={1} />
          <label className="text-xs text-slate-500">Min. kontinuierlicher Anteil (0–1)</label>
          <NumberInput value={minShare} onChange={setMinShare} min={0} max={1} step={0.05} />
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <PrimaryButton onClick={handleCreate} disabled={submitting}>
              {submitting ? '…' : 'Anlegen'}
            </PrimaryButton>
            <CancelButton onClick={() => setShowForm(false)}>Abbrechen</CancelButton>
          </div>
        </FormWrapper>
      )}
    </div>
  )
}

// ── Tick Generator ────────────────────────────────────────────────────────────

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
    <div className="flex items-center gap-1.5 ml-auto text-xs font-mono select-none">
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

// ── Meine Assets Übersicht ────────────────────────────────────────────────────

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

// ── Planetare Vorkommen ───────────────────────────────────────────────────────

function _VorkommenPanel({ deposits }: { deposits: Record<string, DepositEntry> }) {
  const entries = Object.entries(deposits).filter(([, d]) => d.remaining > 0)
  if (entries.length === 0) return (
    <div>
      <div className="text-xs font-bold tracking-widest uppercase text-slate-600 mb-2">Vorkommen</div>
      <div className="text-xs text-slate-700 italic">Keine Daten</div>
    </div>
  )
  return (
    <div>
      <div className="text-xs font-bold tracking-widest uppercase text-slate-600 mb-2">Vorkommen</div>
      <div className="space-y-2">
        {entries
          .sort((a, b) => b[1].remaining - a[1].remaining)
          .map(([goodId, d]) => {
            // initial ≈ max_rate × 10 000 (derived from EnsureDeposits scaling)
            const initial = Math.max(d.max_rate * 10_000, d.remaining)
            const pct = initial > 0 ? Math.min(100, (d.remaining / initial) * 100) : 0
            const barColor = pct > 50 ? 'bg-amber-700/70' : pct > 20 ? 'bg-orange-700/70' : 'bg-red-700/70'
            return (
              <div key={goodId}>
                <div className="flex justify-between text-xs mb-0.5">
                  <span className="text-slate-400">{itemLabel(goodId)}</span>
                  <span className="text-slate-500 tabular-nums">{d.remaining.toFixed(0)}</span>
                </div>
                <div className="h-1 bg-slate-800 rounded-full overflow-hidden">
                  <div className={`h-full rounded-full transition-all ${barColor}`}
                    style={{ width: `${pct}%` }} />
                </div>
              </div>
            )
          })}
      </div>
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export function Economy2Page() {
  const [nodeId, setNodeId]       = useState('')
  const [starId, setStarId]       = useState('')
  const [stock, setStock]         = useState<ItemStock[]>([])
  const [facilities, setFacilities] = useState<Facility[]>([])
  const [orders, setOrders]       = useState<Order[]>([])
  const [routes, setRoutes]       = useState<Route[]>([])
  const [deposits, setDeposits]   = useState<Record<string, DepositEntry>>({})
  const [mineRateLv1, setMineRateLv1] = useState(2.5)
  const [loading, setLoading]     = useState(false)
  const [error, setError]         = useState('')
  const [bootstrapping, setBootstrapping] = useState(false)
  const [bootstrapMsg, setBootstrapMsg]   = useState('')
  const [starType, setStarType]           = useState('')

  // ── Tick Generator state (single instance, persists across view changes) ──
  const [tickSpeed, setTickSpeed]     = useState(1)
  const [tickRunning, setTickRunning] = useState(false)
  const [currentTick, setCurrentTick] = useState<number | null>(null)
  const tickIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    fetch('/api/v2/admin/tick/current').then(r => r.json()).then(d => setCurrentTick(d.tick)).catch(() => {})
  }, [])

  useEffect(() => {
    if (tickIntervalRef.current) clearInterval(tickIntervalRef.current)
    if (!tickRunning) { tickIntervalRef.current = null; return }
    const ms = Math.round(1000 / tickSpeed)
    tickIntervalRef.current = setInterval(() => {
      fetch('/api/v2/admin/tick/advance', { method: 'POST' })
        .then(r => r.json())
        .then(d => setCurrentTick(d.tick))
        .catch(() => {})
    }, ms)
    return () => { if (tickIntervalRef.current) { clearInterval(tickIntervalRef.current); tickIntervalRef.current = null } }
  }, [tickRunning, tickSpeed])

  const nodeIdRef = useRef(nodeId)
  const starIdRef = useRef(starId)
  nodeIdRef.current = nodeId
  starIdRef.current = starId

  const loadData = useCallback(async (nid: string, sid: string) => {
    if (!nid) return
    setLoading(true)
    setError('')
    try {
      const [s, f, o, r, dep] = await Promise.all([
        getStock(nid),
        sid ? listFacilities(sid) : Promise.resolve([]),
        listOrders(nid),
        listRoutes(),
        sid ? getDeposits(sid).then(res => { if (res.mine_rate_lv1) setMineRateLv1(res.mine_rate_lv1); return res.deposits ?? {} }).catch(() => ({})) : Promise.resolve({}),
      ])
      setStock(s)
      setFacilities(f)
      setOrders(o)
      setRoutes(r)
      setDeposits(dep)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ladefehler')
    } finally {
      setLoading(false)
    }
  }, [])

  // ── NATS Live-Updates — reload on every server tick ───────────────────────
  const natsStatus = useNatsTick((tickN) => {
    setCurrentTick(tickN)
    if (nodeIdRef.current) {
      loadData(nodeIdRef.current, starIdRef.current)
    }
  })

  function handleRefresh() {
    loadData(nodeId, starId)
  }

  async function handleBootstrap() {
    if (!starId.trim()) return
    setBootstrapping(true)
    setBootstrapMsg('')
    setError('')
    try {
      const result = await bootstrap(starId.trim())
      setNodeId(result.node_id)
      setBootstrapMsg(`Kit gesetzt: ${result.seeded_facilities} Anlagen, ${Object.keys(result.seeded_stock).length} Güter`)
      await loadData(result.node_id, starId.trim())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Bootstrap fehlgeschlagen')
    } finally {
      setBootstrapping(false)
    }
  }

  const tickProps: TickGeneratorProps = {
    speed: tickSpeed,
    onSetSpeed: setTickSpeed,
    running: tickRunning,
    onToggle: () => setTickRunning(r => !r),
    currentTick,
  }

  return (
    <div className="absolute inset-0 top-0 flex flex-col font-mono bg-slate-950">
      {/* Single top bar — always visible, single TickGenerator instance */}
      <div className="flex items-center gap-3 px-4 py-2 bg-black/70 border-b border-slate-800 backdrop-blur-sm flex-shrink-0">
        {!nodeId ? (
          <span className="text-xs font-bold tracking-widest text-emerald-500 uppercase">Meine Assets</span>
        ) : (
          <>
            <button
              onClick={() => { setNodeId(''); setStarId(''); setDeposits({}) }}
              className="text-xs text-slate-500 hover:text-slate-300 transition-colors mr-1"
              title="Zurück zur Übersicht"
            >← Assets</button>
            <span className="text-xs text-slate-700">|</span>
            <span className="text-xs text-slate-400 font-mono">
              {starType ? (STAR_TYPE_LABELS[starType] ?? starType) : 'Stern'} · {starId.slice(0, 8)}…
            </span>
          </>
        )}
        <TickGenerator {...tickProps} />
        {nodeId && (
          <>
            <button
              onClick={handleBootstrap}
              disabled={bootstrapping || !starId.trim()}
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
        {bootstrapMsg && <span className="text-xs text-blue-400">{bootstrapMsg}</span>}
        {error && <span className="text-xs text-red-400">{error}</span>}
      </div>

      {/* Content */}
      {!nodeId ? (
        <MyAssetsView onSelect={n => { setStarId(n.star_id); setNodeId(n.node_id); setStarType(n.star_type); loadData(n.node_id, n.star_id) }} />
      ) : (
        <div className="flex flex-1 overflow-hidden">
          {/* Left — LAGER + VORKOMMEN */}
          <div className="w-56 flex-shrink-0 bg-black/70 border-r border-slate-800 backdrop-blur-sm overflow-y-auto p-3 space-y-4">
            <LagerPanel stock={stock} loading={loading} />
          </div>

          {/* Center — MINEN + ANLAGEN + AUFTRÄGE */}
          <div className="flex-1 overflow-y-auto p-3 space-y-4">
            <MinenPanel
              facilities={facilities}
              orders={orders}
              deposits={deposits}
              mineRateLv1={mineRateLv1}
              nodeId={nodeId}
              starId={starId}
              loading={loading}
              onRefresh={handleRefresh}
            />
            <div className="border-t border-slate-800" />
            <AnlagenPanel
              facilities={facilities}
              orders={orders}
              stock={stock}
              loading={loading}
              starId={starId}
              nodeId={nodeId}
              onRefresh={handleRefresh}
            />
            <div className="border-t border-slate-800" />
            <AuftraegePanel
              orders={orders}
              loading={loading}
              nodeId={nodeId}
              starId={starId}
              onRefresh={handleRefresh}
            />
          </div>

          {/* Right — TRANSPORT */}
          <div className="w-56 flex-shrink-0 bg-black/70 border-l border-slate-800 backdrop-blur-sm overflow-y-auto p-3">
            <TransportPanel
              routes={routes}
              loading={loading}
              onRefresh={handleRefresh}
            />
          </div>
        </div>
      )}
    </div>
  )
}
