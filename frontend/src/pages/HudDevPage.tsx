/**
 * HudDevPage — CIC HUD Testbed (nur Dev-Build, Route: "hud-dev")
 *
 * Enthält drei parallele visuelle Optionen als Sub-Tabs:
 *  A · Expanse  — Hard Sci-Fi Radar (Amber, Sweep-Line)
 *  B · Military — Militär-HUD (Stahl-Blau, Retikel)
 *  C · Galaxis  — Eigene Sprache (Gravity Well + Nadel-Marker)
 */

import { useState } from 'react'
import { Canvas } from '@react-three/fiber'
import { OrbitControls } from '@react-three/drei'
import { CicOverlay } from '../hud/CicOverlay'
import { SceneExpanse }  from '../three/SceneExpanse'
import { SceneMilitary } from '../three/SceneMilitary'
import { SceneGalaxis }  from '../three/SceneGalaxis'
import type { ShipPosition } from '../three/DropLines'

// ─── Demo-Schiffe (identisch für alle drei Szenen) ────────────────────────────

export const DEMO_SHIPS: ShipPosition[] = [
  { id: 's1', x: -35, y: 22, z: -15 },  // weit außen, hoch
  { id: 's2', x:  20, y: 30, z:  18 },  // außen, sehr hoch
  { id: 's3', x: -12, y: 14, z:  28 },  // Mitteldistanz
  { id: 's4', x:  38, y: 18, z: -30 },  // außen (enemy)
  { id: 's5', x:   6, y: 28, z:  -8 },  // nahe Zentrum
  { id: 's6', x: -20, y:  9, z: -22 },  // flach über der Ebene
  { id: 's7', x:  14, y: 40, z:  35 },  // weit außen, sehr hoch
]

// ─── Szenen-Auswahl ───────────────────────────────────────────────────────────

type Style = 'expanse' | 'military' | 'galaxis'

const STYLES: { id: Style; label: string; desc: string }[] = [
  { id: 'expanse',  label: 'A · Expanse',  desc: 'Radar · Amber'        },
  { id: 'military', label: 'B · Military', desc: 'Präzision · Stahl'    },
  { id: 'galaxis',  label: 'C · Galaxis',  desc: 'Gravity Well · Nadel' },
]

// ─── Token-Palette ────────────────────────────────────────────────────────────

const TOKENS: { label: string; var: string }[] = [
  { label: 'bg',               var: '--color-galaxis-bg' },
  { label: 'surface',          var: '--color-galaxis-surface' },
  { label: 'border',           var: '--color-galaxis-border' },
  { label: 'muted',            var: '--color-galaxis-muted' },
  { label: 'cyan',             var: '--color-galaxis-cyan' },
  { label: 'orange',           var: '--color-galaxis-orange' },
  { label: 'green',            var: '--color-galaxis-green' },
  { label: 'red',              var: '--color-galaxis-red' },
  { label: 'amber',            var: '--color-galaxis-amber' },
  { label: 'range-friendly',   var: '--color-galaxis-range-friendly' },
  { label: 'range-enemy',      var: '--color-galaxis-range-enemy' },
  { label: 'range-scan-ping',  var: '--color-galaxis-range-scan-ping' },
  { label: 'grid-line',        var: '--color-galaxis-grid-line' },
  { label: 'grid-line-warped', var: '--color-galaxis-grid-line-warped' },
  { label: 'spline-valid',     var: '--color-galaxis-spline-valid' },
  { label: 'spline-invalid',   var: '--color-galaxis-spline-invalid' },
]

function TokenPalette() {
  return (
    <div
      className="absolute bottom-4 left-4 flex flex-col gap-1"
      style={{ pointerEvents: 'auto' }}
    >
      <p className="text-[10px] font-bold tracking-widest uppercase mb-1"
         style={{ color: 'var(--color-galaxis-muted)' }}>
        @creaminds/design · galaxis
      </p>
      {TOKENS.map(({ label, var: cssVar }) => (
        <div key={label} className="flex items-center gap-2">
          <div
            className="w-4 h-4 rounded-[4px] border"
            style={{
              background: `var(${cssVar})`,
              borderColor: 'var(--color-galaxis-border)',
            }}
          />
          <span className="text-[10px] font-mono"
                style={{ color: 'var(--color-galaxis-muted)' }}>
            {cssVar}
          </span>
        </div>
      ))}
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function HudDevPage() {
  const [style, setStyle] = useState<Style>('galaxis')

  return (
    <div className="absolute inset-0">

      <Canvas camera={{ position: [0, 60, 90], fov: 50 }}>
        {style === 'expanse'  && <SceneExpanse  ships={DEMO_SHIPS} />}
        {style === 'military' && <SceneMilitary ships={DEMO_SHIPS} />}
        {style === 'galaxis'  && <SceneGalaxis  ships={DEMO_SHIPS} />}
        <OrbitControls makeDefault enablePan enableZoom enableRotate />
      </Canvas>

      <CicOverlay>

        {/* Kopfzeile mit Sub-Tab-Auswahl */}
        <div
          className="absolute top-0 left-0 right-0 h-9 flex items-center px-4 gap-4 border-b"
          style={{
            background: 'rgba(6,10,18,0.90)',
            borderColor: 'var(--color-galaxis-border)',
            pointerEvents: 'auto',
          }}
        >
          <span className="text-[10px] font-bold tracking-[0.3em] uppercase"
                style={{ color: 'var(--color-galaxis-cyan)' }}>
            CIC
          </span>
          <span style={{ color: 'var(--color-galaxis-border)' }}>|</span>

          {/* Szenen-Tabs */}
          {STYLES.map(s => (
            <button
              key={s.id}
              onClick={() => setStyle(s.id)}
              className="flex flex-col items-start leading-none gap-[2px] px-2 py-1 rounded"
              style={{
                cursor: 'pointer',
                background: style === s.id ? 'rgba(0,212,255,0.08)' : 'transparent',
                borderBottom: style === s.id
                  ? '1px solid var(--color-galaxis-cyan)'
                  : '1px solid transparent',
              }}
            >
              <span className="text-[10px] font-bold tracking-widest uppercase"
                    style={{
                      color: style === s.id
                        ? 'var(--color-galaxis-cyan)'
                        : 'var(--color-galaxis-muted)',
                    }}>
                {s.label}
              </span>
              <span className="text-[8px] tracking-wider"
                    style={{ color: 'var(--color-galaxis-muted)' }}>
                {s.desc}
              </span>
            </button>
          ))}

          <span className="ml-auto text-[9px]"
                style={{ color: 'var(--color-galaxis-muted)' }}>
            Drag/Zoom im Canvas ↔ Tab-Wechsel unabhängig
          </span>
        </div>

        <TokenPalette />

      </CicOverlay>
    </div>
  )
}
