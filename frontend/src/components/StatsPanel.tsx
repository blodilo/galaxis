import type { Star, Galaxy } from '../types/galaxy'

interface Props {
  galaxy: Galaxy | null
  stars: Star[]
}

const COUNTS_ORDER = [
  ['M','K','G','F','A','B','O'],
  ['WR','RStar','SStar'],
  ['Pulsar','StellarBH','SMBH'],
]

const SHORT: Record<string, string> = {
  M:'M', K:'K', G:'G', F:'F', A:'A', B:'B', O:'O',
  WR:'WR', RStar:'R★', SStar:'S★',
  Pulsar:'PSR', StellarBH:'BH', SMBH:'SMBH',
}

const DOT: Record<string, string> = {
  O:'#9bb0ff', B:'#aabfff', A:'#cad7ff', F:'#f8f7ff', G:'#fff4ea',
  K:'#ffd2a1', M:'#ffb06a', WR:'#00e5ff', RStar:'#ff4400', SStar:'#ff7722',
  Pulsar:'#e0e8ff', StellarBH:'#ff6600', SMBH:'#ff9900',
}

export function StatsPanel({ galaxy, stars }: Props) {
  const counts = stars.reduce<Record<string, number>>((acc, s) => {
    acc[s.star_type] = (acc[s.star_type] ?? 0) + 1
    return acc
  }, {})

  return (
    <div className="flex flex-col gap-2">
      <span className="text-xs text-slate-400 uppercase tracking-widest">Statistik</span>

      {galaxy && (
        <div className="text-xs text-slate-500 flex flex-col gap-0.5">
          <div className="text-slate-300 font-semibold">{galaxy.name}</div>
          <div>Seed: {galaxy.seed}</div>
          <div className="flex items-center gap-1">
            Status:
            <span className={galaxy.status === 'ready' ? 'text-emerald-400' : 'text-amber-400'}>
              {galaxy.status}
            </span>
          </div>
          <div className="text-slate-200 font-semibold mt-1">
            {stars.length.toLocaleString('de-DE')} Sterne
          </div>
        </div>
      )}

      <div className="flex flex-col gap-0.5 mt-1">
        {COUNTS_ORDER.flat().map(type => {
          const n = counts[type] ?? 0
          if (!n) return null
          return (
            <div key={type} className="flex items-center justify-between text-xs">
              <span className="flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full" style={{ background: DOT[type] }} />
                <span className="text-slate-400">{SHORT[type]}</span>
              </span>
              <span className="text-slate-300">{n.toLocaleString('de-DE')}</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}
