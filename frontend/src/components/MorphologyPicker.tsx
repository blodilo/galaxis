import type { MorphologyTemplate } from '../types/generator'

interface Props {
  templates: MorphologyTemplate[]
  selected: string
  onSelect: (id: string) => void
}

const HUBBLE_COLORS: Record<string, string> = {
  Sa: '#a78bfa', Sb: '#818cf8', Sc: '#60a5fa',
  SBa: '#f472b6', SBb: '#fb7185', SBc: '#f87171',
  Irr: '#34d399',
}

export function MorphologyPicker({ templates, selected, onSelect }: Props) {
  if (templates.length === 0) {
    return (
      <div className="text-xs text-slate-500 italic">Keine Morphologie-Templates verfügbar.</div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <span className="text-xs text-slate-400 uppercase tracking-widest">Morphologie</span>
      <div className="grid grid-cols-2 gap-2">
        {templates.map(t => {
          const isSelected = t.id === selected
          const color = HUBBLE_COLORS[t.hubble_type] ?? '#94a3b8'
          return (
            <button
              key={t.id}
              onClick={() => onSelect(t.id)}
              className={`flex flex-col rounded border transition-all text-left overflow-hidden
                ${isSelected
                  ? 'border-blue-500 ring-1 ring-blue-500/40'
                  : 'border-slate-700 hover:border-slate-500'}`}
            >
              {/* Thumbnail */}
              <div className="relative w-full aspect-square bg-slate-900 overflow-hidden">
                <img
                  src={t.thumbnail_url}
                  alt={t.name}
                  className="w-full h-full object-cover opacity-90"
                  loading="lazy"
                  onError={e => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
                />
                {/* Hubble type badge */}
                <span
                  className="absolute top-1 left-1 text-[10px] font-bold px-1.5 py-0.5 rounded"
                  style={{ background: color + '33', color, border: `1px solid ${color}55` }}
                >
                  {t.hubble_type}
                </span>
                {isSelected && (
                  <div className="absolute inset-0 bg-blue-500/10 border-2 border-blue-500/60 rounded" />
                )}
              </div>

              {/* Caption */}
              <div className="px-2 py-1.5">
                <div className="text-xs font-medium text-slate-200 truncate">{t.name}</div>
                <div className="text-[10px] text-slate-500 truncate mt-0.5">{t.hubble_description}</div>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
