import { useEffect, useState } from 'react'
import { fetchMySystems } from '../api/economy'
import type { MySystem } from '../api/economy'

interface Props {
  onSelect: (starId: string) => void
}

export function MySystemsPicker({ onSelect }: Props) {
  const [systems, setSystems] = useState<MySystem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState('')

  useEffect(() => {
    fetchMySystems()
      .then(setSystems)
      .catch(e => setError(String(e)))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-400 gap-3">
        <div className="w-6 h-6 border-2 border-slate-600 border-t-cyan-400 rounded-full animate-spin" />
        <span className="text-sm">Lade Systeme…</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full text-red-400 text-sm">{error}</div>
    )
  }

  if (systems.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3 text-slate-500">
        <p className="text-sm">Noch keine Systeme mit Anlagen.</p>
        <p className="text-xs text-slate-600">
          Im GOD MODE einen Stern wählen → Planet → "Heimatplaneten anlegen"
        </p>
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto p-6">
      <h1 className="text-xs font-bold tracking-widest text-emerald-400 uppercase mb-4">
        Meine Systeme
      </h1>
      <div className="space-y-2 max-w-lg">
        {systems.map(s => (
          <button
            key={s.star_id}
            onClick={() => onSelect(s.star_id)}
            className="w-full text-left px-4 py-3 bg-slate-900 border border-slate-700
                       hover:border-emerald-700 hover:bg-slate-800 rounded transition-colors group"
          >
            <div className="flex items-center justify-between">
              <span className="font-mono text-slate-300 text-xs group-hover:text-emerald-300 transition-colors">
                {s.star_id.slice(0, 8)}…
              </span>
              <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded ${
                s.running_count > 0
                  ? 'text-emerald-400 bg-emerald-900/30 border border-emerald-800'
                  : 'text-slate-500 bg-slate-800 border border-slate-700'
              }`}>
                {s.running_count > 0 ? `${s.running_count} aktiv` : 'idle'}
              </span>
            </div>
            <div className="flex gap-4 mt-1 text-[11px] text-slate-500">
              <span>{s.facility_count} Anlage{s.facility_count !== 1 ? 'n' : ''}</span>
              <span>{s.planet_count} Planet{s.planet_count !== 1 ? 'en' : ''}</span>
            </div>
          </button>
        ))}
      </div>
    </div>
  )
}
