import { useState } from 'react'
import { useVisualParams } from '../context/VisualParamsContext'
import { PARAM_RANGES, TYPE_SIZE_RANGES, DEFAULT_VISUAL_PARAMS, SPECTRAL_COLOR_LABELS } from '../config/visualParams'

// ── Primitives ────────────────────────────────────────────────────────────────

function Toggle({
  label, checked, onChange,
}: {
  label: string; checked: boolean; onChange: (v: boolean) => void
}) {
  return (
    <label className="flex items-center gap-2 cursor-pointer select-none">
      <input
        type="checkbox" checked={checked} onChange={e => onChange(e.target.checked)}
        className="w-3 h-3 accent-cyan-400 cursor-pointer"
      />
      <span className="text-[10px] text-slate-300">{label}</span>
    </label>
  )
}

function Slider({
  label, value, min, max, step, onChange,
}: {
  label: string; value: number; min: number; max: number; step: number
  onChange: (v: number) => void
}) {
  return (
    <label className="flex flex-col gap-0.5">
      <div className="flex justify-between text-[10px] text-slate-400">
        <span>{label}</span>
        <span className="text-slate-300 tabular-nums">{value.toFixed(step < 1 ? 2 : 0)}</span>
      </div>
      <input
        type="range" min={min} max={max} step={step} value={value}
        onChange={e => onChange(parseFloat(e.target.value))}
        className="w-full h-1 accent-cyan-400 cursor-pointer"
      />
    </label>
  )
}

function Section({
  title, children, onReset,
}: {
  title: string; children: React.ReactNode; onReset?: () => void
}) {
  const [open, setOpen] = useState(true)
  return (
    <div className="border border-slate-800 rounded">
      <button
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center justify-between px-2 py-1.5 text-[10px] font-bold
                   tracking-widest text-slate-400 uppercase hover:text-slate-200 transition-colors"
      >
        <span>{open ? '▾' : '▸'} {title}</span>
        {onReset && open && (
          <span
            role="button"
            onClick={e => { e.stopPropagation(); onReset() }}
            className="text-[9px] text-slate-600 hover:text-red-400 transition-colors normal-case tracking-normal font-normal"
          >
            Reset
          </span>
        )}
      </button>
      {open && (
        <div className="px-2 pb-2 flex flex-col gap-2">
          {children}
        </div>
      )}
    </div>
  )
}

// ── Main Panel ────────────────────────────────────────────────────────────────

