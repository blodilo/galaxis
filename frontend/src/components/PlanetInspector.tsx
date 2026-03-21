import type { ReactNode } from 'react'
import type { Planet, Moon } from '../types/galaxy'

const TYPE_LABELS: Record<string, string> = {
  rocky:         'Gesteinsplanet',
  gas_giant:     'Gasriese',
  ice_giant:     'Eisriese',
  asteroid_belt: 'Asteroidengürtel',
}

// Farben pro Biochemie-Archetyp (spiegelt biochemistry_archetypes_v1.0.yaml wider)
const ARCH_COLORS: Record<string, string> = {
  terran:       '#4ade80',
  thermophilic: '#fb923c',
  anaerobic:    '#facc15',
  cryophilic:   '#7dd3fc',
  chlorine:     '#a3e635',
}

function fmt(n: number | undefined | null, dec = 2): string {
  if (n == null || isNaN(n as number)) return '—'
  return n.toLocaleString('de-DE', { maximumFractionDigits: dec })
}

function Row({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex justify-between gap-2 py-0.5 border-b border-slate-800">
      <span className="text-slate-500 shrink-0">{label}</span>
      <span className="text-slate-200 text-right break-all">{value}</span>
    </div>
  )
}

function TopGases({ comp }: { comp: Record<string, number> }) {
  const sorted = Object.entries(comp).sort((a, b) => b[1] - a[1]).slice(0, 4)
  if (!sorted.length) return <span className="text-slate-600">—</span>
  return <span>{sorted.map(([g, f]) => `${g} ${(f * 100).toFixed(0)}%`).join(' · ')}</span>
}

function TopResources({ res }: { res: Record<string, number> }) {
  const sorted = Object.entries(res).sort((a, b) => b[1] - a[1]).slice(0, 5)
  if (!sorted.length) return <span className="text-slate-600">—</span>
  return (
    <div className="flex flex-wrap gap-1 justify-end">
      {sorted.map(([id, amt]) => (
        <span key={id} className="bg-slate-800 px-1 rounded">
          {id.replace(/_/g, '\u00a0')} {(amt * 100).toFixed(0)}
        </span>
      ))}
    </div>
  )
}

function MoonList({ moons }: { moons: Moon[] }) {
  if (!moons.length) return <span className="text-slate-600">Keine</span>
  return (
    <div className="flex flex-col gap-0">
      {moons.map((m, i) => (
        <div key={m.id} className="flex justify-between py-0.5 border-b border-slate-800/50">
          <span className="text-slate-500">
            {i + 1}. {m.composition_type === 'rocky' ? 'Fels' : m.composition_type === 'icy' ? 'Eis' : 'Misch'}
          </span>
          <span className="text-slate-300">
            {fmt(m.mass_earth, 4)} M⊕ · {fmt(m.surface_temp_k, 0)} K
          </span>
        </div>
      ))}
    </div>
  )
}

interface Props {
  planet: Planet | null
}

