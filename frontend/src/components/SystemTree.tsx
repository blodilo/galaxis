import { useState } from 'react'
import type { Planet, Star } from '../types/galaxy'

const TYPE_SHORT: Record<string, string> = {
  rocky:         'Fels',
  gas_giant:     'Gas',
  ice_giant:     'Eis',
  asteroid_belt: 'Belt',
}

const PLANET_DOT_COLORS: Record<string, string> = {
  rocky:         '#c87941',
  gas_giant:     '#d4a86a',
  ice_giant:     '#6ab4d4',
  asteroid_belt: '#888888',
}

const MOON_COMP_COLORS: Record<string, string> = {
  rocky: '#a09080',
  icy:   '#b0c8d8',
  mixed: '#90a898',
}

const STAR_TYPE_LABELS: Record<string, string> = {
  O: 'O', B: 'B', A: 'A', F: 'F', G: 'G', K: 'K', M: 'M',
  WR: 'WR', RStar: 'R', SStar: 'S',
  Pulsar: 'PSR', StellarBH: 'BH', SMBH: 'SMBH',
}

interface Props {
  star: Star
  planets: Planet[]
  selectedPlanet: Planet | null
  onSelectPlanet: (p: Planet | null) => void
}

export function SystemTree({ star, planets, selectedPlanet, onSelectPlanet }: Props) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  const toggleExpand = (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setExpanded(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleSelectPlanet = (p: Planet) => {
    if (p.planet_type !== 'asteroid_belt' && p.moons.length > 0) {
      setExpanded(prev => new Set([...prev, p.id]))
    }
    onSelectPlanet(p.id === selectedPlanet?.id ? null : p)
  }

  return (
    <div className="flex flex-col gap-0 text-xs">

      {/* Sternsystem-Header */}
      <div className="flex items-center gap-2 py-1.5 mb-1 border-b border-slate-800">
        <div
          className="w-3 h-3 rounded-full shrink-0"
          style={{ background: star.color_hex ?? '#ffffff' }}
        />
        <span className="text-slate-300 font-semibold">
          {STAR_TYPE_LABELS[star.star_type] ?? star.star_type}-Stern
        </span>
        {star.spectral_class && (
          <span className="text-slate-600 ml-auto shrink-0">{star.spectral_class}</span>
        )}
      </div>

      {/* Planetenliste */}
      {planets.map((p, i) => {
        const isSelected = p.id === selectedPlanet?.id
        const isExpanded = expanded.has(p.id)
        const hasMoons = p.moons.length > 0

        return (
          <div key={p.id}>
            {/* Planetenzeile */}
            <div
              className={`flex items-center gap-1.5 py-1 px-1 rounded cursor-pointer transition-colors
                ${isSelected
                  ? 'bg-cyan-900/40 text-cyan-300'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
                }`}
              onClick={() => handleSelectPlanet(p)}
            >
              {/* Expand-Toggle */}
              <span
                className="w-3 text-center shrink-0 text-slate-600 hover:text-slate-400"
                onClick={hasMoons ? (e) => toggleExpand(p.id, e) : undefined}
              >
                {hasMoons ? (isExpanded ? '▾' : '▸') : '·'}
              </span>

              {/* Farbpunkt */}
              <div
                className="w-2 h-2 rounded-full shrink-0"
                style={{ background: PLANET_DOT_COLORS[p.planet_type] ?? '#888' }}
              />

              {/* Label + Abstand */}
              <span className="flex-1 truncate">
                {i + 1}. {TYPE_SHORT[p.planet_type] ?? p.planet_type}
              </span>
              <span className="text-slate-600 ml-auto shrink-0">
                {p.orbit_distance_au.toFixed(2)} AU
              </span>
            </div>

            {/* Mondliste (ausgeklappt) */}
            {isExpanded && hasMoons && (
              <div className="pl-5 flex flex-col gap-0">
                {p.moons.map((m, mi) => (
                  <div
                    key={m.id}
                    className="flex items-center gap-1.5 py-0.5 text-slate-600"
                  >
                    <div
                      className="w-1.5 h-1.5 rounded-full shrink-0"
                      style={{ background: MOON_COMP_COLORS[m.composition_type] ?? '#909090' }}
                    />
                    <span>
                      {String.fromCharCode(97 + mi)}.{' '}
                      {m.composition_type === 'rocky' ? 'Fels'
                        : m.composition_type === 'icy' ? 'Eis'
                        : 'Misch'}
                    </span>
                    <span className="ml-auto text-slate-700">
                      {m.surface_temp_k.toFixed(0)} K
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )
      })}

      {planets.length === 0 && (
        <p className="text-slate-700 py-2 pl-4">Keine Planeten</p>
      )}
    </div>
  )
}
