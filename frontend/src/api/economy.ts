const BASE = '/api/v1/economy'

// ── Types ────────────────────────────────────────────────────────────────────

export interface FacilityConfig {
  level: number
  recipe_id?: string
  ticks_remaining: number
  efficiency_acc: number
  deposit_id?: string
}

export interface Facility {
  id: string
  facility_type: string
  planet_id?: string
  status: string
  config: FacilityConfig
}

export interface ResourceSnapshot {
  present: boolean
  remaining_approx?: string
  remaining_exact?: number
  max_rate?: number
  slots?: number
}

export interface Survey {
  player_id: string
  planet_id: string
  surveyed_at: string
  tick_n: number
  quality: number
  snapshot: Record<string, ResourceSnapshot>
  stale: boolean
}

export interface SystemState {
  star_id: string
  last_tick_n: number
  storage: Record<string, number>
  facilities: Facility[]
  orbital_slots_used: number
  orbital_slots_max: number
  surveys: Survey[]
}

export interface LogEvent {
  type: string
  facility_id: string
  good?: string
  qty?: number
  missing?: string
  acc_before?: number
  acc_after?: number
}

export interface LogRow {
  ID: string
  TickN: number
  Events: LogEvent[]
  CreatedAt: string
}

export interface TickEvent {
  tick: number
  star_id: string
  message?: string
}

// ── Fetch helpers ────────────────────────────────────────────────────────────

async function get<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) throw new Error(`API ${res.status}: ${url}`)
  return res.json()
}

async function post<T>(url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) throw new Error(`API ${res.status}: ${url}`)
  return res.json()
}

// ── API calls ────────────────────────────────────────────────────────────────

export async function fetchSystemState(starId: string): Promise<SystemState> {
  return get<SystemState>(`${BASE}/system/${starId}`)
}

export async function buildFacility(
  starId: string,
  facilityType: string,
  planetId: string | null,
  level: number,
  depositId?: string,
): Promise<{ id: string }> {
  return post(`${BASE}/system/${starId}/build`, {
    facility_type: facilityType,
    planet_id: planetId,
    level,
    deposit_id: depositId,
  })
}

export async function assignRecipe(
  starId: string,
  facilityId: string,
  recipeId: string,
): Promise<{ status: string }> {
  return post(`${BASE}/system/${starId}/facilities/${facilityId}/recipe`, {
    recipe_id: recipeId,
  })
}

export async function fetchLog(starId: string, limit = 20): Promise<LogRow[]> {
  const data = await get<LogRow[] | null>(`${BASE}/system/${starId}/log?limit=${limit}`)
  return data ?? []
}

export async function executeSurvey(
  planetId: string,
  quality: number,
): Promise<Survey> {
  return post(`${BASE}/planets/${planetId}/survey`, { quality })
}

export async function advanceTick(): Promise<{ status: string }> {
  return post('/api/v1/admin/tick/advance')
}

// ── SSE stream ───────────────────────────────────────────────────────────────

/** Opens an SSE connection for tick events in a system.
 *  Returns a cleanup function; call it on unmount.
 */
export function openTickStream(
  starId: string,
  onEvent: (ev: TickEvent) => void,
  onError?: () => void,
): () => void {
  const es = new EventSource(`${BASE}/system/${starId}/events`)

  es.onmessage = (e) => {
    try {
      onEvent(JSON.parse(e.data) as TickEvent)
    } catch {
      // ignore parse errors
    }
  }

  es.onerror = () => {
    onError?.()
    es.close()
  }

  return () => es.close()
}
