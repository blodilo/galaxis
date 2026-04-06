import { useState } from 'react'
import type { Route, AggregatedStock } from '../../types/economy2'
import type { MyNodeEntry } from '../../api/economy2'
import { SectionTitle, Card, PrimaryButton, GhostButton, itemLabel } from './ui'
import { createRoute, deleteGoal } from '../../api/economy2'

interface NetzwerkTabProps {
  nodes: MyNodeEntry[]
  routes: Route[]
  stockAll: AggregatedStock[]
  onRefresh: () => void
}

function NodeCard({ node, stockAll, onConnect, isSelected, onSelect }: {
  node: MyNodeEntry
  stockAll: AggregatedStock[]
  onConnect: () => void
  isSelected: boolean
  onSelect: () => void
}) {
  const topItems = stockAll
    .filter(s => s.total > 0)
    .sort((a, b) => b.total - a.total)
    .slice(0, 3)

  return (
    <div
      onClick={onSelect}
      className={`border rounded p-3 cursor-pointer transition-colors w-52 ${
        isSelected
          ? 'border-emerald-600 bg-emerald-900/20'
          : 'border-slate-700 bg-slate-900/50 hover:border-slate-600'
      }`}
    >
      <div className="flex items-center gap-2 mb-1">
        <span className="w-2 h-2 rounded-full bg-emerald-500 flex-shrink-0" />
        <span className="text-sm font-medium text-slate-200">
          {node.star_type} · {node.star_id.slice(0, 8)}
        </span>
      </div>
      <div className="text-xs text-slate-500 mb-1">
        {node.facility_count} Anlagen · {node.level}
      </div>
      {topItems.length > 0 && (
        <div className="text-xs text-slate-400 space-y-0.5">
          {topItems.map(s => (
            <div key={s.item_id} className="flex justify-between">
              <span>{itemLabel(s.item_id)}</span>
              <span className="text-slate-500">{s.total.toFixed(0)}</span>
            </div>
          ))}
        </div>
      )}
      {isSelected && (
        <div className="mt-2">
          <GhostButton onClick={e => { e.stopPropagation(); onConnect() }}>
            Route zu...
          </GhostButton>
        </div>
      )}
    </div>
  )
}

function RouteArrow({ from, to, route }: {
  from: MyNodeEntry
  to: MyNodeEntry
  route: Route
}) {
  return (
    <div className="flex items-center gap-1 text-xs text-slate-400 my-1">
      <span className="text-slate-500">{from.star_id.slice(0, 6)}</span>
      <span className="text-emerald-600">──{route.capacity_per_tick}/tick──►</span>
      <span className="text-slate-500">{to.star_id.slice(0, 6)}</span>
      <span className={`ml-1 text-xs ${route.status === 'active' ? 'text-emerald-400' : 'text-yellow-400'}`}>
        [{route.status}]
      </span>
    </div>
  )
}

function CreateRoutePanel({ fromNode, toNode, onCancel, onCreated }: {
  fromNode: MyNodeEntry
  toNode: MyNodeEntry
  onCancel: () => void
  onCreated: () => void
}) {
  const [capacity, setCapacity] = useState(20)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleCreate() {
    setLoading(true)
    setError(null)
    try {
      await createRoute({
        from_node_id: fromNode.node_id,
        to_node_id: toNode.node_id,
        capacity_per_tick: capacity,
      })
      onCreated()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fehler')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="bg-slate-900 border border-slate-700 rounded p-3 text-xs text-slate-300 space-y-2 mt-2">
      <div className="text-slate-400 font-medium">
        Route: {fromNode.star_type}·{fromNode.star_id.slice(0,6)} → {toNode.star_type}·{toNode.star_id.slice(0,6)}
      </div>
      {error && <div className="text-red-400">{error}</div>}
      <div className="flex items-center gap-2">
        <span className="text-slate-500">Kapazität:</span>
        <input
          type="number"
          value={capacity}
          min={1}
          onChange={e => setCapacity(Number(e.target.value))}
          className="w-16 bg-slate-800 border border-slate-700 rounded px-1 text-slate-300"
        />
        <span className="text-slate-500">/tick</span>
        <PrimaryButton onClick={handleCreate} disabled={loading}>
          Route anlegen
        </PrimaryButton>
        <GhostButton onClick={onCancel}>Abbrechen</GhostButton>
      </div>
    </div>
  )
}

export default function NetzwerkTab({ nodes, routes, stockAll, onRefresh }: NetzwerkTabProps) {
  const [selectedNode, setSelectedNode] = useState<MyNodeEntry | null>(null)
  const [connectingTo, setConnectingTo] = useState<MyNodeEntry | null>(null)

  function handleSelect(node: MyNodeEntry) {
    if (selectedNode?.node_id === node.node_id) {
      setSelectedNode(null)
      setConnectingTo(null)
    } else if (selectedNode && !connectingTo) {
      // Second click → create route between selectedNode and this node
      setConnectingTo(node)
    } else {
      setSelectedNode(node)
      setConnectingTo(null)
    }
  }

  function getNodeRoutes(node: MyNodeEntry) {
    return routes.filter(
      r => r.from_node_id === node.node_id || r.to_node_id === node.node_id
    )
  }

  return (
    <div className="p-4">
      <SectionTitle>Netzwerk</SectionTitle>

      {nodes.length === 0 && (
        <div className="text-slate-500 text-sm">Keine Nodes vorhanden.</div>
      )}

      <div className="flex flex-wrap gap-4 mb-6">
        {nodes.map(node => (
          <NodeCard
            key={node.node_id}
            node={node}
            stockAll={stockAll}
            isSelected={selectedNode?.node_id === node.node_id}
            onSelect={() => handleSelect(node)}
            onConnect={() => {
              // After "Route zu..." click, next node click = target
              // This is handled by handleSelect's second-click logic
            }}
          />
        ))}
      </div>

      {selectedNode && connectingTo && (
        <CreateRoutePanel
          fromNode={selectedNode}
          toNode={connectingTo}
          onCancel={() => { setSelectedNode(null); setConnectingTo(null) }}
          onCreated={() => { setSelectedNode(null); setConnectingTo(null); onRefresh() }}
        />
      )}

      {routes.length > 0 && (
        <div>
          <SectionTitle>Aktive Routen</SectionTitle>
          <Card>
            {routes.map(route => {
              const from = nodes.find(n => n.node_id === route.from_node_id)
              const to = nodes.find(n => n.node_id === route.to_node_id)
              if (!from || !to) return null
              return (
                <RouteArrow key={route.id} from={from} to={to} route={route} />
              )
            })}
          </Card>
        </div>
      )}

      {selectedNode && !connectingTo && (
        <div className="text-xs text-slate-500 mt-2">
          Klicke auf einen anderen Node um eine Route anzulegen.
        </div>
      )}
    </div>
  )
}
