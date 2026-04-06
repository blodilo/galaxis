export interface Node {
  id: string
  player_id: string
  star_id: string
  planet_id: string | null
  level: 'planetary' | 'orbital' | 'intersystem'
}

export interface ItemStock {
  item_id: string
  total: number
  allocated: number
  available: number  // total - allocated
}

export interface FacilityConfig {
  level: number
  ticks_remaining: number
  efficiency_acc: number
  deposit_good_id?: string
}

export interface Facility {
  id: string
  player_id: string
  star_id: string
  planet_id: string | null
  node_id: string
  factory_type: string
  status: 'idle' | 'running' | 'building' | 'paused_input' | 'paused_depleted' | 'destroyed'
  config: FacilityConfig
  current_order_id: string | null
}

export interface RecipeInput {
  item_id: string
  amount: number
}

export interface Recipe {
  recipe_id: string
  product_id: string
  factory_type: string
  inputs: RecipeInput[]
  base_yield: number
  ticks: number
  efficiency: number
  geological_input?: string
}

export interface Order {
  id: string
  player_id: string
  star_id: string
  node_id: string
  facility_id: string | null
  order_type: 'batch' | 'continuous' | 'build'
  status: 'pending' | 'waiting' | 'ready' | 'running' | 'completed' | 'cancelled' | 'paused_depleted'
  recipe_id: string
  product_id: string
  factory_type: string
  inputs: RecipeInput[]
  base_yield: number
  recipe_ticks: number
  target_qty: number
  allocated_inputs: Record<string, number>
  produced_qty: number
  priority: number
}

export interface Route {
  id: string
  player_id: string
  from_node_id: string
  to_node_id: string
  capacity_per_tick: number
  min_continuous_share: number
  status: 'active' | 'suspended'
}

export interface DepositEntry {
  remaining: number
  max_rate: number
}

export interface MyNodeEntry {
  node_id: string
  star_id: string
  planet_id: string | null
  level: string
  star_type: string
  x: number
  y: number
  facility_count: number
}

export interface Goal {
  id: string
  player_id: string
  star_id: string
  product_id: string
  target_qty: number
  priority: number
  status: 'active' | 'completed' | 'cancelled'
  transport_overrides: Record<string, string> // item_id → star_id
  created_at: string
}

export interface AggregatedStock {
  item_id: string
  total: number
  allocated: number
  available: number
}

export type BOMStatus =
  | { type: 'ok'; qty: number; node_id: string }
  | { type: 'running' }
  | { type: 'waiting' }
  | { type: 'no_factory' }
  | { type: 'route_missing'; available_at: string } // node_id where item exists
  | { type: 'in_transit' }
  | { type: 'transport_override'; source_star_id: string }
  | { type: 'missing' }

export interface BOMNode {
  item_id: string
  qty: number
  recipe: Recipe | null      // null = raw material
  factory_type: string | null
  status: BOMStatus
  children: BOMNode[]
  transport_override?: string // star_id if active
}
