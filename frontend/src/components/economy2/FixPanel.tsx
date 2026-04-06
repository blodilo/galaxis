import { useState } from 'react'
import type { BOMNode, Goal, AggregatedStock, Route } from '../../types/economy2'
import type { Facility } from '../../types/economy2'
import type { BOMContext } from './BOMTree'
import { PrimaryButton, GhostButton, itemLabel, factoryLabel } from './ui'
import { createOrder, createRoute } from '../../api/economy2'

interface FixPanelProps {
  node: BOMNode
  goal: Goal
  ctx: BOMContext
  nodes: Array<{ node_id: string; star_id: string; x: number; y: number; star_type: string }>
  onSetOverride: (itemId: string, starId: string) => void
  onClose: () => void
  onRefresh: () => void
}

export default function FixPanel({ node, goal, ctx, nodes, onSetOverride, onClose, onRefresh }: FixPanelProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const starName = (starId: string) => {
    const n = nodes.find(nd => nd.star_id === starId)
    return n ? `${n.star_type} ${n.star_id.slice(0, 6)}` : starId.slice(0, 8)
  }

  const nodeForStar = (starId: string) => nodes.find(n => n.star_id === starId)

  async function handleCreateOrder() {
    if (!node.recipe) return
    const targetNode = nodeForStar(goal.star_id)
    if (!targetNode) return
    setLoading(true)
    setError(null)
    try {
      await createOrder({
        node_id: targetNode.node_id,
        star_id: goal.star_id,
        factory_type: node.recipe.factory_type,
        product_id: node.item_id,
        order_type: 'batch',
        target_qty: node.qty,
        priority: goal.priority,
      })
      onRefresh()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fehler')
    } finally {
      setLoading(false)
    }
  }

  async function handleCreateRoute(fromNodeId: string, capacity: number) {
    const targetNode = nodeForStar(goal.star_id)
    if (!targetNode) return
    setLoading(true)
    setError(null)
    try {
      await createRoute({
        from_node_id: fromNodeId,
        to_node_id: targetNode.node_id,
        capacity_per_tick: capacity,
      })
      onRefresh()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fehler')
    } finally {
      setLoading(false)
    }
  }

  const status = node.status

  return (
    <div className="bg-slate-900 border border-slate-700 rounded p-3 text-xs text-slate-300 my-1 space-y-3">
      {error && <div className="text-red-400">{error}</div>}

      {status.type === 'no_factory' && node.recipe && (
        <NoFactoryPanel
          node={node}
          goal={goal}
          ctx={ctx}
          nodes={nodes}
          starName={starName}
          onSetOverride={onSetOverride}
          onClose={onClose}
          loading={loading}
        />
      )}

      {status.type === 'route_missing' && (
        <RouteMissingPanel
          node={node}
          availableAt={status.available_at}
          ctx={ctx}
          nodes={nodes}
          starName={starName}
          onSetOverride={onSetOverride}
          onCreateRoute={handleCreateRoute}
          onClose={onClose}
          loading={loading}
        />
      )}

      {status.type === 'missing' && node.recipe && (
        <MissingPanel
          node={node}
          goal={goal}
          ctx={ctx}
          nodes={nodes}
          starName={starName}
          onSetOverride={onSetOverride}
          onCreateOrder={handleCreateOrder}
          onClose={onClose}
          loading={loading}
        />
      )}

      {!['no_factory', 'route_missing', 'missing'].includes(status.type) && (
        <div className="text-slate-500">Kein Fix notwendig für Status: {status.type}</div>
      )}
    </div>
  )
}

// ── Sub-panels ────────────────────────────────────────────────────────────────

function NoFactoryPanel({ node, goal, ctx, nodes, starName, onSetOverride, onClose, loading }: {
  node: BOMNode; goal: Goal; ctx: BOMContext
  nodes: Array<{ node_id: string; star_id: string; x: number; y: number; star_type: string }>
  starName: (s: string) => string
  onSetOverride: (itemId: string, starId: string) => void
  onClose: () => void
  loading: boolean
}) {
  // Find stars that have this item in stock
  const availableStars = nodes
    .map(n => n.star_id)
    .filter((starId, i, arr) => arr.indexOf(starId) === i)
    .filter(starId => {
      const stock = ctx.stockMap.get(node.item_id)
      return stock && stock.available >= node.qty
    })

  return (
    <div className="space-y-2">
      <div className="text-slate-400 font-medium">
        Du brauchst {node.recipe ? factoryLabel(node.recipe.factory_type) : '?'} für {itemLabel(node.item_id)}.
      </div>
      <div className="text-slate-500">
        Keine {node.recipe ? factoryLabel(node.recipe.factory_type) : '?'} vorhanden. Bau-Funktion folgt.
      </div>

      {availableStars.length > 0 && (
        <div className="border-t border-slate-700 pt-2">
          <div className="text-slate-400 mb-1">Oder: Transportieren von</div>
          {availableStars.map(starId => (
            <div key={starId} className="flex items-center justify-between">
              <span>{starName(starId)}</span>
              <PrimaryButton onClick={() => { onSetOverride(node.item_id, starId); onClose() }} disabled={loading}>
                ⬡ Transport von hier
              </PrimaryButton>
            </div>
          ))}
        </div>
      )}

      <GhostButton onClick={onClose}>Abbrechen</GhostButton>
    </div>
  )
}

