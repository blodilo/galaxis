import type { ItemStock, Facility, Order, Route, Recipe, DepositEntry } from '../types/economy2'

const BASE = '/api/v2/econ2'
const PLAYER_ID = '00000000-0000-0000-0000-000000000001'

const HEADERS: HeadersInit = {
  'Content-Type': 'application/json',
  'X-Player-ID': PLAYER_ID,
}

async function get<T>(url: string): Promise<T> {
  const res = await fetch(url, { headers: HEADERS })
  if (!res.ok) throw new Error(`API error ${res.status}: ${url}`)
  return res.json()
}

async function post<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: 'POST',
    headers: HEADERS,
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`API error ${res.status}: ${url}`)
  return res.json()
}

async function del(url: string): Promise<void> {
  const res = await fetch(url, { method: 'DELETE', headers: HEADERS })
  if (!res.ok) throw new Error(`API error ${res.status}: ${url}`)
}

// Nodes
export async function createNode(starId: string, planetId?: string): Promise<{ node_id: string }> {
  return post<{ node_id: string }>(`${BASE}/nodes`, { star_id: starId, planet_id: planetId ?? null })
}

// Stock
export async function getStock(nodeId: string): Promise<ItemStock[]> {
  const data = await get<{ stock: Record<string, { total: number; allocated: number }> }>(`${BASE}/stock?node_id=${nodeId}`)
  return Object.entries(data.stock ?? {}).map(([item_id, s]) => ({
    item_id,
    total: s.total,
    allocated: s.allocated,
    available: s.total - s.allocated,
  }))
}

// Facilities
export async function createFacility(data: {
  star_id: string
  planet_id?: string
  factory_type: string
  level?: number
  deposit_good_id?: string
}): Promise<Facility> {
  return post<Facility>(`${BASE}/facilities`, data)
}

export async function listFacilities(starId: string): Promise<Facility[]> {
  const data = await get<{ facilities: Facility[] } | Facility[]>(`${BASE}/facilities?star_id=${starId}`)
  return Array.isArray(data) ? data : (data as { facilities: Facility[] }).facilities ?? []
}

export async function destroyFacility(id: string): Promise<void> {
  return del(`${BASE}/facilities/${id}`)
}

// Orders
export async function createOrder(data: {
  node_id: string
  star_id: string
  factory_type: string
  product_id: string
  order_type: 'batch' | 'continuous' | 'build'
  target_qty: number
  priority?: number
}): Promise<Order> {
  return post<Order>(`${BASE}/orders`, data)
}

// Recipes
export async function listRecipes(): Promise<Recipe[]> {
  const data = await get<{ recipes: Recipe[] }>(`${BASE}/recipes`)
  return data.recipes ?? []
}

export async function listOrders(nodeId: string): Promise<Order[]> {
  const data = await get<{ orders: Order[] } | Order[]>(`${BASE}/orders?node_id=${nodeId}`)
  return Array.isArray(data) ? data : (data as { orders: Order[] }).orders ?? []
}

export async function cancelOrder(id: string): Promise<void> {
  return del(`${BASE}/orders/${id}`)
}

// Routes
export async function createRoute(data: {
  from_node_id: string
  to_node_id: string
  capacity_per_tick: number
  min_continuous_share?: number
}): Promise<Route> {
  return post<Route>(`${BASE}/routes`, data)
}

export async function listRoutes(): Promise<Route[]> {
  const data = await get<{ routes: Route[] } | Route[]>(`${BASE}/routes`)
  return Array.isArray(data) ? data : (data as { routes: Route[] }).routes ?? []
}

// Bootstrap
export interface BootstrapResult {
  node_id: string
  seeded_stock: Record<string, number>
  seeded_facilities: number
}

export async function bootstrap(starId: string): Promise<BootstrapResult> {
  return post<BootstrapResult>(`${BASE}/bootstrap`, { star_id: starId })
}

// Deposits
export async function getDeposits(starId: string): Promise<{ planet_id: string; deposits: Record<string, DepositEntry>; mine_rate_lv1: number }> {
  return get<{ planet_id: string; deposits: Record<string, DepositEntry>; mine_rate_lv1: number }>(`${BASE}/deposits?star_id=${starId}`)
}

// My nodes
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

export async function listMyNodes(): Promise<MyNodeEntry[]> {
  const data = await get<{ nodes: MyNodeEntry[] }>(`${BASE}/my-nodes`)
  return data.nodes ?? []
}