export function VisualTuner() {
  const { params, setParam, setTypeSize, resetSection, resetAll } = useVisualParams()
  const spectralTypeOrder = ['O','B','A','F','G','K','M','WR','RStar','SStar','Pulsar','StellarBH','SMBH']

  return (
    <div
      className="absolute top-10 right-0 bottom-0 w-64 z-20
                 bg-black/85 border-l border-slate-800 backdrop-blur-sm
                 overflow-y-auto flex flex-col gap-2 p-2"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-1 pt-1 pb-0.5">
        <span className="text-[10px] font-bold tracking-widest text-slate-300 uppercase">
          Visual Tuning
        </span>
        <button
          onClick={resetAll}
          className="text-[9px] text-slate-600 hover:text-red-400 transition-colors"
        >
          Alles reset
        </button>
      </div>

      {/* Post-processing */}
      <Section title="Post-Processing" onReset={() => resetSection('postprocessing')}>
        <Slider {...PARAM_RANGES.exposure!}     value={params.exposure}       onChange={v => setParam('exposure', v)} />
        <Slider {...PARAM_RANGES.bloomIntensity!} value={params.bloomIntensity} onChange={v => setParam('bloomIntensity', v)} />
        <Slider {...PARAM_RANGES.bloomThreshold!} value={params.bloomThreshold} onChange={v => setParam('bloomThreshold', v)} />
        <Slider {...PARAM_RANGES.bloomSmoothing!} value={params.bloomSmoothing} onChange={v => setParam('bloomSmoothing', v)} />
      </Section>

      {/* Stars global */}
      <Section title="Sterne – Global" onReset={() => resetSection('stars')}>
        <Slider {...PARAM_RANGES.starSizeScale!}  value={params.starSizeScale}  onChange={v => setParam('starSizeScale', v)} />
        <Slider {...PARAM_RANGES.starSizeCap!}    value={params.starSizeCap}    onChange={v => setParam('starSizeCap', v)} />
        <Slider {...PARAM_RANGES.starPointScale!} value={params.starPointScale} onChange={v => setParam('starPointScale', v)} />
        <Slider {...PARAM_RANGES.starGaussian!}   value={params.starGaussian}   onChange={v => setParam('starGaussian', v)} />
      </Section>

      {/* Per-type sizes */}
      <Section title="Sterntypen – Größen" onReset={() => resetSection('typesizes')}>
        {Object.entries(TYPE_SIZE_RANGES).map(([type, range]) => (
          <Slider
            key={type}
            {...range}
            value={params.typeSizes[type] ?? DEFAULT_VISUAL_PARAMS.typeSizes[type]}
            onChange={v => setTypeSize(type, v)}
          />
        ))}
      </Section>

      {/* Stern-Shader */}
      <Section title="Stern-Shader" onReset={() => resetSection('shader')}>
        {([
          [0, 'Klassisch (Value Noise)'],
          [1, 'Gradient + Domain Warp'],
          [2, 'Plasma (cosNoise)'],
          [3, '2D-Projektion (Neuimpl.)'],
          [4, '2D-Projektion + Warp'],
          [5, 'Voronoi-Granulation'],
        ] as [number, string][]).map(([v, label]) => (
          <label key={v} className="flex items-center gap-2 cursor-pointer select-none">
            <input
              type="radio"
              name="starShaderVariant"
              checked={params.starShaderVariant === v}
              onChange={() => setParam('starShaderVariant', v as 0|1|2|3|4|5)}
              className="w-3 h-3 accent-cyan-400 cursor-pointer"
            />
            <span className="text-[10px] text-slate-300">{label}</span>
          </label>
        ))}
      </Section>

      {/* V5 Voronoi params — nur bei aktivem Shader */}
      {params.starShaderVariant === 5 && (
        <Section title="Voronoi-Granulation" onReset={() => resetSection('voronoi')}>
          <Slider {...PARAM_RANGES.v5CellScale!}  value={params.v5CellScale}  onChange={v => setParam('v5CellScale',  v)} />
          <Slider {...PARAM_RANGES.v5Lifetime!}   value={params.v5Lifetime}   onChange={v => setParam('v5Lifetime',   v)} />
          <Slider {...PARAM_RANGES.v5RiseTime!}   value={params.v5RiseTime}   onChange={v => setParam('v5RiseTime',   v)} />
          <Slider {...PARAM_RANGES.v5MaxRadius!}  value={params.v5MaxRadius}  onChange={v => setParam('v5MaxRadius',  v)} />
          <Slider {...PARAM_RANGES.v5LaneWidth!}  value={params.v5LaneWidth}  onChange={v => setParam('v5LaneWidth',  v)} />
        </Section>
      )}

      {/* Star lighting */}
      <Section title="Sterne – Beleuchtung" onReset={() => resetSection('starlighting')}>
        <Slider {...PARAM_RANGES.starLuminosity!} value={params.starLuminosity} onChange={v => setParam('starLuminosity', v)} />
        <Slider {...PARAM_RANGES.starAnimSpeed!}  value={params.starAnimSpeed}  onChange={v => setParam('starAnimSpeed', v)} />
      </Section>

      {/* Spectral colors */}
      <Section title="Spektralfarben" onReset={() => resetSection('starcolors')}>
        {spectralTypeOrder.map(type => (
          <label key={type} className="flex items-center justify-between gap-2">
            <span className="text-[10px] text-slate-400 shrink-0 w-24">{SPECTRAL_COLOR_LABELS[type] ?? type}</span>
            <input
              type="color"
              value={params.spectralColors[type] ?? '#ffffff'}
              onChange={e => setParam('spectralColors', { ...params.spectralColors, [type]: e.target.value })}
              className="w-8 h-5 cursor-pointer rounded border-0 bg-transparent"
            />
          </label>
        ))}
      </Section>

      {/* Layer toggles */}
      <Section title="Layer – Systemansicht" onReset={() => resetSection('layers')}>
        <Toggle label="Protuberanzen"    checked={params.layerProminences}      onChange={v => setParam('layerProminences', v)} />
        <Toggle label="Orbitalbahnen"    checked={params.layerOrbits}           onChange={v => setParam('layerOrbits', v)} />
        <Toggle label="Rotationsachse"   checked={params.layerAxisInfo}         onChange={v => setParam('layerAxisInfo', v)} />
        <Toggle label="Richtungspfeil"   checked={params.layerOrbitalChevron}   onChange={v => setParam('layerOrbitalChevron', v)} />
      </Section>

      {/* System view */}
      <Section title="Systemansicht" onReset={() => resetSection('system')}>
        <div className="text-[9px] text-slate-600 uppercase tracking-widest pt-1">Planeten</div>
        <Slider {...PARAM_RANGES.planetVisMax!} value={params.planetVisMax} onChange={v => setParam('planetVisMax', v)} />
        <Slider {...PARAM_RANGES.planetVisMin!} value={params.planetVisMin} onChange={v => setParam('planetVisMin', v)} />
        <div className="text-[9px] text-slate-600 uppercase tracking-widest pt-1">Monde</div>
        <Slider {...PARAM_RANGES.moonSizeFactor!} value={params.moonSizeFactor} onChange={v => setParam('moonSizeFactor', v)} />
        <Slider {...PARAM_RANGES.moonOrbitMin!}   value={params.moonOrbitMin}   onChange={v => setParam('moonOrbitMin', v)} />
        <Slider {...PARAM_RANGES.moonOrbitMax!}   value={params.moonOrbitMax}   onChange={v => setParam('moonOrbitMax', v)} />
      </Section>
    </div>
  )
}
