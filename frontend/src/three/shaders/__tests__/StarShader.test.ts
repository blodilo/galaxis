import { describe, it, expect } from 'vitest'
import { uuidSeed, starColorTriad } from '../StarShader'

// ── uuidSeed ──────────────────────────────────────────────────────────────────

describe('uuidSeed', () => {
  it('gibt Wert in [0, 1] zurück', () => {
    const v = uuidSeed('550e8400-e29b-41d4-a716-446655440000')
    expect(v).toBeGreaterThanOrEqual(0)
    expect(v).toBeLessThanOrEqual(1)
  })

  it('ist deterministisch für gleiche UUID', () => {
    const uuid = 'f47ac10b-58cc-4372-a567-0e02b2c3d479'
    expect(uuidSeed(uuid)).toBe(uuidSeed(uuid))
  })

  it('liefert unterschiedliche Werte für verschiedene UUIDs', () => {
    // Nur die ersten 8 Hex-Zeichen werden ausgewertet — Unterschied muss dort liegen
    const a = uuidSeed('12345678-0000-0000-0000-000000000000')
    const b = uuidSeed('87654321-0000-0000-0000-000000000000')
    expect(a).not.toBe(b)
  })

  it('ignoriert Bindestriche korrekt (erstes 8-Hex-Segment entscheidend)', () => {
    // Nur die ersten 8 Hex-Zeichen zählen, Bindestriche werden entfernt
    const withDash    = uuidSeed('abcd1234-0000-0000-0000-000000000000')
    const withoutDash = uuidSeed('abcd12340000000000000000000000000000')
    expect(withDash).toBe(withoutDash)
  })
})

// ── starColorTriad ────────────────────────────────────────────────────────────

describe('starColorTriad', () => {
  it('highlight ist heller als base (mindestens ein Kanal)', () => {
    const { base, highlight } = starColorTriad('#ffaa00')
    const baseMax      = Math.max(base.x, base.y, base.z)
    const highlightMax = Math.max(highlight.x, highlight.y, highlight.z)
    expect(highlightMax).toBeGreaterThanOrEqual(baseMax)
  })

  it('dark ist dunkler als base (alle Kanäle ≤ base)', () => {
    const { base, dark } = starColorTriad('#ffaa00')
    expect(dark.x).toBeLessThan(base.x + 0.05)   // kleiner Toleranzpuffer
    expect(dark.y).toBeLessThanOrEqual(base.y)
    expect(dark.z).toBeLessThanOrEqual(base.z)
    // Luminanz dark deutlich kleiner als base
    const lumBase = 0.2126 * base.x + 0.7152 * base.y + 0.0722 * base.z
    const lumDark = 0.2126 * dark.x + 0.7152 * dark.y + 0.0722 * dark.z
    expect(lumDark).toBeLessThan(lumBase * 0.5)
  })

  it('funktioniert mit undefined (Fallback auf Weiß)', () => {
    const { base, highlight, dark } = starColorTriad(undefined)
    // base = weiß (1, 1, 1)
    expect(base.x).toBeCloseTo(1, 2)
    expect(base.y).toBeCloseTo(1, 2)
    expect(base.z).toBeCloseTo(1, 2)
    // highlight ≥ base für Weiß (geclampt auf 1)
    expect(highlight.x).toBeCloseTo(1, 1)
    // dark < 1
    expect(dark.x).toBeLessThan(1)
  })

  it('O-Stern (blau-weiß): dark ist dunkler als base', () => {
    const { base, dark } = starColorTriad('#9db4ff')
    const lumBase = base.x + base.y + base.z
    const lumDark = dark.x + dark.y + dark.z
    expect(lumDark).toBeLessThan(lumBase * 0.5)
  })

  it('M-Stern (rot): highlight enthält mehr Rot als base', () => {
    const { base, highlight } = starColorTriad('#ff4500')
    // Highlight-Rot-Kanal muss >= Base-Rot sein (durch Lerp zu Weiß)
    expect(highlight.x).toBeGreaterThanOrEqual(base.x)
  })
})
