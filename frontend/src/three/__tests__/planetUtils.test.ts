import { describe, it, expect } from 'vitest'
import { calcPlanetVisR, computeOrbitPos, makeLcg } from '../SystemScene'

// ── calcPlanetVisR ────────────────────────────────────────────────────────────

describe('calcPlanetVisR', () => {
  const MIN = 0.02
  const MAX = 0.4
  const MAX_R = 1.0

  it('gibt 0 zurück wenn radiusEarth ≤ 0', () => {
    expect(calcPlanetVisR(0, MAX_R, MIN, MAX)).toBe(0)
    expect(calcPlanetVisR(-1, MAX_R, MIN, MAX)).toBe(0)
  })

  it('Ergebnis liegt zwischen visMin und maxR', () => {
    const r = calcPlanetVisR(1.0, MAX_R, MIN, MAX)
    expect(r).toBeGreaterThanOrEqual(MIN)
    expect(r).toBeLessThanOrEqual(MAX_R)
  })

  it('sehr kleiner Planet wird auf visMin geclampt', () => {
    const r = calcPlanetVisR(0.01, MAX_R, MIN, MAX)
    expect(r).toBe(MIN)
  })

  it('sehr großer Planet wird auf maxR geclampt', () => {
    const r = calcPlanetVisR(50, 0.05, MIN, MAX)
    expect(r).toBeLessThanOrEqual(0.05)
  })

  it('logarithmische Skalierung: Erde (1.0) kleiner als Jupiter (11.2)', () => {
    const earth   = calcPlanetVisR(1.0,  MAX_R, MIN, MAX)
    const jupiter = calcPlanetVisR(11.2, MAX_R, MIN, MAX)
    expect(jupiter).toBeGreaterThan(earth)
  })

  it('ist monoton steigend in radiusEarth', () => {
    const r1 = calcPlanetVisR(0.5, MAX_R, MIN, MAX)
    const r2 = calcPlanetVisR(1.0, MAX_R, MIN, MAX)
    const r3 = calcPlanetVisR(5.0, MAX_R, MIN, MAX)
    expect(r2).toBeGreaterThanOrEqual(r1)
    expect(r3).toBeGreaterThanOrEqual(r2)
  })
})

// ── computeOrbitPos ───────────────────────────────────────────────────────────

describe('computeOrbitPos', () => {
  it('Kreisbahn (ecc=0, theta=0) liegt bei (a, 0, 0)', () => {
    const pos = computeOrbitPos(5, 0, 0, 0, 0)
    expect(pos.x).toBeCloseTo(5, 5)
    expect(pos.y).toBeCloseTo(0, 5)
    expect(pos.z).toBeCloseTo(0, 5)
  })

  it('Kreisbahn (ecc=0, theta=π/2) liegt bei (0, 0, b)', () => {
    const pos = computeOrbitPos(5, 0, 0, 0, Math.PI / 2)
    expect(pos.x).toBeCloseTo(0, 5)
    expect(pos.y).toBeCloseTo(0, 5)
    expect(pos.z).toBeCloseTo(5, 5)
  })

  it('Kreisbahn (ecc=0): Abstand vom Ursprung = a', () => {
    for (const theta of [0, 0.5, 1.0, 2.0, Math.PI]) {
      const pos  = computeOrbitPos(3, 0, 0, 0, theta)
      const dist = Math.sqrt(pos.x ** 2 + pos.y ** 2 + pos.z ** 2)
      expect(dist).toBeCloseTo(3, 4)
    }
  })

  it('Ellipse (ecc=0.5): Perihel = a*(1-e), Aphel = a*(1+e)', () => {
    const a   = 10
    const ecc = 0.5
    const perihelPos = computeOrbitPos(a, ecc, 0, 0, 0)
    const aphelPos   = computeOrbitPos(a, ecc, 0, 0, Math.PI)

    // Perihelabstand = a*(1-e) = 5
    const perihelDist = Math.sqrt(perihelPos.x ** 2 + perihelPos.z ** 2)
    expect(perihelDist).toBeCloseTo(a * (1 - ecc), 4)

    // Aphelabstand = a*(1+e) = 15
    const aphelDist = Math.sqrt(aphelPos.x ** 2 + aphelPos.z ** 2)
    expect(aphelDist).toBeCloseTo(a * (1 + ecc), 4)
  })

  it('Inklination 90° kippt Orbit in XY-Ebene', () => {
    // Bei 90° Inklination und theta=π/2 liegt der Punkt auf der Y-Achse
    const pos = computeOrbitPos(5, 0, 0, 90, Math.PI / 2)
    // z-Komponente sollte nahe 0 sein, y-Komponente ≈ 5
    expect(Math.abs(pos.z)).toBeLessThan(0.01)
    expect(Math.abs(pos.y)).toBeCloseTo(5, 4)
  })
})

// ── makeLcg ──────────────────────────────────────────────────────────────────

describe('makeLcg', () => {
  it('gibt Werte in [0, 1) zurück', () => {
    const rng = makeLcg(0.42)
    for (let i = 0; i < 1000; i++) {
      const v = rng()
      expect(v).toBeGreaterThanOrEqual(0)
      expect(v).toBeLessThan(1)
    }
  })

  it('ist deterministisch: gleicher Seed → gleiche Sequenz', () => {
    const rng1 = makeLcg(0.7)
    const rng2 = makeLcg(0.7)
    for (let i = 0; i < 50; i++) {
      expect(rng1()).toBe(rng2())
    }
  })

  it('unterschiedliche Seeds → unterschiedliche erste Werte', () => {
    const v1 = makeLcg(0.1)()
    const v2 = makeLcg(0.9)()
    expect(v1).not.toBe(v2)
  })

  it('aufeinanderfolgende Werte sind verschieden (kein Fixpunkt)', () => {
    const rng = makeLcg(0.5)
    const values = Array.from({ length: 20 }, () => rng())
    const unique = new Set(values)
    expect(unique.size).toBeGreaterThan(15)
  })

  it('Verteilung deckt den Raum gut ab (keine Häufung in schmalen Bändern)', () => {
    const rng = makeLcg(0.333)
    const BINS = 10
    const counts = new Array(BINS).fill(0)
    const N = 2000
    for (let i = 0; i < N; i++) {
      const bin = Math.floor(rng() * BINS)
      counts[bin]++
    }
    // Jeder Bin soll mindestens 10 % der erwarteten Gleichverteilung erhalten
    const expected = N / BINS
    counts.forEach(c => expect(c).toBeGreaterThan(expected * 0.5))
  })
})
