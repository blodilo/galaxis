import { useState } from 'react'
import type { GameParams } from '../types/generator'

interface Props {
  params: GameParams
  onChange: (p: GameParams) => void
}

// ── generic helpers ──────────────────────────────────────────────────────────

function setNested<T extends object>(obj: T, path: string[], value: unknown): T {
  if (path.length === 1) return { ...obj, [path[0]]: value }
  const key = path[0] as keyof T
  return { ...obj, [key]: setNested(obj[key] as object, path.slice(1), value) }
}

// ── sub-components ───────────────────────────────────────────────────────────

function NumField({
  label, value, onChange, step = 'any', hint,
}: {
  label: string; value: number; onChange: (v: number) => void
  step?: string | number; hint?: string
}) {
  return (
    <label className="flex flex-col gap-0.5">
      <span className="text-[10px] text-slate-400">{label}</span>
      {hint && <span className="text-[9px] text-slate-600 leading-tight">{hint}</span>}
      <input
        type="number"
        step={step}
        value={value}
        onChange={e => onChange(Number(e.target.value))}
        className="bg-slate-900 border border-slate-700 rounded px-2 py-1 text-xs text-slate-200
                   focus:outline-none focus:border-blue-500 w-full"
      />
    </label>
  )
}

function TextField({
  label, value, onChange, hint,
}: {
  label: string; value: string; onChange: (v: string) => void; hint?: string
}) {
  return (
    <label className="flex flex-col gap-0.5">
      <span className="text-[10px] text-slate-400">{label}</span>
      {hint && <span className="text-[9px] text-slate-600 leading-tight">{hint}</span>}
      <input
        type="text"
        value={value}
        onChange={e => onChange(e.target.value)}
        className="bg-slate-900 border border-slate-700 rounded px-2 py-1 text-xs text-slate-200
                   focus:outline-none focus:border-blue-500 w-full"
      />
    </label>
  )
}

function MapField({
  label, value, onChange,
}: {
  label: string; value: Record<string, number> | null; onChange: (v: Record<string, number>) => void
}) {
  const map = value ?? {}
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[10px] text-slate-400">{label}</span>
      <div className="border border-slate-800 rounded overflow-hidden">
        {Object.entries(map).map(([k, v]) => (
          <div key={k} className="flex items-center gap-1 px-2 py-0.5 border-b border-slate-800 last:border-0">
            <span className="text-[10px] text-slate-500 w-40 shrink-0 truncate font-mono">{k}</span>
            <input
              type="number"
              step="any"
              value={v}
              onChange={e => onChange({ ...map, [k]: Number(e.target.value) })}
              className="bg-transparent border border-slate-700 rounded px-1.5 py-0.5 text-[10px]
                         text-slate-300 focus:outline-none focus:border-blue-500 flex-1 min-w-0"
            />
          </div>
        ))}
        {Object.keys(map).length === 0 && (
          <span className="text-[10px] text-slate-600 px-2 py-1 block italic">Keine Einträge</span>
        )}
      </div>
    </div>
  )
}

// ── collapsible section ──────────────────────────────────────────────────────

function Section({ title, tag, children }: { title: string; tag?: string; children: React.ReactNode }) {
  const [open, setOpen] = useState(true)
  return (
    <div className="border border-slate-800 rounded overflow-hidden">
      <button
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center justify-between px-3 py-2 bg-slate-900/60 hover:bg-slate-800/60 transition-colors"
      >
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold text-slate-300">{title}</span>
          {tag && <span className="text-[9px] text-slate-600 font-mono">{tag}</span>}
        </div>
        <span className="text-slate-600 text-xs">{open ? '▲' : '▼'}</span>
      </button>
      {open && (
        <div className="px-3 py-3 grid grid-cols-2 gap-x-4 gap-y-2 bg-slate-950/40">
          {children}
        </div>
      )}
    </div>
  )
}

// ── main component ───────────────────────────────────────────────────────────

