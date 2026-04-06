// Inspired by GraphCanvas.tsx from /home/data/projects/graph (MIT License)
// @ts-ignore
import dagreExt from 'cytoscape-dagre'
import cytoscape from 'cytoscape'
import CytoscapeComponent from 'react-cytoscapejs'
import { useRef, useMemo } from 'react'
import type { AggregatedStock, Facility, Order, Recipe } from '../../types/economy2'
import { itemLabel, factoryLabel } from './ui'

try { cytoscape.use(dagreExt) } catch { /* already registered */ }

// ── Stylesheet ────────────────────────────────────────────────────────────────

const STYLESHEET: cytoscape.StylesheetStyle[] = [
  {
    selector: 'node[type = "item"]',
    style: {
      shape: 'roundrectangle',
      width: 120,
      height: 44,
      'background-color': 'data(bgColor)' as unknown as string,
      'border-color': 'data(borderColor)' as unknown as string,
      'border-width': 1.5,
      label: 'data(label)',
      'font-size': 11,
      'font-family': 'JetBrains Mono, ui-monospace, monospace',
      color: '#e2e8f0',
      'text-valign': 'center',
      'text-halign': 'center',
      'text-wrap': 'wrap',
      'text-max-width': '110px' as unknown as number,
    },
  },
  {
    selector: 'node[type = "facility"]',
    style: {
      shape: 'hexagon',
      width: 100,
      height: 90,
      'background-color': 'data(bgColor)' as unknown as string,
      'border-color': 'data(borderColor)' as unknown as string,
      'border-width': 1.5,
      label: 'data(label)',
      'font-size': 10,
      'font-family': 'JetBrains Mono, ui-monospace, monospace',
      color: '#e2e8f0',
      'text-valign': 'center',
      'text-halign': 'center',
      'text-wrap': 'wrap',
      'text-max-width': '88px' as unknown as number,
    },
  },
  {
    selector: 'edge',
    style: {
      'line-color': 'data(color)' as unknown as string,
      'target-arrow-color': 'data(color)' as unknown as string,
      'target-arrow-shape': 'triangle',
      'curve-style': 'bezier',
      width: 1.5,
      label: 'data(label)',
      'font-size': 9,
      'font-family': 'ui-monospace, monospace',
      color: '#64748b',
      'text-background-opacity': 1,
      'text-background-color': '#0f172a',
      'text-background-padding': '2px',
      'text-background-shape': 'roundrectangle',
    },
  },
  {
    selector: 'node:selected',
    style: { 'border-width': 3, 'border-color': '#38bdf8' },
  },
]

// ── Color helpers ─────────────────────────────────────────────────────────────

function itemNodeColors(stock: AggregatedStock | undefined): { bgColor: string; borderColor: string } {
  if (!stock || stock.total === 0) return { bgColor: '#1e293b', borderColor: '#475569' }
  const ratio = stock.available / stock.total
  if (ratio > 0.5) return { bgColor: '#14532d', borderColor: '#16a34a' }
  if (ratio > 0.1) return { bgColor: '#78350f', borderColor: '#d97706' }
  return { bgColor: '#450a0a', borderColor: '#dc2626' }
}

function facilityNodeColors(status: string): { bgColor: string; borderColor: string } {
  if (status === 'running') return { bgColor: '#052e16', borderColor: '#15803d' }
  if (status === 'building') return { bgColor: '#1e3a5f', borderColor: '#2563eb' }
  if (['paused_input', 'paused_depleted'].includes(status)) return { bgColor: '#451a03', borderColor: '#d97706' }
  if (status === 'idle') return { bgColor: '#1e1e2e', borderColor: '#475569' }
  return { bgColor: '#1a1a1a', borderColor: '#334155' }
}

function orderEdgeColor(status: string): string {
  if (['running', 'ready'].includes(status)) return '#22c55e'
  if (status === 'waiting') return '#f59e0b'
  if (['pending'].includes(status)) return '#475569'
  return '#334155'
}

// ── Graph builder ─────────────────────────────────────────────────────────────

interface GraphData {
  nodes: cytoscape.ElementDefinition[]
  edges: cytoscape.ElementDefinition[]
}

