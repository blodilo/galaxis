import { useRef } from 'react'
import type { Goal, AggregatedStock, BOMNode } from '../../types/economy2'
import { SectionTitle, itemLabel, StatusBadge, PrimaryButton } from './ui'
import { reorderGoals } from '../../api/economy2'

interface LeftRailProps {
  goals: Goal[]
  stockAll: AggregatedStock[]
  activeGoalId: string | null
  onSelectGoal: (id: string) => void
  onRefresh: () => void
}

function GoalStatusPill({ goal }: { goal: Goal }) {
  // Simple heuristic based on goal status
  if (goal.status === 'completed') return <StatusBadge label="✓ fertig" color="green" />
  if (goal.status === 'cancelled') return <StatusBadge label="✗ abgebrochen" color="slate" />
  return <StatusBadge label="● aktiv" color="slate" />
}

function GoalRow({
  goal,
  isActive,
  onSelect,
  onDragStart,
  onDragOver,
  onDrop,
}: {
  goal: Goal
  isActive: boolean
  onSelect: () => void
  onDragStart: (e: React.DragEvent) => void
  onDragOver: (e: React.DragEvent) => void
  onDrop: (e: React.DragEvent) => void
}) {
  return (
    <div
      draggable
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDrop={onDrop}
      onClick={onSelect}
      className={`flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer text-xs transition-colors
        ${isActive ? 'bg-emerald-900/30 border border-emerald-800' : 'hover:bg-slate-800/50 border border-transparent'}`}
    >
      <span className="text-slate-600 cursor-grab">⠿</span>
      <span className="flex-1 text-slate-300 truncate">{itemLabel(goal.product_id)}</span>
      <span className="text-slate-600">×{goal.target_qty}</span>
      <GoalStatusPill goal={goal} />
    </div>
  )
}

export default function LeftRail({ goals, stockAll, activeGoalId, onSelectGoal, onRefresh }: LeftRailProps) {
  const dragId = useRef<string | null>(null)

  function handleDragStart(e: React.DragEvent, goalId: string) {
    dragId.current = goalId
    e.dataTransfer.effectAllowed = 'move'
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
  }

  async function handleDrop(e: React.DragEvent, targetGoalId: string) {
    e.preventDefault()
    if (!dragId.current || dragId.current === targetGoalId) return

    const ids = [...goals.map(g => g.id)]
    const fromIdx = ids.indexOf(dragId.current)
    const toIdx = ids.indexOf(targetGoalId)
    if (fromIdx === -1 || toIdx === -1) return

    ids.splice(fromIdx, 1)
    ids.splice(toIdx, 0, dragId.current)

    try {
      await reorderGoals(ids)
      onRefresh()
    } catch (e) {
      console.error(e)
    }
    dragId.current = null
  }

  // Low stock items (available < 10% of total, or total 0 if active goal needs it)
  const lowStockItems = stockAll.filter(s => s.total > 0 && s.available < s.total * 0.1)

  return (
    <aside className="w-60 flex-shrink-0 bg-slate-950 border-r border-slate-800 flex flex-col overflow-hidden">
      {/* Goals */}
      <div className="p-3 border-b border-slate-800">
        <SectionTitle>Ziele</SectionTitle>
        {goals.length === 0 ? (
          <div className="text-xs text-slate-600 py-1">Noch keine Ziele</div>
        ) : (
          <div className="space-y-0.5">
            {goals.map(goal => (
              <GoalRow
                key={goal.id}
                goal={goal}
                isActive={goal.id === activeGoalId}
                onSelect={() => onSelectGoal(goal.id)}
                onDragStart={e => handleDragStart(e, goal.id)}
                onDragOver={handleDragOver}
                onDrop={e => handleDrop(e, goal.id)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Alerts */}
      {lowStockItems.length > 0 && (
        <div className="p-3 border-b border-slate-800">
          <SectionTitle>Lager-Alerts</SectionTitle>
          <div className="space-y-0.5">
            {lowStockItems.slice(0, 5).map(s => (
              <div key={s.item_id} className="flex items-center justify-between text-xs">
                <span className="text-slate-400">{itemLabel(s.item_id)}</span>
                <span className="text-yellow-400">{s.available.toFixed(0)} / {s.total.toFixed(0)}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Stock summary */}
      <div className="p-3 flex-1 overflow-auto">
        <SectionTitle>Lager gesamt</SectionTitle>
        {stockAll.length === 0 ? (
          <div className="text-xs text-slate-600">Kein Lagerbestand</div>
        ) : (
          <div className="space-y-0.5">
            {stockAll
              .filter(s => s.total > 0)
              .sort((a, b) => b.total - a.total)
              .map(s => (
                <div key={s.item_id} className="flex items-center justify-between text-xs">
                  <span className="text-slate-400 truncate flex-1">{itemLabel(s.item_id)}</span>
                  <span className={`ml-2 flex-shrink-0 ${
                    s.available < s.total * 0.1 ? 'text-yellow-400'
                    : s.available > s.total * 0.5 ? 'text-emerald-400'
                    : 'text-slate-300'
                  }`}>
                    {s.available.toFixed(0)}
                  </span>
                </div>
              ))}
          </div>
        )}
      </div>
    </aside>
  )
}
