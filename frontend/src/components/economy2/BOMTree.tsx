import { useState, useMemo } from 'react'
import type { BOMNode, BOMStatus, Recipe, AggregatedStock, Order, Route, Goal } from '../../types/economy2'
import type { Facility } from '../../types/economy2'
import { itemLabel, factoryLabel, StatusBadge, GhostButton } from './ui'
import FixPanel from './FixPanel'

// ── BOM computation ───────────────────────────────────────────────────────────

export interface BOMContext {
  recipes: Recipe[]
  stockMap: Map<string, AggregatedStock>       // item_id → stock
  stockByNode: Map<string, Map<string, AggregatedStock>> // node_id → item_id → stock
  facilities: Facility[]
  orders: Order[]
  routes: Route[]
  nodeToStarMap: Map<string, string>           // node_id → star_id
  transportOverrides: Record<string, string>  // item_id → star_id
}

function findRecipe(recipes: Recipe[], itemId: string): Recipe | null {
  return recipes.find(r => r.product_id === itemId && r.factory_type !== 'construction') ?? null
}

function hasFactory(facilities: Facility[], factoryType: string): boolean {
  return facilities.some(f => f.factory_type === factoryType && f.status !== 'destroyed')
}

function hasActiveOrder(orders: Order[], itemId: string): Order | null {
  return orders.find(o =>
    o.product_id === itemId &&
    !['completed', 'cancelled'].includes(o.status)
  ) ?? null
}

function findItemAtOtherNode(
  stockByNode: Map<string, Map<string, AggregatedStock>>,
  nodeToStarMap: Map<string, string>,
  itemId: string,
  needed: number,
): string | null {
  for (const [nodeId, nodeStock] of stockByNode) {
    const s = nodeStock.get(itemId)
    if (s && s.available >= needed) {
      return nodeToStarMap.get(nodeId) ?? nodeId
    }
  }
  return null
}

function hasRoute(routes: Route[], fromStarId: string, toNodeIds: string[]): boolean {
  for (const r of routes) {
    if (toNodeIds.includes(r.to_node_id) && r.status === 'active') {
      // Check if from_node matches the star
      if (r.from_node_id === fromStarId) return true
    }
  }
  return false
}

function computeStatus(
  itemId: string,
  qty: number,
  recipe: Recipe | null,
  ctx: BOMContext,
): BOMStatus {
  const override = ctx.transportOverrides[itemId]
  if (override) {
    return { type: 'transport_override', source_star_id: override }
  }

  const stock = ctx.stockMap.get(itemId)
  if (stock && stock.available >= qty) {
    return { type: 'ok', qty: stock.available, node_id: '' }
  }

  if (recipe === null) {
    // Raw material — check if available elsewhere
    const otherNodeId = findItemAtOtherNode(ctx.stockByNode, ctx.nodeToStarMap, itemId, qty)
    if (otherNodeId) {
      return { type: 'route_missing', available_at: otherNodeId }
    }
    return { type: 'missing' }
  }

  const activeOrder = hasActiveOrder(ctx.orders, itemId)
  if (activeOrder) {
    if (activeOrder.status === 'waiting') return { type: 'waiting' }
    return { type: 'running' }
  }

  const otherNodeId = findItemAtOtherNode(ctx.stockByNode, ctx.nodeToStarMap, itemId, qty)
  if (otherNodeId) {
    const toNodeIds = [...ctx.nodeToStarMap.keys()]
    if (!hasRoute(ctx.routes, otherNodeId, toNodeIds)) {
      return { type: 'route_missing', available_at: otherNodeId }
    }
    return { type: 'in_transit' }
  }

  if (!hasFactory(ctx.facilities, recipe.factory_type)) {
    return { type: 'no_factory' }
  }

  return { type: 'missing' }
}

export function buildBOMNode(
  itemId: string,
  qty: number,
  ctx: BOMContext,
  visiting: Set<string>,
): BOMNode {
  const recipe = findRecipe(ctx.recipes, itemId)
  const status = computeStatus(itemId, qty, recipe, ctx)

  // If transport override or raw material or status is ok — no children needed
  if (
    status.type === 'transport_override' ||
    recipe === null ||
    status.type === 'ok'
  ) {
    return {
      item_id: itemId,
      qty,
      recipe,
      factory_type: recipe?.factory_type ?? null,
      status,
      children: [],
      transport_override: ctx.transportOverrides[itemId],
    }
  }

  if (visiting.has(itemId)) {
    // Cycle — return as-is without children
    return { item_id: itemId, qty, recipe, factory_type: recipe.factory_type, status, children: [] }
  }

  visiting.add(itemId)
  const runs = qty / recipe.base_yield
  const children: BOMNode[] = recipe.inputs.map(inp =>
    buildBOMNode(inp.item_id, inp.amount * runs, ctx, visiting)
  )
  visiting.delete(itemId)

  return {
    item_id: itemId,
    qty,
    recipe,
    factory_type: recipe.factory_type,
    status,
    children,
    transport_override: ctx.transportOverrides[itemId],
  }
}

// ── Status pill ───────────────────────────────────────────────────────────────

