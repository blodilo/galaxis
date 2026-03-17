import type { StarFilter } from '../types/galaxy'

const STAR_GROUPS = [
  { label: 'Hauptreihe', keys: ['O','B','A','F','G','K','M'] as const },
  { label: 'Exotika',    keys: ['WR','RStar','SStar','Pulsar','StellarBH','SMBH'] as const },
  { label: 'Nebel',      keys: ['HII','SNR','Globular'] as const },
  { label: 'Overlay',    keys: ['showFTLW'] as const },
]

const LABELS: Record<string, string> = {
  O:'O', B:'B', A:'A', F:'F', G:'G', K:'K', M:'M',
  WR:'Wolf-Rayet', RStar:'Roter Riese', SStar:'S-Stern',
  Pulsar:'Pulsar', StellarBH:'Schwarzes Loch', SMBH:'SMBH',
  HII:'H-II Region', SNR:'Supernova-Überrest', Globular:'Kugelsternhaufen',
  showFTLW:'FTLW-Heatmap',
}

const DOT_COLORS: Record<string, string> = {
  O:'#9bb0ff', B:'#aabfff', A:'#cad7ff', F:'#f8f7ff',
  G:'#fff4ea', K:'#ffd2a1', M:'#ffb06a',
  WR:'#00e5ff', RStar:'#ff4400', SStar:'#ff7722',
  Pulsar:'#e0e8ff', StellarBH:'#ff6600', SMBH:'#ff9900',
  HII:'#ff2244', SNR:'#22aaff', Globular:'#ffcc44',
  showFTLW:'#22ff88',
}

interface Props {
  filter: StarFilter
  onChange: (f: StarFilter) => void
}

export function FilterPanel({ filter, onChange }: Props) {
  const toggle = (key: string) =>
    onChange({ ...filter, [key]: !filter[key as keyof StarFilter] })

  const allOn  = Object.values(filter).every(Boolean)
  const toggleAll = () => {
    const next = !allOn
    const newFilter = {} as StarFilter
    for (const k of Object.keys(filter)) {
      (newFilter as any)[k] = next
    }
    onChange(newFilter)
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-xs text-slate-400 uppercase tracking-widest">Filter</span>
        <button
          onClick={toggleAll}
          className="text-xs text-slate-500 hover:text-slate-300 transition-colors"
        >
          {allOn ? 'Alle aus' : 'Alle an'}
        </button>
      </div>

      {/* Planeten-Filter */}
      <div>
        <div className="text-xs text-slate-500 mb-1">Planeten</div>
        <div className="flex flex-col gap-0.5">
          <button
            onClick={() => toggle('onlyWithPlanets')}
            className={`flex items-center gap-2 px-2 py-1 rounded text-xs text-left transition-colors
              ${filter.onlyWithPlanets ? 'bg-slate-800 text-slate-200' : 'text-slate-600 hover:text-slate-500'}`}
          >
            <span
              className="w-2 h-2 rounded-full shrink-0"
              style={{ background: filter.onlyWithPlanets ? '#4ade80' : '#334155' }}
            />
            Nur mit Planeten
          </button>
        </div>
      </div>

      {STAR_GROUPS.map(group => (
        <div key={group.label}>
          <div className="text-xs text-slate-500 mb-1">{group.label}</div>
          <div className="flex flex-col gap-0.5">
            {group.keys.map(key => {
              const active = filter[key as keyof StarFilter]
              return (
                <button
                  key={key}
                  onClick={() => toggle(key)}
                  className={`flex items-center gap-2 px-2 py-1 rounded text-xs text-left transition-colors
                    ${active ? 'bg-slate-800 text-slate-200' : 'text-slate-600 hover:text-slate-500'}`}
                >
                  <span
                    className="w-2 h-2 rounded-full shrink-0"
                    style={{ background: active ? DOT_COLORS[key] : '#334155' }}
                  />
                  {LABELS[key]}
                </button>
              )
            })}
          </div>
        </div>
      ))}
    </div>
  )
}
