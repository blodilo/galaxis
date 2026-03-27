import { useState, useEffect, useCallback, useRef } from 'react'
import type { ItemStock, Facility, Order, Route } from '../types/economy2'
import {
  getStock,
  createFacility,
  listFacilities,
  destroyFacility,
  createOrder,
  listOrders,
  cancelOrder,
  createRoute,
  listRoutes,
  bootstrap,
} from '../api/economy2'

// ── Facility catalog ──────────────────────────────────────────────────────────

const FACTORY_TYPES = [
  { id: 'mine',          label: 'Mine',               description: 'Abbau geologischer Rohstoffe aus Planetenlagerstätten' },
  { id: 'smelter',       label: 'Schmelze',            description: 'Verhüttung: Eisenerz → Stahl und Titanstahl' },
  { id: 'refinery',      label: 'Raffinerie',          description: 'Veredelung zu Halbleitern und Fusionskraftstoff' },
  { id: 'precision_fab', label: 'Präzisionsfertigung', description: 'Hochpräzise Bauteile und Navigationscomputer' },
]

const MINE_GOODS = [
  { id: 'iron_ore',    label: 'Eisenerz' },
  { id: 'silicates',   label: 'Silikate' },
  { id: 'titan',       label: 'Titan' },
  { id: 'rare_earths', label: 'Seltene Erden' },
  { id: 'he3',         label: 'Helium-3' },
  { id: 'hydrogen',    label: 'Wasserstoff' },
]

