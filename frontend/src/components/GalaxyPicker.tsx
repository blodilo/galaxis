import { useEffect, useRef } from 'react'
import type { Galaxy, GalaxyStatus } from '../types/galaxy'

interface Props {
  galaxies: Galaxy[]
  currentId: string | null
  onSelect: (g: Galaxy) => void   // load ready/active galaxy in viewer
  onResume: (g: Galaxy) => void   // continue step-by-step generation
  onNew: () => void               // open generator for a new galaxy
  onDelete: (g: Galaxy) => void   // delete galaxy
  onClose: () => void
}

interface StatusMeta {
  dot: string       // Tailwind color class for the dot
  label: string
  action: 'open' | 'resume' | 'wait' | 'none'
  actionLabel: string
}

function statusMeta(s: GalaxyStatus): StatusMeta {
  switch (s) {
    case 'ready':      return { dot: 'bg-emerald-500', label: 'Bereit',      action: 'open',   actionLabel: 'Öffnen →'      }
    case 'active':     return { dot: 'bg-cyan-500',    label: 'Aktiv',       action: 'open',   actionLabel: 'Öffnen →'      }
    case 'morphology': return { dot: 'bg-blue-400',    label: 'Schritt 1/4', action: 'resume', actionLabel: 'Fortsetzen →'  }
    case 'spectral':   return { dot: 'bg-blue-400',    label: 'Schritt 2/4', action: 'resume', actionLabel: 'Fortsetzen →'  }
    case 'objects':    return { dot: 'bg-blue-400',    label: 'Schritt 3/4', action: 'resume', actionLabel: 'Fortsetzen →'  }
    case 'generating': return { dot: 'bg-yellow-400',  label: 'Generiert…',  action: 'wait',   actionLabel: 'Läuft…'        }
    case 'error':      return { dot: 'bg-red-500',     label: 'Fehler',      action: 'resume', actionLabel: 'Prüfen →'      }
    default:           return { dot: 'bg-slate-600',   label: s,             action: 'none',   actionLabel: ''              }
  }
}

function formatDate(iso?: string): string {
  if (!iso) return ''
  try {
    return new Date(iso).toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: '2-digit' })
  } catch { return '' }
}

export function GalaxyPicker({ galaxies, currentId, onSelect, onResume, onNew, onDelete, onClose }: Props) {
  const ref = useRef<HTMLDivElement>(null)

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose()
    }
    // Use setTimeout to avoid closing on the same click that opened the picker
    const id = setTimeout(() => document.addEventListener('mousedown', handler), 0)
    return () => { clearTimeout(id); document.removeEventListener('mousedown', handler) }
  }, [onClose])

  // Close on Escape
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  return (
    <div
      ref={ref}
      className="absolute top-full left-0 mt-1 z-50 w-80
                 bg-slate-900 border border-slate-700 rounded-lg shadow-2xl
                 overflow-hidden"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-slate-800">
        <span className="text-[10px] uppercase tracking-widest text-slate-400 font-semibold">
          Galaxien
        </span>
        <span className="text-[10px] text-slate-600">{galaxies.length} gesamt</span>
      </div>

      {/* Galaxy list — max 10 rows, then scroll */}
      <div className="overflow-y-auto" style={{ maxHeight: 'calc(10 * 2.75rem)' }}>
        {galaxies.length === 0 && (
          <div className="px-3 py-4 text-xs text-slate-500 text-center">
            Noch keine Galaxien vorhanden.
          </div>
        )}
        {galaxies.map(g => {
          const meta = statusMeta(g.status)
          const isCurrent = g.id === currentId
          return (
            <div
              key={g.id}
              className={`flex items-center gap-2.5 px-3 h-11 border-b border-slate-800/60
                          ${isCurrent ? 'bg-slate-800/50' : 'hover:bg-slate-800/30'} transition-colors`}
            >
              {/* Status dot */}
              <div className={`w-2 h-2 rounded-full shrink-0 ${meta.dot}`} />

              {/* Name + metadata */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5 min-w-0">
                  <span className={`text-xs truncate font-medium
                    ${isCurrent ? 'text-white' : 'text-slate-200'}`}>
                    {g.name}
                  </span>
                  {isCurrent && (
                    <span className="text-[9px] text-slate-500 shrink-0">●</span>
                  )}
                </div>
                <div className="flex items-center gap-1.5 text-[9px] text-slate-500">
                  <span className={meta.dot.replace('bg-', 'text-').replace('-500', '-400').replace('-400', '-400')}>
                    {meta.label}
                  </span>
                  {g.star_count > 0 && (
                    <>
                      <span>·</span>
                      <span>{g.star_count.toLocaleString('de-DE')} Sterne</span>
                    </>
                  )}
                  {g.created_at && (
                    <>
                      <span>·</span>
                      <span>{formatDate(g.created_at)}</span>
                    </>
                  )}
                </div>
              </div>

              {/* Action button */}
              {meta.action === 'open' && (
                <button
                  onClick={() => { onSelect(g); onClose() }}
                  className="shrink-0 text-[10px] text-slate-400 hover:text-white transition-colors
                             px-2 py-0.5 rounded border border-slate-700 hover:border-slate-500"
                >
                  {meta.actionLabel}
                </button>
              )}
              {(meta.action === 'resume') && (
                <button
                  onClick={() => { onResume(g); onClose() }}
                  className="shrink-0 text-[10px] text-blue-400 hover:text-blue-200 transition-colors
                             px-2 py-0.5 rounded border border-blue-800 hover:border-blue-600"
                >
                  {meta.actionLabel}
                </button>
              )}
              {meta.action === 'wait' && (
                <span className="shrink-0 text-[10px] text-slate-600 px-2 py-0.5">
                  {meta.actionLabel}
                </span>
              )}

              {/* Delete button */}
              <button
                onClick={(e) => { e.stopPropagation(); onDelete(g) }}
                className="shrink-0 text-[10px] text-slate-600 hover:text-red-400 transition-colors px-1"
                title="Galaxie löschen"
              >
                ✕
              </button>
            </div>
          )
        })}
      </div>

      {/* Footer */}
      <div className="px-3 py-2 border-t border-slate-800">
        <button
          onClick={() => { onNew(); onClose() }}
          className="w-full text-[10px] text-slate-400 hover:text-white py-1.5 rounded
                     border border-dashed border-slate-700 hover:border-slate-500 transition-colors"
        >
          + Neue Galaxie generieren
        </button>
      </div>
    </div>
  )
}