function RouteMissingPanel({ node, availableAt, ctx, nodes, starName, onSetOverride, onCreateRoute, onClose, loading }: {
  node: BOMNode; availableAt: string; ctx: BOMContext
  nodes: Array<{ node_id: string; star_id: string; x: number; y: number; star_type: string }>
  starName: (s: string) => string
  onSetOverride: (itemId: string, starId: string) => void
  onCreateRoute: (fromNodeId: string, capacity: number) => void
  onClose: () => void
  loading: boolean
}) {
  const [capacity, setCapacity] = useState(20)
  const fromNode = nodes.find(n => n.star_id === availableAt || n.node_id === availableAt)

  const stock = ctx.stockMap.get(node.item_id)
  const available = stock?.available ?? 0

  return (
    <div className="space-y-2">
      <div className="text-slate-400 font-medium">
        {itemLabel(node.item_id)}: {available.toFixed(0)} Einheiten @ {starName(availableAt)}
      </div>
      <div className="text-slate-500">
        Wird benötigt @ Ziel-Stern. Keine Route vorhanden.
      </div>

      {fromNode && (
        <div className="flex items-center gap-2">
          <span className="text-slate-400">Route: {starName(availableAt)} → Ziel</span>
          <span className="text-slate-500">Kapazität:</span>
          <input
            type="number"
            value={capacity}
            min={1}
            onChange={e => setCapacity(Number(e.target.value))}
            className="w-16 bg-slate-800 border border-slate-700 rounded px-1 text-slate-300"
          />
          <span className="text-slate-500">/tick</span>
          <PrimaryButton onClick={() => onCreateRoute(fromNode.node_id, capacity)} disabled={loading}>
            Route anlegen
          </PrimaryButton>
        </div>
      )}

      <div className="border-t border-slate-700 pt-2">
        <div className="text-slate-500 mb-1">Oder: Transport-Override setzen</div>
        <PrimaryButton onClick={() => { onSetOverride(node.item_id, availableAt); onClose() }} disabled={loading}>
          ⬡ Transport von {starName(availableAt)} nutzen
        </PrimaryButton>
      </div>

      <GhostButton onClick={onClose}>Abbrechen</GhostButton>
    </div>
  )
}

function MissingPanel({ node, goal, ctx, nodes, starName, onSetOverride, onCreateOrder, onClose, loading }: {
  node: BOMNode; goal: Goal; ctx: BOMContext
  nodes: Array<{ node_id: string; star_id: string; x: number; y: number; star_type: string }>
  starName: (s: string) => string
  onSetOverride: (itemId: string, starId: string) => void
  onCreateOrder: () => void
  onClose: () => void
  loading: boolean
}) {
  // Check if factory exists
  const factoryType = node.recipe?.factory_type
  const hasFactory = factoryType
    ? ctx.facilities.some(f => f.factory_type === factoryType && f.status !== 'destroyed')
    : false

  // Find other stars with this item
  const availableStars = nodes
    .map(n => n.star_id)
    .filter((starId, i, arr) => arr.indexOf(starId) === i && starId !== goal.star_id)
    .filter(() => {
      const stock = ctx.stockMap.get(node.item_id)
      return stock && stock.available >= node.qty
    })

  return (
    <div className="space-y-2">
      <div className="text-slate-400 font-medium">
        {itemLabel(node.item_id)} × {node.qty.toFixed(0)} fehlt
      </div>

      {hasFactory && (
        <div>
          <div className="text-slate-500 mb-1">
            {node.recipe ? factoryLabel(node.recipe.factory_type) : '?'} vorhanden — Auftrag anlegen:
          </div>
          <div className="flex items-center gap-2">
            <span>@ {starName(goal.star_id)}</span>
            <PrimaryButton onClick={onCreateOrder} disabled={loading}>
              Auftrag anlegen
            </PrimaryButton>
          </div>
        </div>
      )}

      {availableStars.length > 0 && (
        <div className="border-t border-slate-700 pt-2">
          <div className="text-slate-400 mb-1">Oder: Transportieren von</div>
          {availableStars.map(starId => (
            <div key={starId} className="flex items-center justify-between">
              <span>{starName(starId)}</span>
              <PrimaryButton onClick={() => { onSetOverride(node.item_id, starId); onClose() }} disabled={loading}>
                ⬡ Transport
              </PrimaryButton>
            </div>
          ))}
        </div>
      )}

      <GhostButton onClick={onClose}>Abbrechen</GhostButton>
    </div>
  )
}
