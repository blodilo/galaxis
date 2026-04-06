import { useState } from 'react'
import type { Goal, Recipe, AggregatedStock, Order, Route } from '../../types/economy2'
import type { Facility } from '../../types/economy2'
import type { MyNodeEntry } from '../../api/economy2'
import { SectionTitle, Card, PrimaryButton, itemLabel } from './ui'
import BOMTree from './BOMTree'
import { createGoal, reorderGoals, deleteGoal } from '../../api/economy2'

interface PlanTabProps {
  goals: Goal[]
  recipes: Recipe[]
  stockAll: AggregatedStock[]
  facilities: Facility[]
  orders: Order[]
  routes: Route[]
  nodes: MyNodeEntry[]
  onRefresh: () => void
}

function GoalPicker({ recipes, nodes, onRefresh }: {
  recipes: Recipe[]
  nodes: MyNodeEntry[]
  onRefresh: () => void
}) {
  const [productId, setProductId] = useState('')
  const [qty, setQty] = useState(1)
  const [starId, setStarId] = useState(nodes[0]?.star_id ?? '')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Buildable products = non-extractor, non-construction recipes
  const buildableProducts = [...new Map(
    recipes
      .filter(r => r.factory_type !== 'extractor' && r.factory_type !== 'construction')
      .map(r => [r.product_id, r])
  ).values()]

  async function handleCreate() {
    if (!productId || !starId) return
    setLoading(true)
    setError(null)
    try {
      await createGoal({ star_id: starId, product_id: productId, target_qty: qty })
      onRefresh()
      setProductId('')
      setQty(1)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fehler')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card>
      <div className="text-sm text-slate-400 mb-3">Was willst du bauen?</div>
      <div className="flex items-center gap-2 flex-wrap">
        <select
          value={productId}
          onChange={e => setProductId(e.target.value)}
          className="bg-slate-800 border border-slate-700 rounded px-2 py-1 text-sm text-slate-300 flex-1 min-w-0"
        >
          <option value="">— Produkt wählen —</option>
          {buildableProducts.map(r => (
            <option key={r.product_id} value={r.product_id}>
              {itemLabel(r.product_id)}
            </option>
          ))}
        </select>

        <div className="flex items-center gap-1">
          <span className="text-sm text-slate-500">Menge:</span>
          <input
            type="number"
            value={qty}
            min={1}
            onChange={e => setQty(Math.max(1, Number(e.target.value)))}
            className="w-16 bg-slate-800 border border-slate-700 rounded px-1 py-1 text-sm text-slate-300"
          />
        </div>

        {nodes.length > 1 && (() => {
          const uniqueStars = [...new Map(nodes.map(n => [n.star_id, n])).values()]
          return uniqueStars.length > 1 ? (
            <select
              value={starId}
              onChange={e => setStarId(e.target.value)}
              className="bg-slate-800 border border-slate-700 rounded px-2 py-1 text-sm text-slate-300"
            >
              {uniqueStars.map(n => (
                <option key={n.star_id} value={n.star_id}>
                  {n.star_type} · {n.star_id.slice(0, 8)}
                </option>
              ))}
            </select>
          ) : null
        })()}

        <PrimaryButton onClick={handleCreate} disabled={loading || !productId}>
          Ziel anlegen
        </PrimaryButton>
      </div>
      {error && <div className="text-red-400 text-xs mt-2">{error}</div>}
    </Card>
  )
}

function GoalCard({
  goal,
  recipes,
  stockAll,
  facilities,
  orders,
  routes,
  nodes,
  onRefresh,
  onDelete,
  onSetOverride,
  onClearOverride,
}: {
  goal: Goal
  recipes: Recipe[]
  stockAll: AggregatedStock[]
  facilities: Facility[]
  orders: Order[]
  routes: Route[]
  nodes: MyNodeEntry[]
  onRefresh: () => void
  onDelete: (id: string) => void
  onSetOverride: (goalId: string, itemId: string, starId: string) => void
  onClearOverride: (goalId: string, itemId: string) => void
}) {
  const [expanded, setExpanded] = useState(true)

  return (
    <Card>
      <div className="flex items-center gap-2 mb-2">
        <button onClick={() => setExpanded(e => !e)} className="text-slate-500 text-xs w-4">
          {expanded ? '▼' : '▶'}
        </button>
        <span className="text-sm text-slate-200 font-medium flex-1">
          {itemLabel(goal.product_id)} × {goal.target_qty}
        </span>
        <span className="text-xs text-slate-500">Prio {goal.priority}</span>
        <button
          onClick={() => onDelete(goal.id)}
          className="text-xs text-red-500 hover:text-red-400 px-1"
          title="Ziel abbrechen"
        >
          ✕
        </button>
      </div>

      {expanded && (
        <BOMTree
          goal={goal}
          recipes={recipes}
          stockAll={stockAll}
          facilities={facilities}
          orders={orders}
          routes={routes}
          nodes={nodes}
          onSetOverride={(itemId, starId) => onSetOverride(goal.id, itemId, starId)}
          onClearOverride={(itemId) => onClearOverride(goal.id, itemId)}
          onRefresh={onRefresh}
        />
      )}
    </Card>
  )
}

export default function PlanTab({
  goals,
  recipes,
  stockAll,
  facilities,
  orders,
  routes,
  nodes,
  onRefresh,
}: PlanTabProps) {
  const [localGoals, setLocalGoals] = useState<Goal[]>(goals)
  // Sync when parent refreshes
  if (JSON.stringify(goals.map(g => g.id)) !== JSON.stringify(localGoals.map(g => g.id))) {
    setLocalGoals(goals)
  }

  async function handleDelete(id: string) {
    try {
      await deleteGoal(id)
      onRefresh()
    } catch (e) {
      console.error(e)
    }
  }

  async function handleSetOverride(goalId: string, itemId: string, starId: string) {
    // Optimistic update — persist via PATCH if backend supports transport_overrides update
    // For now: just refresh to reflect state
    onRefresh()
    console.info('transport override set', goalId, itemId, starId)
  }

  async function handleClearOverride(goalId: string, itemId: string) {
    onRefresh()
    console.info('transport override cleared', goalId, itemId)
  }

  return (
    <div className="p-4 space-y-4">
      <SectionTitle>Produktionsziele</SectionTitle>

      {nodes.length > 0 && (
        <GoalPicker recipes={recipes} nodes={nodes} onRefresh={onRefresh} />
      )}

      {goals.length === 0 && (
        <div className="text-slate-500 text-sm py-4 text-center">
          Noch keine Ziele. Leg ein Ziel an um die Produktionskette aufzulösen.
        </div>
      )}

      {goals.map(goal => (
        <GoalCard
          key={goal.id}
          goal={goal}
          recipes={recipes}
          stockAll={stockAll}
          facilities={facilities}
          orders={orders}
          routes={routes}
          nodes={nodes}
          onRefresh={onRefresh}
          onDelete={handleDelete}
          onSetOverride={handleSetOverride}
          onClearOverride={handleClearOverride}
        />
      ))}
    </div>
  )
}