function StatusPill({ status, onFix }: { status: BOMStatus; onFix: () => void }) {
  switch (status.type) {
    case 'ok':
      return <StatusBadge label={`✓ ${status.qty.toFixed(0)}`} color="green" />
    case 'running':
      return <StatusBadge label="● läuft" color="green" />
    case 'waiting':
      return <StatusBadge label="⚠ wartet" color="yellow" />
    case 'no_factory':
      return (
        <span className="flex items-center gap-1">
          <StatusBadge label="✗ keine Fabrik" color="red" />
          <GhostButton onClick={onFix}>Fix →</GhostButton>
        </span>
      )
    case 'route_missing':
      return (
        <span className="flex items-center gap-1">
          <StatusBadge label="→ Route fehlt" color="yellow" />
          <GhostButton onClick={onFix}>Fix →</GhostButton>
        </span>
      )
    case 'in_transit':
      return <StatusBadge label="→ unterwegs" color="cyan" />
    case 'transport_override':
      return <StatusBadge label="⬡ Transport" color="cyan" />
    case 'missing':
      return (
        <span className="flex items-center gap-1">
          <StatusBadge label="✗ fehlt" color="red" />
          <GhostButton onClick={onFix}>Fix →</GhostButton>
        </span>
      )
  }
}

// ── BOM Row ───────────────────────────────────────────────────────────────────

function hasBlockedDescendant(node: BOMNode): boolean {
  if (['no_factory', 'route_missing', 'missing', 'waiting'].includes(node.status.type)) return true
  return node.children.some(hasBlockedDescendant)
}

function BOMRow({
  node,
  depth,
  ctx,
  goal,
  onSetOverride,
  onClearOverride,
  onRefresh,
  nodes,
}: {
  node: BOMNode
  depth: number
  ctx: BOMContext
  goal: Goal
  onSetOverride: (itemId: string, starId: string) => void
  onClearOverride: (itemId: string) => void
  onRefresh: () => void
  nodes: Array<{ node_id: string; star_id: string; x: number; y: number; star_type: string }>
}) {
  const defaultOpen = hasBlockedDescendant(node) || node.status.type !== 'ok'
  const [open, setOpen] = useState(defaultOpen)
  const [showFix, setShowFix] = useState(false)

  const hasChildren = node.children.length > 0 && node.status.type !== 'transport_override'
  const indent = depth * 16

  return (
    <div>
      <div
        className="flex items-center gap-2 py-1 text-sm hover:bg-slate-800/30 rounded px-1"
        style={{ paddingLeft: indent + 4 }}
      >
        {hasChildren ? (
          <button
            onClick={() => setOpen(o => !o)}
            className="text-slate-500 w-4 text-xs flex-shrink-0"
          >
            {open ? '▼' : '▶'}
          </button>
        ) : (
          <span className="w-4 flex-shrink-0" />
        )}

        <span className="text-slate-300 flex-1">
          {itemLabel(node.item_id)}
          <span className="text-slate-600 text-xs ml-1">×{node.qty.toFixed(0)}</span>
          {node.factory_type && (
            <span className="text-slate-600 text-xs ml-1">
              [{factoryLabel(node.factory_type)}]
            </span>
          )}
        </span>

        {node.transport_override ? (
          <span className="flex items-center gap-1">
            <StatusBadge label="⬡ Transport" color="cyan" />
            <GhostButton onClick={() => onClearOverride(node.item_id)}>lokal?</GhostButton>
          </span>
        ) : (
          <StatusPill status={node.status} onFix={() => setShowFix(f => !f)} />
        )}
      </div>

      {showFix && (
        <div style={{ paddingLeft: indent + 24 }} className="mb-1">
          <FixPanel
            node={node}
            goal={goal}
            ctx={ctx}
            nodes={nodes}
            onSetOverride={onSetOverride}
            onClose={() => setShowFix(false)}
            onRefresh={onRefresh}
          />
        </div>
      )}

      {open && hasChildren && node.children.map(child => (
        <BOMRow
          key={child.item_id}
          node={child}
          depth={depth + 1}
          ctx={ctx}
          goal={goal}
          onSetOverride={onSetOverride}
          onClearOverride={onClearOverride}
          onRefresh={onRefresh}
          nodes={nodes}
        />
      ))}
    </div>
  )
}

// ── BOMTree (public component) ────────────────────────────────────────────────

interface BOMTreeProps {
  goal: Goal
  recipes: Recipe[]
  stockAll: AggregatedStock[]
  facilities: Facility[]
  orders: Order[]
  routes: Route[]
  nodes: Array<{ node_id: string; star_id: string; x: number; y: number; star_type: string }>
  onSetOverride: (itemId: string, starId: string) => void
  onClearOverride: (itemId: string) => void
  onRefresh: () => void
}

export default function BOMTree({
  goal,
  recipes,
  stockAll,
  facilities,
  orders,
  routes,
  nodes,
  onSetOverride,
  onClearOverride,
  onRefresh,
}: BOMTreeProps) {
  const ctx = useMemo<BOMContext>(() => {
    const stockMap = new Map(stockAll.map(s => [s.item_id, s]))

    // Build per-node stock map
    const stockByNode = new Map<string, Map<string, AggregatedStock>>()
    // We don't have per-node stock in stockAll — skip for now, use aggregate only
    // TODO: extend when per-node stock-all endpoint is available

    const nodeToStarMap = new Map(nodes.map(n => [n.node_id, n.star_id]))

    return {
      recipes,
      stockMap,
      stockByNode,
      facilities,
      orders,
      routes,
      nodeToStarMap,
      transportOverrides: goal.transport_overrides,
    }
  }, [goal, recipes, stockAll, facilities, orders, routes, nodes])

  const root = useMemo(
    () => buildBOMNode(goal.product_id, goal.target_qty, ctx, new Set()),
    [goal, ctx]
  )

  return (
    <div className="font-mono text-xs">
      <BOMRow
        node={root}
        depth={0}
        ctx={ctx}
        goal={goal}
        onSetOverride={onSetOverride}
        onClearOverride={onClearOverride}
        onRefresh={onRefresh}
        nodes={nodes}
      />
    </div>
  )
}