export function ParamEditor({ params, onChange }: Props) {
  function set(section: keyof GameParams, field: string, value: unknown) {
    onChange({ ...params, [section]: { ...(params[section] as object), [field]: value } })
  }

  function setDeep(section: keyof GameParams, path: string[], value: unknown) {
    const updated = setNested(params[section] as object, path, value)
    onChange({ ...params, [section]: updated })
  }

  const g = params.galaxy
  const f = params.ftlw
  const s = params.sensors
  const t = params.time
  const e = params.economy
  const pg = params.planet_generation
  const res = params.research
  const c = params.combat
  const srv = params.server

  return (
    <div className="flex flex-col gap-2 overflow-y-auto">
      <span className="text-xs text-slate-400 uppercase tracking-widest">Parameter</span>

      {/* 1. Galaxie */}
      <Section title="Galaxie" tag="galaxy">
        <NumField label="Seed" value={g.seed} onChange={v => set('galaxy', 'seed', v)} step={1} />
        <NumField label="Sternanzahl" value={g.num_stars} onChange={v => set('galaxy', 'num_stars', v)} step={1000} hint="[PERFORMANCE]" />
        <NumField label="Radius (ly)" value={g.radius_ly} onChange={v => set('galaxy', 'radius_ly', v)} step={1000} />
        <NumField label="Spiralarme" value={g.arms} onChange={v => set('galaxy', 'arms', v)} step={1} />
        <NumField label="Arm-Winding" value={g.arm_winding} onChange={v => set('galaxy', 'arm_winding', v)} step={0.05} hint="[KALIBRIERUNG]" />
        <NumField label="Arm-Spread" value={g.arm_spread} onChange={v => set('galaxy', 'arm_spread', v)} step={0.05} />
        <NumField label="SMBH-Masse (M☉)" value={g.smbh_mass_solar} onChange={v => set('galaxy', 'smbh_mass_solar', v)} step={100000} />
        <TextField label="Morphologie-Typ" value={g.type} onChange={v => set('galaxy', 'type', v)} hint="Sa / Sb / SBb / Irr …" />
      </Section>

      {/* 2. FTLW */}
      <Section title="FTL-Widerstandsfeld" tag="ftlw">
        <NumField label="Vakuum-Basis" value={f.vacuum_base} onChange={v => set('ftlw', 'vacuum_base', v)} hint="[KALIBRIERUNG]" />
        <NumField label="K-Faktor" value={f.k_factor} onChange={v => set('ftlw', 'k_factor', v)} hint="[KALIBRIERUNG]" />
        <NumField label="Cutoff (%)" value={f.cutoff_percent} onChange={v => set('ftlw', 'cutoff_percent', v)} hint="[PERFORMANCE]" />
        <NumField label="Voxel-Größe (ly)" value={f.voxel_size_ly} onChange={v => set('ftlw', 'voxel_size_ly', v)} step={50} hint="[PERFORMANCE]" />
        <NumField label="Grob-Voxel (ly)" value={f.coarse_voxel_size_ly} onChange={v => set('ftlw', 'coarse_voxel_size_ly', v)} step={250} hint="[PERFORMANCE]" />
        <NumField label="Pulsar-Multiplikator" value={f.pulsar_multiplier} onChange={v => set('ftlw', 'pulsar_multiplier', v)} hint="[BALANCING]" />
        <NumField label="Schwarzes-Loch-Mult." value={f.black_hole_multiplier} onChange={v => set('ftlw', 'black_hole_multiplier', v)} hint="[BALANCING]" />
      </Section>

      {/* 3. Sensoren */}
      <Section title="Sensoren & Fog of War" tag="sensors">
        <NumField label="Optisch-K" value={s.optical_k} onChange={v => set('sensors', 'optical_k', v)} hint="[KALIBRIERUNG]" />
        <NumField label="FTL-K" value={s.ftl_k} onChange={v => set('sensors', 'ftl_k', v)} hint="[KALIBRIERUNG]" />
        <NumField label="Thermisch-K" value={s.ship_thermal_k} onChange={v => set('sensors', 'ship_thermal_k', v)} hint="[KALIBRIERUNG]" />
        <NumField label="Survey (Ticks)" value={s.survey_duration_ticks} onChange={v => set('sensors', 'survey_duration_ticks', v)} step={1} hint="[BALANCING]" />
        <NumField label="LKP-Decay (Ticks)" value={s.last_known_position_decay_ticks} onChange={v => set('sensors', 'last_known_position_decay_ticks', v)} step={1} hint="[BALANCING]" />
        {/* Info quality sub-section */}
        <div className="col-span-2 mt-1">
          <span className="text-[10px] text-slate-500 block mb-1">Detektions-Qualitätsschwellen</span>
          <div className="grid grid-cols-2 gap-x-4 gap-y-2">
            <NumField label="Volldetail (&lt; x)" value={s.info_quality.full_detail_threshold} onChange={v => setDeep('sensors', ['info_quality', 'full_detail_threshold'], v)} />
            <NumField label="Mitteldetail (&lt; x)" value={s.info_quality.medium_detail_threshold} onChange={v => setDeep('sensors', ['info_quality', 'medium_detail_threshold'], v)} />
            <NumField label="Niedrigdetail (&lt; x)" value={s.info_quality.low_detail_threshold} onChange={v => setDeep('sensors', ['info_quality', 'low_detail_threshold'], v)} />
          </div>
        </div>
        <div className="col-span-2 mt-1">
          <MapField label="Sensor-Ratings (m²)" value={s.sensor_ratings} onChange={v => set('sensors', 'sensor_ratings', v)} />
        </div>
        <div className="col-span-2 mt-1">
          <MapField label="Thermische Signaturen (L☉)" value={s.ship_thermal_signatures} onChange={v => set('sensors', 'ship_thermal_signatures', v)} />
        </div>
      </Section>

      {/* 4. Zeit */}
      <Section title="Zeitverlauf & Ticks" tag="time">
        <NumField label="Strategie-Tick (min)" value={t.strategy_tick_minutes} onChange={v => set('time', 'strategy_tick_minutes', v)} step={1} hint="[BALANCING]" />
        <NumField label="Kampf-Tick (s)" value={t.combat_tick_seconds} onChange={v => set('time', 'combat_tick_seconds', v)} step={1} hint="[BALANCING]" />
        <NumField label="Combat-Opt-In (h)" value={t.combat_opt_in_window_hours} onChange={v => set('time', 'combat_opt_in_window_hours', v)} step={1} hint="[BALANCING]" />
        <NumField label="Max Action-Queue" value={t.max_action_queue_depth} onChange={v => set('time', 'max_action_queue_depth', v)} step={1} hint="[BALANCING]" />
      </Section>

      {/* 5. Wirtschaft */}
      <Section title="Wirtschaft & Produktion" tag="economy">
        <NumField label="Detail-Effizienz-Bonus" value={e.detail_mode_efficiency_bonus} onChange={v => set('economy', 'detail_mode_efficiency_bonus', v)} hint="[BALANCING]" />
        <NumField label="Detail-Upgrade-Downtime" value={e.detail_mode_upgrade_downtime_ticks} onChange={v => set('economy', 'detail_mode_upgrade_downtime_ticks', v)} step={1} hint="[BALANCING]" />
        <NumField label="Detail-Break-Even" value={e.detail_mode_break_even_ticks} onChange={v => set('economy', 'detail_mode_break_even_ticks', v)} step={1} hint="[BALANCING]" />
        <NumField label="Bevölk.-Wachstum/mo" value={e.base_population_growth_rate} onChange={v => set('economy', 'base_population_growth_rate', v)} hint="[BALANCING]" />
        <NumField label="Steuerrate (Basis)" value={e.tax_rate_base} onChange={v => set('economy', 'tax_rate_base', v)} hint="[BALANCING]" />
        <NumField label="Oberflächen-Kostenexp." value={e.planet_surface_cost_exponent} onChange={v => set('economy', 'planet_surface_cost_exponent', v)} hint="[BALANCING]" />
        <NumField label="Asteroid-Ertrag-Mult." value={e.asteroid_yield_multiplier} onChange={v => set('economy', 'asteroid_yield_multiplier', v)} hint="[BALANCING]" />
      </Section>

      {/* 6. Planetensystem */}
      <Section title="Planetensystem-Generierung" tag="planet_generation">
        <NumField label="Frostgrenze (AU-Konst.)" value={pg.frost_line_constant_au} onChange={v => set('planet_generation', 'frost_line_constant_au', v)} hint="[KALIBRIERUNG]" />
        <NumField label="Kollisionsmond-Prob." value={pg.moon_collision_probability} onChange={v => set('planet_generation', 'moon_collision_probability', v)} hint="[BALANCING]" />
        <NumField label="Gasriesen-Monde Min" value={pg.gas_giant_moon_count_min} onChange={v => set('planet_generation', 'gas_giant_moon_count_min', v)} step={1} hint="[BALANCING]" />
        <NumField label="Gasriesen-Monde Max" value={pg.gas_giant_moon_count_max} onChange={v => set('planet_generation', 'gas_giant_moon_count_max', v)} step={1} hint="[BALANCING]" />
        <NumField label="Max Planeten/System" value={pg.max_planets_per_system} onChange={v => set('planet_generation', 'max_planets_per_system', v)} step={1} />
        <NumField label="Nutzfläche terran (Basis)" value={pg.usable_surface_terran_base} onChange={v => set('planet_generation', 'usable_surface_terran_base', v)} hint="[BALANCING]" />
        <NumField label="Nutzfläche hostile (Basis)" value={pg.usable_surface_hostile_base} onChange={v => set('planet_generation', 'usable_surface_hostile_base', v)} hint="[BALANCING]" />
        {pg.atmosphere_type_weights && Object.keys(pg.atmosphere_type_weights).length > 0 && (
          <div className="col-span-2 mt-1">
            <MapField label="Atmosphären-Typ-Gewichte" value={pg.atmosphere_type_weights} onChange={v => set('planet_generation', 'atmosphere_type_weights', v)} />
          </div>
        )}
      </Section>

      {/* 7. Forschung */}
      <Section title="Forschung & Tech-Baum" tag="research">
        <NumField label="Forschungsgeschw. (Mult.)" value={res.base_research_speed} onChange={v => set('research', 'base_research_speed', v)} hint="[BALANCING]" />
        <NumField label="Wissenschaftler-Bonus" value={res.scientist_research_bonus} onChange={v => set('research', 'scientist_research_bonus', v)} hint="[BALANCING]" />
        <NumField label="Risikoreduktion/Wiss." value={res.scientist_risk_reduction} onChange={v => set('research', 'scientist_risk_reduction', v)} hint="[BALANCING]" />
        <NumField label="Labor-Bonus-Faktor" value={res.lab_bonus_factor} onChange={v => set('research', 'lab_bonus_factor', v)} hint="[BALANCING]" />
        <NumField label="Parallele Forschungsslots" value={res.parallel_research_slots} onChange={v => set('research', 'parallel_research_slots', v)} step={1} hint="[BALANCING]" />
      </Section>

      {/* 8. Kampf */}
      <Section title="Kampf" tag="combat">
        <NumField label="Railgun-Basis (km/s)" value={c.railgun_base_velocity_km_s} onChange={v => set('combat', 'railgun_base_velocity_km_s', v)} hint="[BALANCING]" />
        <NumField label="Graser-Antimaterie/Schuss" value={c.graser_antimaterie_cost_per_shot} onChange={v => set('combat', 'graser_antimaterie_cost_per_shot', v)} hint="[BALANCING]" />
        <NumField label="Sandcaster-Radius (km)" value={c.sandcaster_intercept_radius_km} onChange={v => set('combat', 'sandcaster_intercept_radius_km', v)} hint="[BALANCING]" />
        <NumField label="Gefechts-Arena (km)" value={c.combat_arena_radius_km} onChange={v => set('combat', 'combat_arena_radius_km', v)} hint="[BALANCING]" />
      </Section>

      {/* 9. Server */}
      <Section title="Server" tag="server">
        <TextField label="Instanzname" value={srv.instance_name} onChange={v => set('server', 'instance_name', v)} />
        <NumField label="Max. Spieler" value={srv.max_players} onChange={v => set('server', 'max_players', v)} step={1} />
        <NumField label="Max. KI-Fraktionen" value={srv.max_ai_factions} onChange={v => set('server', 'max_ai_factions', v)} step={1} />
      </Section>
    </div>
  )
}