// Baukosten — Platzhalter, zieht später aus game-params YAML
const BUILD_COSTS: Record<string, Record<string, number>> = {
  mine:          { steel: 10,  base_component: 2  },
  smelter:       { steel: 25,  base_component: 5  },
  refinery:      { titansteel: 15, semiconductor_wafer: 5,  base_component: 8  },
  precision_fab: { titansteel: 20, semiconductor_wafer: 10, base_component: 15 },
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

function Stepper({ value, onChange, min = 1, max = 10 }: {
  value: number; onChange: (v: number) => void; min?: number; max?: number
}) {
  return (
    <div className="flex items-center gap-2">
      <button
        onClick={() => onChange(Math.max(min, value - 1))}
        disabled={value <= min}
        className="w-6 h-6 flex items-center justify-center rounded border border-slate-700
                   text-slate-400 hover:border-slate-500 disabled:opacity-30 transition-colors text-sm"
      >−</button>
      <span className="text-sm text-slate-200 w-6 text-center font-mono">{value}</span>
      <button
        onClick={() => onChange(Math.min(max, value + 1))}
        disabled={value >= max}
        className="w-6 h-6 flex items-center justify-center rounded border border-slate-700
                   text-slate-400 hover:border-slate-500 disabled:opacity-30 transition-colors text-sm"
      >+</button>
    </div>
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

function AnlagenPanel({ facilities, orders, stock, loading, starId, onRefresh }: AnlagenPanelProps) {
  const [showForm, setShowForm] = useState(false)
  const [factoryType, setFactoryType] = useState(FACTORY_TYPES[0].id)
  const [depositGoodId, setDepositGoodId] = useState(MINE_GOODS[0].id)
  const [qty, setQty] = useState(1)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const orderByFacility = orders.reduce<Record<string, Order>>((acc, o) => {
    if (o.facility_id) acc[o.facility_id] = o
    return acc
  }, {})

  const stockMap = Object.fromEntries(stock.map(s => [s.item_id, s.available]))
  const costs = BUILD_COSTS[factoryType] ?? {}
  const costEntries = Object.entries(costs)
  const selectedFacilityDef = FACTORY_TYPES.find(f => f.id === factoryType)

  async function handleCreate() {
    setSubmitting(true)
    setError('')
    try {
      for (let i = 0; i < qty; i++) {
        await createFacility({
          star_id: starId,
          factory_type: factoryType,
          ...(factoryType === 'mine' ? { deposit_good_id: depositGoodId } : {}),
        })
      }
      setShowForm(false)
      setQty(1)
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
      {!loading && facilities.length === 0 && (
        <p className="text-xs text-slate-600 italic mb-2">Keine Anlagen</p>
      )}
      {!loading && facilities.map(f => {
        const activeOrder = f.current_order_id ? orderByFacility[f.id] ?? null : null
        const label = FACTORY_TYPES.find(ft => ft.id === f.factory_type)?.label ?? f.factory_type
        return (
          <Card key={f.id}>
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-xs font-mono font-bold text-slate-300 uppercase truncate">
                  {label}
                </span>
                <StatusBadge status={f.status} colors={FACILITY_STATUS_COLORS} />
              </div>
              <DangerButton onClick={() => handleDestroy(f.id)}>Zerstören</DangerButton>
            </div>
            {f.status === 'running' && f.current_order_id && (
              <p className="text-xs text-slate-500 mt-0.5">
                ↳ {activeOrder ? activeOrder.product_id : short(f.current_order_id)}
              </p>
            )}
          </Card>
        )
      })}

      {!showForm && (
        <PrimaryButton onClick={() => setShowForm(true)}>+ Anlage bauen</PrimaryButton>
      )}
      {showForm && (
        <FormWrapper>
          {/* Anlagentyp */}
          <label className="text-xs text-slate-500">Anlagentyp</label>
          <Select value={factoryType} onChange={v => { setFactoryType(v); setQty(1) }}>
            {FACTORY_TYPES.map(ft => (
              <option key={ft.id} value={ft.id}>{ft.label}</option>
            ))}
          </Select>
          {selectedFacilityDef && (
            <p className="text-xs text-slate-600 italic -mt-1">{selectedFacilityDef.description}</p>
          )}

          {/* Mine: Lagerstätte wählen */}
          {factoryType === 'mine' && (
            <>
              <label className="text-xs text-slate-500">Lagerstätte</label>
              <Select value={depositGoodId} onChange={setDepositGoodId}>
                {MINE_GOODS.map(g => (
                  <option key={g.id} value={g.id}>{g.label}</option>
                ))}
              </Select>
            </>
          )}

          {/* Anzahl */}
          <label className="text-xs text-slate-500">Anzahl</label>
          <Stepper value={qty} onChange={setQty} min={1} max={10} />

          {/* Baukosten */}
          {costEntries.length > 0 && (
            <div className="mt-1">
              <div className="grid grid-cols-3 gap-x-2 text-xs text-slate-600 mb-1">
                <span>Ressource</span>
                <span className="text-right">pro Stück</span>
                <span className="text-right">Gesamt</span>
              </div>
              {costEntries.map(([res, amt]) => {
                const total = amt * qty
                const have = stockMap[res] ?? 0
                const ok = have >= total
                return (
                  <div key={res} className="grid grid-cols-3 gap-x-2 text-xs py-0.5">
                    <span className="text-slate-400 font-mono truncate">{res}</span>
                    <span className="text-right text-slate-500">{amt}</span>
                    <span className={`text-right font-mono ${ok ? 'text-emerald-400' : 'text-red-400'}`}>
                      {total}
                      {!ok && (
                        <span className="text-red-600 ml-1">−{(total - have).toFixed(0)}</span>
                      )}
                    </span>
                  </div>
                )
              })}
            </div>
          )}

          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2 mt-1">
            <PrimaryButton onClick={handleCreate} disabled={submitting || !starId}>
              {submitting ? '…' : qty > 1 ? `${qty}× Anlegen` : 'Anlegen'}
            </PrimaryButton>
            <CancelButton onClick={() => { setShowForm(false); setQty(1) }}>Abbrechen</CancelButton>
          </div>
          {!starId && (
            <p className="text-xs text-orange-400">Star-ID erforderlich</p>
          )}
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
  const [showForm, setShowForm] = useState(false)
  const [productId, setProductId] = useState('')
  const [factoryType, setFactoryType] = useState('')
  const [orderType, setOrderType] = useState<'batch' | 'continuous'>('batch')
  const [targetQty, setTargetQty] = useState('100')
  const [priority, setPriority] = useState('5')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleCreate() {
    if (!productId.trim() || !factoryType.trim()) return
    setSubmitting(true)
    setError('')
    try {
      await createOrder({
        node_id: nodeId,
        star_id: starId,
        factory_type: factoryType.trim(),
        product_id: productId.trim(),
        order_type: orderType,
        target_qty: parseFloat(targetQty) || 100,
        priority: parseInt(priority) || 5,
      })
      setProductId('')
      setFactoryType('')
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
      {!loading && orders.length === 0 && (
        <p className="text-xs text-slate-600 italic mb-2">Keine Aufträge</p>
      )}
      {!loading && orders.map(o => {
        const progress = o.target_qty > 0 ? Math.min(o.produced_qty / o.target_qty, 1) : 0
        const isTerminal = TERMINAL_ORDER_STATUSES.has(o.status)
        return (
          <Card key={o.id}>
            <div className="flex items-center justify-between gap-2 flex-wrap">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-xs font-mono text-slate-200 truncate">{o.product_id}</span>
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
                      {inp.item_id}: {ok ? '✓' : '✗'}
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
          <label className="text-xs text-slate-500">Produkt ID</label>
          <TextInput value={productId} onChange={setProductId} placeholder="iron_ingot …" />
          <label className="text-xs text-slate-500">Anlagentyp</label>
          <TextInput value={factoryType} onChange={setFactoryType} placeholder="smelter …" />
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
            <PrimaryButton onClick={handleCreate} disabled={submitting}>
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

// ── Main page ─────────────────────────────────────────────────────────────────

const AUTO_REFRESH_SEC = 10

export function Economy2Page() {
  const [nodeId, setNodeId]       = useState('')
  const [starId, setStarId]       = useState('')
  const [stock, setStock]         = useState<ItemStock[]>([])
  const [facilities, setFacilities] = useState<Facility[]>([])
  const [orders, setOrders]       = useState<Order[]>([])
  const [routes, setRoutes]       = useState<Route[]>([])
  const [loading, setLoading]     = useState(false)
  const [error, setError]         = useState('')
  const [countdown, setCountdown] = useState(AUTO_REFRESH_SEC)
  const [bootstrapping, setBootstrapping] = useState(false)
  const [bootstrapMsg, setBootstrapMsg]   = useState('')

  const loadData = useCallback(async (nid: string, sid: string) => {
    if (!nid) return
    setLoading(true)
    setError('')
    try {
      const [s, f, o, r] = await Promise.all([
        getStock(nid),
        sid ? listFacilities(sid) : Promise.resolve([]),
        listOrders(nid),
        listRoutes(),
      ])
      setStock(s)
      setFacilities(f)
      setOrders(o)
      setRoutes(r)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ladefehler')
    } finally {
      setLoading(false)
    }
  }, [])

  // Auto-refresh countdown
  const countdownRef = useRef(AUTO_REFRESH_SEC)
  useEffect(() => {
    if (!nodeId) return
    countdownRef.current = AUTO_REFRESH_SEC
    setCountdown(AUTO_REFRESH_SEC)

    const tick = setInterval(() => {
      countdownRef.current -= 1
      setCountdown(countdownRef.current)
      if (countdownRef.current <= 0) {
        countdownRef.current = AUTO_REFRESH_SEC
        setCountdown(AUTO_REFRESH_SEC)
        loadData(nodeId, starId)
      }
    }, 1000)

    return () => clearInterval(tick)
  }, [nodeId, starId, loadData])

  function handleRefresh() {
    countdownRef.current = AUTO_REFRESH_SEC
    setCountdown(AUTO_REFRESH_SEC)
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

  return (
    <div className="absolute inset-0 top-0 flex flex-col font-mono">
      {/* Page top bar */}
      <div className="flex items-center gap-3 px-4 py-2 bg-black/70 border-b border-slate-800 backdrop-blur-sm flex-shrink-0">
        <span className="text-xs text-slate-500 uppercase tracking-widest">Node:</span>
        <input
          type="text"
          value={nodeId}
          onChange={e => setNodeId(e.target.value)}
          placeholder="Node UUID …"
          className="bg-black border border-slate-700 rounded px-2 py-1 text-xs text-slate-300 w-72"
        />
        <span className="text-xs text-slate-500 uppercase tracking-widest">Star:</span>
        <input
          type="text"
          value={starId}
          onChange={e => setStarId(e.target.value)}
          placeholder="Star UUID …"
          className="bg-black border border-slate-700 rounded px-2 py-1 text-xs text-slate-300 w-72"
        />
        <button
          onClick={handleBootstrap}
          disabled={bootstrapping || !starId.trim()}
          className="text-xs px-2 py-0.5 rounded border border-blue-700 text-blue-400
                     hover:bg-blue-900/30 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          {bootstrapping ? '…' : '⬡ Spielstart-Kit'}
        </button>
        <PrimaryButton onClick={handleRefresh} disabled={!nodeId}>
          Aktualisieren
        </PrimaryButton>
        {nodeId && (
          <span className="text-xs text-slate-600 ml-2">
            Auto-Refresh in {countdown}s
          </span>
        )}
        {bootstrapMsg && <span className="text-xs text-blue-400 ml-2">{bootstrapMsg}</span>}
        {error && <span className="text-xs text-red-400 ml-2">{error}</span>}
      </div>

      {/* Three-column layout */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left — LAGER */}
        <div className="w-56 flex-shrink-0 bg-black/70 border-r border-slate-800 backdrop-blur-sm overflow-y-auto p-3">
          <LagerPanel stock={stock} loading={loading} />
        </div>

        {/* Center — ANLAGEN + AUFTRÄGE */}
        <div className="flex-1 overflow-y-auto p-3 space-y-4">
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
    </div>
  )
}