export function PlanetInspector({ planet }: Props) {
  if (!planet) {
    return (
      <div className="flex flex-col gap-2">
        <span className="text-xs text-slate-400 uppercase tracking-widest">Planet</span>
        <p className="text-slate-600 text-xs mt-2">Klicke einen Planeten an.</p>
      </div>
    )
  }

  const tempC        = planet.surface_temp_k - 273.15
  const archColor    = ARCH_COLORS[planet.biochem_archetype] ?? '#6b7280'
  const dominantBio  = Object.entries(planet.biomass_potential ?? {})
    .sort((a, b) => b[1] - a[1])[0]

  return (
    <div className="flex flex-col gap-2 text-xs">

      {/* Kopf */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-slate-400 uppercase tracking-widest">Planet</span>
        <span className="text-slate-500">#{planet.orbit_index + 1}</span>
      </div>
      <div className="text-sm font-semibold text-white">
        {TYPE_LABELS[planet.planet_type] ?? planet.planet_type}
      </div>

      {/* Physik */}
      <div className="flex flex-col gap-0 mt-1">
        <Row label="Orbit (a)"   value={`${fmt(planet.orbit_distance_au)} AU`} />
        {planet.eccentricity > 0.001 && (
          <>
            <Row label="Exzentrizität" value={fmt(planet.eccentricity, 3)} />
            <Row label="Perihel"       value={`${fmt(planet.perihelion_au, 3)} AU`} />
            <Row label="Aphel"         value={`${fmt(planet.aphelion_au, 3)} AU`} />
            <Row
              label="T-Spanne (eq)"
              value={`${fmt(planet.temp_eq_min_k, 0)}–${fmt(planet.temp_eq_max_k, 0)} K`}
            />
          </>
        )}
        {planet.inclination_deg > 0.1 && (
          <Row label="Inklination" value={`${fmt(planet.inclination_deg, 1)}°`} />
        )}
        <Row label="Masse"       value={`${fmt(planet.mass_earth)} M⊕`} />
        <Row label="Radius"      value={`${fmt(planet.radius_earth)} R⊕`} />
        <Row label="Schwerkraft" value={`${fmt(planet.surface_gravity_g)} g`} />
        <Row label="Temperatur"  value={`${fmt(planet.surface_temp_k, 0)} K · ${fmt(tempC, 0)} °C`} />
        <Row label="Albedo"      value={fmt(planet.albedo)} />
        <Row label="Achsneigung" value={`${fmt(planet.axial_tilt_deg, 1)}°`} />
        <Row label="Rotation"    value={`${fmt(planet.rotation_period_h, 1)} h`} />
        {planet.has_rings && (
          <Row label="Ringe" value={<span className="text-amber-400">Ja</span>} />
        )}
      </div>

      {/* Atmosphäre (nur Rocky) */}
      {planet.planet_type === 'rocky' && (
        <div className="flex flex-col gap-0">
          <Row
            label="Atm. Druck"
            value={
              planet.atm_pressure_atm > 0
                ? `${fmt(planet.atm_pressure_atm, 3)} atm`
                : <span className="text-slate-600">Keine</span>
            }
          />
          {planet.atm_pressure_atm > 0 && (
            <Row label="Gase" value={<TopGases comp={planet.atm_composition ?? {}} />} />
          )}
          {planet.greenhouse_delta_k !== 0 && (
            <Row label="Treibhaus ΔT" value={`${fmt(planet.greenhouse_delta_k, 1)} K`} />
          )}
        </div>
      )}

      {/* Biochemie (nur Rocky) */}
      {planet.planet_type === 'rocky' && (
        <div className="border-t border-slate-800 pt-2 flex flex-col gap-1">
          <div className="flex justify-between items-center">
            <span className="text-slate-400 uppercase tracking-widest">Biochemie</span>
            {planet.biochem_archetype
              ? (
                <span
                  className="text-xs font-bold px-1.5 py-0.5 rounded"
                  style={{ color: archColor, border: `1px solid ${archColor}40` }}
                >
                  {planet.biochem_archetype}
                </span>
              )
              : <span className="text-slate-600">Unbewohnbar</span>
            }
          </div>
          {dominantBio && dominantBio[1] > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-slate-500 shrink-0">Biomasse</span>
              <div className="flex-1 h-1.5 bg-slate-800 rounded-full overflow-hidden">
                <div
                  className="h-full rounded-full"
                  style={{ width: `${dominantBio[1] * 100}%`, background: archColor }}
                />
              </div>
              <span style={{ color: archColor }}>{fmt(dominantBio[1] * 100, 0)} %</span>
            </div>
          )}
          {planet.usable_surface_fraction > 0 && (
            <Row label="Nutzfläche" value={`${fmt(planet.usable_surface_fraction * 100, 1)} %`} />
          )}
        </div>
      )}

      {/* Ressourcen */}
      <div className="border-t border-slate-800 pt-2">
        <span className="text-slate-400 uppercase tracking-widest block mb-1">Ressourcen</span>
        <TopResources res={planet.resource_deposits ?? {}} />
      </div>

      {/* Monde */}
      {planet.moons.length > 0 && (
        <div className="border-t border-slate-800 pt-2">
          <span className="text-slate-400 uppercase tracking-widest block mb-1">
            Monde ({planet.moons.length})
          </span>
          <MoonList moons={planet.moons} />
        </div>
      )}

    </div>
  )
}
