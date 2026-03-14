import { useEffect, useState } from 'react'
import { fetchMorphologies, fetchDefaultParams, postGenerate } from '../api/generate'
import { MorphologyPicker } from '../components/MorphologyPicker'
import { ParamEditor } from '../components/ParamEditor'
import { GenerateJobPanel } from '../components/GenerateJobPanel'
import type { MorphologyTemplate, GameParams, GenerateJob } from '../types/generator'

interface Props {
  onViewGalaxy: (galaxyId: string) => void
}

export function GeneratorPage({ onViewGalaxy }: Props) {
  const [morphologies, setMorphologies] = useState<MorphologyTemplate[]>([])
  const [params, setParams]             = useState<GameParams | null>(null)
  const [selectedMorphology, setSelectedMorphology] = useState('')
  const [galaxyName, setGalaxyName]     = useState('')
  const [job, setJob]                   = useState<GenerateJob | null>(null)
  const [submitting, setSubmitting]     = useState(false)
  const [loadError, setLoadError]       = useState('')

  useEffect(() => {
    Promise.all([fetchMorphologies(), fetchDefaultParams()])
      .then(([morphs, defaults]) => {
        setMorphologies(morphs)
        setParams(defaults)
        setGalaxyName(defaults.server.instance_name || 'Neue Galaxie')
        if (morphs.length > 0) setSelectedMorphology(morphs[0].id)
      })
      .catch(e => setLoadError(String(e)))
  }, [])

  async function handleGenerate() {
    if (!params) return
    setSubmitting(true)
    try {
      const newJob = await postGenerate({
        ...params,
        name: galaxyName,
        morphology_id: selectedMorphology,
      })
      setJob(newJob)
    } catch (e) {
      setLoadError(String(e))
    } finally {
      setSubmitting(false)
    }
  }

  if (loadError) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-red-400 text-sm border border-red-800/40 bg-red-900/20 rounded px-4 py-3">
          {loadError}
        </div>
      </div>
    )
  }

  if (!params) {
    return (
      <div className="flex items-center justify-center h-full gap-3 text-slate-500">
        <div className="w-5 h-5 border-2 border-slate-700 border-t-blue-500 rounded-full animate-spin" />
        <span className="text-sm">Lade Parameter…</span>
      </div>
    )
  }

  const isRunning = job?.status === 'pending' || job?.status === 'running'

  return (
    <div className="flex h-full overflow-hidden">

      {/* ── Links: Morphologie-Auswahl ── */}
      <div className="w-72 shrink-0 border-r border-slate-800 overflow-y-auto p-4 flex flex-col gap-4">
        <MorphologyPicker
          templates={morphologies}
          selected={selectedMorphology}
          onSelect={setSelectedMorphology}
        />
      </div>

      {/* ── Mitte: Parameter + Generierung ── */}
      <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
        <ParamEditor params={params} onChange={setParams} />
      </div>

      {/* ── Rechts: Name, Starten, Status ── */}
      <div className="w-64 shrink-0 border-l border-slate-800 p-4 flex flex-col gap-4">
        <div className="flex flex-col gap-1">
          <label className="text-[10px] text-slate-400 uppercase tracking-widest">Name der Galaxie</label>
          <input
            type="text"
            value={galaxyName}
            onChange={e => setGalaxyName(e.target.value)}
            className="bg-slate-900 border border-slate-700 rounded px-2 py-1.5 text-sm text-slate-200
                       focus:outline-none focus:border-blue-500"
          />
        </div>

        {selectedMorphology && (
          <div className="text-xs text-slate-500">
            Morphologie: <span className="text-slate-300">
              {morphologies.find(m => m.id === selectedMorphology)?.hubble_type ?? '—'}
              {' '}({morphologies.find(m => m.id === selectedMorphology)?.name ?? selectedMorphology})
            </span>
          </div>
        )}

        <button
          onClick={handleGenerate}
          disabled={submitting || isRunning || !galaxyName.trim()}
          className={`px-4 py-2.5 rounded text-sm font-semibold transition-colors
            ${submitting || isRunning || !galaxyName.trim()
              ? 'bg-slate-800 text-slate-600 cursor-not-allowed'
              : 'bg-blue-600 hover:bg-blue-500 text-white'}`}
        >
          {submitting ? 'Startet…' : isRunning ? 'Läuft…' : 'Galaxie generieren'}
        </button>

        <GenerateJobPanel
          job={job}
          onDone={id => {
            setTimeout(() => onViewGalaxy(id), 800)
          }}
        />

        {/* Hinweis */}
        <div className="mt-auto text-[10px] text-slate-700 leading-relaxed">
          Die Generierung läuft im Hintergrund. Bei 50.000 Sternen ca. 1–3 Min.
          Der Viewer wechselt automatisch nach Abschluss.
        </div>
      </div>

    </div>
  )
}
