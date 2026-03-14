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
