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

export interface Order {
  id: string
  player_id: string
  star_id: string
  node_id: string
  facility_id: string | null
  order_type: 'batch' | 'continuous'
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
