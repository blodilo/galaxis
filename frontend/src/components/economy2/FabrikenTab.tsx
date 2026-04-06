import { useState } from 'react'
import type { Facility, Order, Recipe, AggregatedStock } from '../../types/economy2'
import type { MyNodeEntry } from '../../api/economy2'
import { SectionTitle, Card, StatusLamp, StatusBadge, PrimaryButton, DangerButton, GhostButton, itemLabel, factoryLabel } from './ui'
import ProductionGraph from './ProductionGraph'

interface FabrikenTabProps {
  nodes: MyNodeEntry[]
  facilities: Facility[]
  orders: Order[]
  recipes: Recipe[]
  stockAll: AggregatedStock[]
  onStartFacility: (facility: Facility) => Promise<void>
  onStopFacility: (facility: Facility) => Promise<void>
  onRefresh: () => void
}

function facilityCurrentProduct(facility: Facility, orders: Order[]): string | null {
  // For extractors, the config always knows what's being extracted — more reliable than order lookup
  if (facility.config.deposit_good_id) return itemLabel(facility.config.deposit_good_id)
  if (!facility.current_order_id) return null
  const order = orders.find(o => o.id === facility.current_order_id)
  return order ? itemLabel(order.product_id) : null
}

function facilityRate(facility: Facility, orders: Order[]): string {
  const order = orders.find(o => o.id === facility.current_order_id)
  if (!order || order.recipe_ticks <= 0) return '—'
  const rate = order.base_yield / order.recipe_ticks
  return `${rate.toFixed(2)} u/tick`
}

function statusColor(status: string): 'green' | 'yellow' | 'red' | 'slate' {
  if (status === 'running') return 'green'
  if (['paused_input', 'paused_depleted'].includes(status)) return 'yellow'
  if (status === 'building') return 'slate'
  return 'slate'
}

function FacilityRow({ facility, orders, onStartFacility, onStopFacility }: {
  facility: Facility
  orders: Order[]
  onStartFacility: (f: Facility) => Promise<void>
  onStopFacility: (f: Facility) => Promise<void>
}) {
  const product = facilityCurrentProduct(facility, orders)
  const rate = facilityRate(facility, orders)
  const blockingOrder = orders.find(o => o.id === facility.current_order_id && o.status === 'waiting')

  return (
    <div className="flex items-center gap-3 py-1.5 px-2 hover:bg-slate-800/30 rounded text-xs border-b border-slate-800/50 last:border-0">
      <StatusLamp status={facility.status} />

      <span className="w-32 text-slate-300 font-medium">
        {factoryLabel(facility.factory_type)}
        {facility.config.deposit_good_id && (
          <span className="text-slate-500 ml-1">({itemLabel(facility.config.deposit_good_id)})</span>
        )}
      </span>

      <span className="w-24">
        <StatusBadge
          label={facility.status === 'idle' ? 'idle' : facility.status}
          color={statusColor(facility.status)}
        />
      </span>

      <span className="flex-1 text-slate-400">
        {product ? (
          <span>
            {product}
            {rate !== '—' && <span className="text-slate-600 ml-2">{rate}</span>}
          </span>
        ) : (
          <span className="text-slate-600">kein Auftrag</span>
        )}
        {blockingOrder && (
          <span className="text-yellow-400 ml-2">wartet auf Input</span>
        )}
      </span>

      <div className="flex gap-1">
        {facility.status === 'idle' && (
          <PrimaryButton onClick={() => onStartFacility(facility)}>
            Start
          </PrimaryButton>
        )}
        {facility.status === 'running' && (
          <PrimaryButton onClick={() => onStopFacility(facility)}>
            Stop
          </PrimaryButton>
        )}
      </div>
    </div>
  )
}

function FacilityTableHeader() {
  return (
    <div className="flex items-center gap-3 py-1 px-2 text-xs text-slate-600 border-b border-slate-800 mb-0.5">
      <span className="w-2 flex-shrink-0" />
      <span className="w-32">Anlage</span>
      <span className="w-24">Status</span>
      <span className="flex-1">Produkt · Rate</span>
      <span className="w-20 text-right">Aktionen</span>
    </div>
  )
}

function StarGroup({ starId, nodes, facilities, orders, onStartFacility, onStopFacility }: {
  starId: string
  nodes: MyNodeEntry[]
  facilities: Facility[]
  orders: Order[]
  onStartFacility: (f: Facility) => Promise<void>
  onStopFacility: (f: Facility) => Promise<void>
}) {
  const starFacilities = facilities.filter(f => f.star_id === starId)
  if (starFacilities.length === 0) return null

  const node = nodes.find(n => n.star_id === starId)
  const running = starFacilities.filter(f => f.status === 'running').length
  const idle = starFacilities.filter(f => f.status === 'idle').length
  const blocked = starFacilities.filter(f => ['paused_input', 'paused_depleted'].includes(f.status)).length

  const healthDot = blocked > 0 ? 'text-yellow-400' : running > 0 ? 'text-emerald-400' : 'text-slate-500'

  return (
    <div className="mb-4">
      <div className="flex items-center gap-2 mb-1.5">
        <span className={`text-sm font-medium ${healthDot}`}>●</span>
        <span className="text-sm font-medium text-slate-200">
          {node?.star_type ?? '?'} · {starId.slice(0, 8)}
        </span>
        <span className="text-slate-500 text-xs">
          {starFacilities.length} Anlagen
          {running > 0 && ` · ${running} laufen`}
          {idle > 0 && ` · ${idle} idle`}
          {blocked > 0 && ` · ${blocked} blockiert`}
        </span>
      </div>

      <div className="border border-slate-800 rounded">
        <FacilityTableHeader />
        {starFacilities.map(f => (
          <FacilityRow
            key={f.id}
            facility={f}
            orders={orders}
            onStartFacility={onStartFacility}
            onStopFacility={onStopFacility}
          />
        ))}
      </div>
    </div>
  )
}

export default function FabrikenTab({
  nodes,
  facilities,
  orders,
  recipes,
  stockAll,
  onStartFacility,
  onStopFacility,
  onRefresh,
}: FabrikenTabProps) {
  const [view, setView] = useState<'list' | 'graph'>('list')
  const starIds = [...new Set(facilities.map(f => f.star_id))]

  if (starIds.length === 0) {
    return (
      <div className="text-slate-500 text-sm p-4">
        Keine Anlagen vorhanden. Starte mit einem Bootstrap-Paket.
      </div>
    )
  }

  return (
    <div className="p-4">
      <div className="flex items-center gap-2 mb-3">
        <SectionTitle>Alle Anlagen</SectionTitle>
        <div className="ml-auto flex gap-1">
          <GhostButton onClick={() => setView('list')} disabled={view === 'list'}>Liste</GhostButton>
          <GhostButton onClick={() => setView('graph')} disabled={view === 'graph'}>Graph</GhostButton>
        </div>
      </div>

      {view === 'graph' && (
        <ProductionGraph
          orders={orders}
          facilities={facilities}
          stockAll={stockAll}
          recipes={recipes}
          height={560}
        />
      )}

      {view === 'list' && starIds.map(starId => (
        <StarGroup
          key={starId}
          starId={starId}
          nodes={nodes}
          facilities={facilities}
          orders={orders}
          onStartFacility={onStartFacility}
          onStopFacility={onStopFacility}
        />
      ))}
    </div>
  )
}
