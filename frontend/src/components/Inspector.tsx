import type { Star } from '../types/galaxy'

interface Props {
  star: Star | null
  onViewSystem?: (star: Star) => void
}

const TYPE_LABELS: Record<string, string> = {
  O:'O-Stern (Blau-Riese)', B:'B-Stern (Blau-Weiß)', A:'A-Stern (Weiß)',
  F:'F-Stern (Gelblich-Weiß)', G:'G-Stern (Gelb)', K:'K-Stern (Orange)',
  M:'M-Stern (Roter Zwerg)', WR:'Wolf-Rayet-Stern', RStar:'Roter Überriese',
  SStar:'S-Stern (AGB)', Pulsar:'Neutronenstern / Pulsar',
  StellarBH:'Stellares Schwarzes Loch', SMBH:'Supermassives Schwarzes Loch',
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex justify-between gap-2 py-0.5 border-b border-slate-800">
      <span className="text-slate-500 shrink-0">{label}</span>
      <span className="text-slate-200 text-right break-all">{value}</span>
    </div>
  )
}

function fmt(n: number | undefined | null, decimals = 3): string {
  if (n == null || n === 0) return '—'
  return n.toLocaleString('de-DE', { maximumFractionDigits: decimals })
}

export function Inspector({ star, onViewSystem }: Props) {
  if (!star) {
    return (
      <div className="flex flex-col gap-2">
        <span className="text-xs text-slate-400 uppercase tracking-widest">Inspektor</span>
        <p className="text-slate-600 text-xs mt-2">Klicke einen Stern an, um Details zu sehen.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-slate-400 uppercase tracking-widest">Inspektor</span>
        <span
          className="w-3 h-3 rounded-full shrink-0"
          style={{ background: star.color_hex }}
        />
      </div>

      <div className="text-sm font-semibold text-white">{TYPE_LABELS[star.star_type] ?? star.star_type}</div>

      <div className="text-xs flex flex-col gap-0 mt-1">
        <Row label="ID" value={<span className="font-mono text-slate-400">{star.id.slice(0, 12)}…</span>} />
        <Row label="Position" value={`(${fmt(star.x, 0)}, ${fmt(star.y, 0)}, ${fmt(star.z, 0)}) ly`} />
        <Row label="Spektralklasse" value={star.spectral_class || '—'} />
        <Row label="Masse" value={star.mass_solar ? `${fmt(star.mass_solar)} M☉` : '—'} />
        <Row label="Leuchtkraft" value={star.luminosity_solar ? `${fmt(star.luminosity_solar)} L☉` : '—'} />
        <Row label="Radius" value={star.radius_solar ? `${fmt(star.radius_solar)} R☉` : '—'} />
        <Row label="Temperatur" value={star.temperature_k ? `${fmt(star.temperature_k, 0)} K` : '—'} />
        <Row label="Nebel-ID" value={star.nebula_id ? <span className="font-mono text-slate-400">{star.nebula_id.slice(0,12)}…</span> : '—'} />
        <Row
          label="Planeten"
          value={
            star.planets_generated
              ? <span className="text-emerald-400">Generiert</span>
              : <span className="text-amber-600">Kein Scan</span>
          }
        />
      </div>

      {star.planets_generated && onViewSystem && (
        <button
          onClick={() => onViewSystem(star)}
          className="mt-2 w-full text-xs py-1.5 rounded border border-cyan-700 text-cyan-400
                     hover:bg-cyan-900/30 transition-colors tracking-widest uppercase"
        >
          System anzeigen →
        </button>
      )}

    </div>
  )
}
