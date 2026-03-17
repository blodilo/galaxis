import type { MorphologyTemplate, GameParams, GenerateRequest, GenerateJob } from '../types/generator'

const BASE = '/api/v1'

export async function fetchMorphologies(): Promise<MorphologyTemplate[]> {
  const res = await fetch(`${BASE}/catalog/morphologies`)
  if (!res.ok) throw new Error(`API ${res.status}: /catalog/morphologies`)
  const data = await res.json()
  return data.morphologies ?? []
}

export async function fetchDefaultParams(): Promise<GameParams> {
  const res = await fetch(`${BASE}/params/defaults`)
  if (!res.ok) throw new Error(`API ${res.status}: /params/defaults`)
  return res.json()
}

export async function postGenerate(req: GenerateRequest): Promise<GenerateJob> {
  const res = await fetch(`${BASE}/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
  const body = await res.json().catch(() => ({ error: `HTTP ${res.status}` }))
  if (!res.ok) throw new Error(body.error ?? `API ${res.status}`)
  return body
}

export async function fetchJobStatus(jobID: string): Promise<GenerateJob> {
  const res = await fetch(`${BASE}/generate/${jobID}/status`)
  if (!res.ok) throw new Error(`API ${res.status}: job ${jobID}`)
  return res.json()
}

export async function postGenerateStep1(req: GenerateRequest): Promise<GenerateJob> {
  const res = await fetch(`${BASE}/generate/step1`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
  const body = await res.json().catch(() => ({ error: `HTTP ${res.status}` }))
  if (!res.ok) throw new Error(body.error ?? `API ${res.status}`)
  return body
}

export async function postGalaxyStep(
  galaxyId: string,
  step: 'spectral' | 'objects' | 'planets'
): Promise<GenerateJob> {
  const res = await fetch(`${BASE}/galaxy/${galaxyId}/steps/${step}`, {
    method: 'POST',
  })
  const body = await res.json().catch(() => ({ error: `HTTP ${res.status}` }))
  if (!res.ok) throw new Error(body.error ?? `API ${res.status}`)
  return body
}

export async function deleteGalaxy(galaxyId: string): Promise<void> {
  const res = await fetch(`${BASE}/galaxy/${galaxyId}`, { method: 'DELETE' })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error((body as any).error ?? `HTTP ${res.status}`)
  }
}

export interface ProgressEvent {
  seq: number
  step: string
  done: number
  total: number
  msg?: string
}

/** Opens an SSE stream for a job's progress. Returns a cleanup function. */
export function openProgressStream(
  jobID: string,
  onEvent: (ev: ProgressEvent) => void,
  onDone: () => void,
): () => void {
  const es = new EventSource(`${BASE}/generate/${jobID}/progress`)
  es.onmessage = (e) => {
    try { onEvent(JSON.parse(e.data) as ProgressEvent) } catch { /* ignore */ }
  }
  es.addEventListener('done', () => { es.close(); onDone() })
  // Do NOT close on onerror — let EventSource auto-reconnect with Last-Event-ID.
  // The backend replays missed events on reconnect via the replay buffer.
  return () => es.close()
}
