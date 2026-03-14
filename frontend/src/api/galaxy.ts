import type { Galaxy, Star, Nebula } from '../types/galaxy'

const BASE = '/api/v1'

async function get<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) throw new Error(`API error ${res.status}: ${url}`)
  return res.json()
}

export async function fetchGalaxies(): Promise<Galaxy[]> {
  const data = await get<{ galaxies: Galaxy[] }>(`${BASE}/galaxies`)
  return data.galaxies ?? []
}

/** Fetches all stars for a galaxy in parallel pages of 10 000. */
export async function fetchAllStars(galaxyID: string): Promise<Star[]> {
  const pageSize = 10_000

  // First page — also tells us total via subsequent pages
  const first = await get<{ stars: Star[]; limit: number; offset: number }>(
    `${BASE}/galaxy/${galaxyID}/stars?limit=${pageSize}&offset=0`,
  )
  const stars = first.stars ?? []

  // If full page returned, keep fetching until we get a partial page
  if (stars.length === pageSize) {
    let offset = pageSize
    while (true) {
      const page = await get<{ stars: Star[] }>(
        `${BASE}/galaxy/${galaxyID}/stars?limit=${pageSize}&offset=${offset}`,
      )
      const batch = page.stars ?? []
      stars.push(...batch)
      if (batch.length < pageSize) break
      offset += pageSize
    }
  }

  return stars
}

export async function fetchStar(galaxyID: string, starID: string): Promise<Star> {
  return get<Star>(`${BASE}/galaxy/${galaxyID}/stars/${starID}`)
}

export async function fetchNebulae(galaxyID: string): Promise<Nebula[]> {
  const data = await get<{ nebulae: Nebula[] }>(`${BASE}/galaxy/${galaxyID}/nebulae`)
  return data.nebulae ?? []
}