function buildGraphElements(
  orders: Order[],
  facilities: Facility[],
  stockAll: AggregatedStock[],
  recipes: Recipe[],
): GraphData {
  const stockMap = new Map(stockAll.map(s => [s.item_id, s]))
  const facilityMap = new Map(facilities.map(f => [f.id, f]))

  const itemNodeIds = new Set<string>()
  const nodes: cytoscape.ElementDefinition[] = []
  const edges: cytoscape.ElementDefinition[] = []

  // One node per facility that has an active order
  const facilitiesWithOrders = new Set(orders.map(o => o.facility_id).filter(Boolean))

  // Also show all non-destroyed facilities
  for (const f of facilities) {
    if (f.status === 'destroyed') continue
    const colors = facilityNodeColors(f.status)
    const label = f.config.deposit_good_id
      ? `${factoryLabel(f.factory_type)}\n(${itemLabel(f.config.deposit_good_id)})`
      : factoryLabel(f.factory_type)
    nodes.push({
      data: {
        id: `fac_${f.id}`,
        type: 'facility',
        label,
        facilityId: f.id,
        status: f.status,
        ...colors,
      },
    })
  }

  // One node per item involved in any order (inputs + outputs)
  function ensureItemNode(itemId: string) {
    if (itemNodeIds.has(itemId)) return
    itemNodeIds.add(itemId)
    const stock = stockMap.get(itemId)
    const colors = itemNodeColors(stock)
    const stockLabel = stock ? `${stock.available.toFixed(0)}/${stock.total.toFixed(0)}` : '—'
    nodes.push({
      data: {
        id: `item_${itemId}`,
        type: 'item',
        itemId,
        label: `${itemLabel(itemId)}\n${stockLabel}`,
        ...colors,
      },
    })
  }

  // Build lookups for matching orders to facilities:
  // - Extractors: match by factory_type + deposit_good_id == product_id
  // - Others: match by factory_type
  const facilityByType = new Map<string, string>()           // factory_type → fac node id
  const extractorByGood = new Map<string, string>()          // deposit_good_id → fac node id
  for (const f of facilities) {
    if (f.status === 'destroyed') continue
    if (f.factory_type === 'extractor' && f.config.deposit_good_id) {
      if (!extractorByGood.has(f.config.deposit_good_id)) {
        extractorByGood.set(f.config.deposit_good_id, `fac_${f.id}`)
      }
    } else {
      if (!facilityByType.has(f.factory_type)) {
        facilityByType.set(f.factory_type, `fac_${f.id}`)
      }
    }
  }

  // Deduplicate: one logical edge per (factory_type, product_id) pair to avoid clutter
  const seenEdges = new Set<string>()

  // Edges: input_item → facility (consumption), facility → output_item (production)
  // Skip extractor orders — those are handled in the dedicated extractor block below.
  for (const order of orders) {
    if (['completed', 'cancelled'].includes(order.status)) continue
    if (order.factory_type === 'extractor') continue

    // Resolve facility node: use assigned facility, or match by factory_type
    let facNode: string | null = null
    if (order.facility_id) {
      facNode = `fac_${order.facility_id}`
    } else {
      facNode = facilityByType.get(order.factory_type) ?? null
    }
    const outputItemId = order.product_id
    ensureItemNode(outputItemId)

    // facility → output item (deduplicated)
    const outKey = `${facNode}_${outputItemId}`
    if (facNode && !seenEdges.has(outKey)) {
      seenEdges.add(outKey)
      edges.push({
        data: {
          id: `edge_out_${order.id}`,
          source: facNode,
          target: `item_${outputItemId}`,
          label: `${order.produced_qty.toFixed(0)}/${order.target_qty.toFixed(0)}`,
          color: orderEdgeColor(order.status),
        },
      })
    }

    // input items → facility (deduplicated)
    for (const input of order.inputs ?? []) {
      ensureItemNode(input.item_id)
      const inKey = `${input.item_id}_${facNode}`
      if (facNode && !seenEdges.has(inKey)) {
        seenEdges.add(inKey)
        edges.push({
          data: {
            id: `edge_in_${order.id}_${input.item_id}`,
            source: `item_${input.item_id}`,
            target: facNode,
            label: `×${input.amount}`,
            color: orderEdgeColor(order.status),
          },
        })
      }
    }
  }

  // Extractor edges: every extractor gets its own edge to its output item.
  // This replaces the order-based matching for extractors (which deduplicates incorrectly).
  for (const f of facilities) {
    if (f.status === 'destroyed' || !f.config.deposit_good_id) continue
    const itemId = f.config.deposit_good_id
    ensureItemNode(itemId)
    const edgeId = `edge_extractor_${f.id}`
    if (seenEdges.has(edgeId)) continue
    seenEdges.add(edgeId)

    const order = f.current_order_id
      ? orders.find(o => o.id === f.current_order_id)
      : null
    const label = order ? `${order.produced_qty.toFixed(0)}/${order.target_qty > 0 ? order.target_qty.toFixed(0) : '∞'}` : ''
    const color = f.status === 'running' ? '#22c55e' : '#334155'

    edges.push({
      data: {
        id: edgeId,
        source: `fac_${f.id}`,
        target: `item_${itemId}`,
        label,
        color,
      },
    })
  }

  return { nodes, edges }
}

// ── Component ─────────────────────────────────────────────────────────────────

interface ProductionGraphProps {
  orders: Order[]
  facilities: Facility[]
  stockAll: AggregatedStock[]
  recipes: Recipe[]
  height?: number
}

export default function ProductionGraph({
  orders,
  facilities,
  stockAll,
  recipes,
  height = 500,
}: ProductionGraphProps) {
  const cyRef = useRef<cytoscape.Core | null>(null)

  const elements = useMemo(
    () => {
      const { nodes, edges } = buildGraphElements(orders, facilities, stockAll, recipes)
      return [...nodes, ...edges]
    },
    [orders, facilities, stockAll, recipes],
  )

  const layout = {
    name: 'dagre',
    rankDir: 'LR',       // left-to-right: raw material → facility → product
    nodeSep: 40,
    rankSep: 100,
    padding: 20,
    animate: false,
  }

  if (elements.filter(e => !e.data.source).length === 0) {
    return (
      <div
        className="flex items-center justify-center text-slate-600 text-sm"
        style={{ height }}
      >
        Keine aktiven Aufträge — Graph leer.
      </div>
    )
  }

  return (
    <div style={{ height }} className="bg-slate-950 rounded border border-slate-800 relative">
      <CytoscapeComponent
        elements={elements}
        stylesheet={STYLESHEET}
        layout={layout}
        style={{ width: '100%', height: '100%' }}
        cy={cy => { cyRef.current = cy }}
        className="rounded"
      />
      <div className="absolute bottom-2 right-2 flex gap-3 text-xs text-slate-600 pointer-events-none">
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-green-700" />läuft</span>
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-yellow-700" />wartet</span>
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-slate-700" />pending</span>
        <span className="flex items-center gap-1">⬡ Anlage</span>
        <span className="flex items-center gap-1">▭ Gut</span>
      </div>
    </div>
  )
}
