import { useEffect, useState, useRef } from 'react'
import { fetchMorphologies, fetchDefaultParams, postGenerateStep1, postGalaxyStep, deleteGalaxy, fetchJobStatus, openProgressStream } from '../api/generate'
import { MorphologyPicker } from '../components/MorphologyPicker'
import { ParamEditor } from '../components/ParamEditor'
import type { MorphologyTemplate, GameParams, GenerateJob } from '../types/generator'
import type { ProgressEvent } from '../api/generate'
import type { Galaxy } from '../types/galaxy'

interface Props {
  onViewGalaxy: (galaxyId: string) => void
  onGalaxiesChanged?: (completedGalaxyId?: string) => void
  resumeGalaxy?: Galaxy | null
}

function statusToSteps(status: string): number {
  switch (status) {
    case 'morphology': return 1
    case 'spectral':   return 2
    case 'objects':    return 3
    case 'ready':      return 4
    default:           return 0
  }
}

type Phase = 'config' | 'stepping'

const STEP_LABELS = [
  'Morphologie',
  'Spektralklassen',
  'Sonstige Objekte',
  'Planetensysteme',
]

export function GeneratorPage({ onViewGalaxy, onGalaxiesChanged, resumeGalaxy }: Props) {
  const [morphologies, setMorphologies] = useState<MorphologyTemplate[]>([])
  const [params, setParams]             = useState<GameParams | null>(null)
  const [selectedMorphology, setSelectedMorphology] = useState('')
  const [galaxyName, setGalaxyName]     = useState('')
  const [loadError, setLoadError]       = useState('')

  // Workflow state
  const [phase, setPhase]               = useState<Phase>('config')
  const [galaxyId, setGalaxyId]         = useState<string | null>(null)
  const [completedSteps, setCompletedSteps] = useState(0) // 0–4
  const [stepJob, setStepJob]           = useState<GenerateJob | null>(null)
  const [stepRunning, setStepRunning]   = useState(false)
  const [stepError, setStepError]       = useState('')
  const [discarding, setDiscarding]     = useState(false)
  const [progress, setProgress]         = useState<ProgressEvent | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const closeStreamRef = useRef<(() => void) | null>(null)

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

  // Job polling
  useEffect(() => {
    if (!stepJob || stepJob.status === 'done' || stepJob.status === 'error') {
      if (intervalRef.current) clearInterval(intervalRef.current)
      return
    }
    intervalRef.current = setInterval(async () => {
      try {
        const updated = await fetchJobStatus(stepJob.job_id)
        setStepJob(updated)
        if (updated.status === 'done') {
          clearInterval(intervalRef.current!)
          setStepRunning(false)
          if (updated.galaxy_id && !galaxyId) {
            setGalaxyId(updated.galaxy_id)
          }
          setCompletedSteps(s => s + 1)
          onGalaxiesChanged?.(updated.galaxy_id ?? galaxyId ?? undefined)
        }
        if (updated.status === 'error') {
          clearInterval(intervalRef.current!)
          setStepRunning(false)
          setStepError(updated.error || 'Fehler beim Ausführen')
        }
      } catch { /* ignore transient errors */ }
    }, 2000)
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [stepJob?.job_id, stepJob?.status])

  // Resume an in-progress galaxy from the galaxy picker
  useEffect(() => {
    if (!resumeGalaxy) return
    closeStreamRef.current?.()
    closeStreamRef.current = null
    if (intervalRef.current) clearInterval(intervalRef.current)
    setPhase('stepping')
    setGalaxyId(resumeGalaxy.id)
    setGalaxyName(resumeGalaxy.name)
    setCompletedSteps(statusToSteps(resumeGalaxy.status))
    setStepJob(null)
    setStepRunning(false)
    setStepError('')
    setProgress(null)
  }, [resumeGalaxy?.id])

  // SSE progress stream — open when a new job starts, close when done/error.
  useEffect(() => {
    closeStreamRef.current?.()
    closeStreamRef.current = null
    if (!stepJob || !['pending', 'running'].includes(stepJob.status)) {
      setProgress(null)
      return
    }
    const close = openProgressStream(
      stepJob.job_id,
      (ev) => setProgress(ev),
      () => setProgress(null),
    )
    closeStreamRef.current = close
    return close
  }, [stepJob?.job_id])

  async function handleStartMorphology() {
    if (!params) return
    setStepError('')
    setStepRunning(true)
    try {
      const job = await postGenerateStep1({
        ...params,
        name: galaxyName,
        morphology_id: selectedMorphology,
      })
      setStepJob(job)
      setPhase('stepping')
    } catch (e) {
      setStepRunning(false)
      setStepError(String(e))
    }
  }

  async function handleRunStep(stepIndex: number) {
    if (!galaxyId) return
    const stepNames = ['spectral', 'objects', 'planets'] as const
    const stepName = stepNames[stepIndex - 2]
    setStepError('')
    setStepRunning(true)
    try {
      const job = await postGalaxyStep(galaxyId, stepName)
      setStepJob(job)
    } catch (e) {
      setStepRunning(false)
      setStepError(String(e))
    }
  }

  async function handleDiscard() {
    if (!galaxyId) {
      setPhase('config')
      setCompletedSteps(0)
      setStepJob(null)
      setStepRunning(false)
      setStepError('')
      setGalaxyId(null)
      return
    }
    setDiscarding(true)
    closeStreamRef.current?.()
    closeStreamRef.current = null
    try {
      await deleteGalaxy(galaxyId)
    } catch { /* ignore */ }
    setPhase('config')
    setCompletedSteps(0)
    setStepJob(() => { if (intervalRef.current) clearInterval(intervalRef.current); return null })
    setStepRunning(false)
    setStepError('')
    setGalaxyId(null)
    setProgress(null)
    setDiscarding(false)
    onGalaxiesChanged?.()
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

  return (
    <div className="flex h-full overflow-hidden">

      {/* ── Links: Morphologie-Auswahl (nur in Config-Phase) ── */}
      <div className="w-72 shrink-0 border-r border-slate-800 overflow-y-auto p-4 flex flex-col gap-4">
        {phase === 'config' && (
          <MorphologyPicker
            templates={morphologies}
            selected={selectedMorphology}
            onSelect={setSelectedMorphology}
          />
        )}
      </div>

      {/* ── Mitte: Parameter (nur in Config-Phase) ── */}
      <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
        {phase === 'config' && <ParamEditor params={params} onChange={setParams} />}
      </div>

      {/* ── Rechts: Workflow ── */}
      <div className="w-64 shrink-0 border-l border-slate-800 p-4 flex flex-col gap-4">

        {phase === 'config' && (
          <>
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

            {stepError && (
              <div className="text-xs text-red-400 bg-red-900/20 border border-red-800/40 rounded px-2 py-1">
                {stepError}
              </div>
            )}

            <button
              onClick={handleStartMorphology}
              disabled={stepRunning || !galaxyName.trim()}
              className={`px-4 py-2.5 rounded text-sm font-semibold transition-colors
                ${stepRunning || !galaxyName.trim()
                  ? 'bg-slate-800 text-slate-600 cursor-not-allowed'
                  : 'bg-blue-600 hover:bg-blue-500 text-white'}`}
            >
              {stepRunning ? 'Startet…' : 'Morphologie generieren →'}
            </button>

            <div className="mt-auto text-[10px] text-slate-700 leading-relaxed">
              Die Generierung läuft schrittweise. Jeder Schritt kann einzeln inspiziert und verworfen werden.
            </div>
          </>
        )}

        {phase === 'stepping' && (
          <>
            <div className="flex flex-col gap-1">
              <div className="text-[10px] text-slate-400 uppercase tracking-widest">Galaxie</div>
              <div className="text-sm text-slate-200 font-semibold">{galaxyName}</div>
            </div>

            {stepError && (
              <div className="text-xs text-red-400 bg-red-900/20 border border-red-800/40 rounded px-2 py-1">
                {stepError}
              </div>
            )}

            {/* Step list */}
            <div className="flex flex-col gap-3">
              {STEP_LABELS.map((label, i) => {
                const stepNum = i + 1
                const isDone = completedSteps >= stepNum
                const isRunning = stepRunning && completedSteps === stepNum - 1
                const isNext = !stepRunning && completedSteps === stepNum - 1
                const isLocked = completedSteps < stepNum - 1

                return (
                  <div key={label} className={`border rounded p-2.5 flex flex-col gap-1.5
                    ${isDone ? 'border-slate-700 bg-slate-900/30' : 'border-slate-800'}`}>
                    <div className="flex items-center gap-2">
                      <span className={`w-4 h-4 rounded-full text-[9px] flex items-center justify-center font-bold shrink-0
                        ${isDone ? 'bg-emerald-700 text-emerald-200' : isRunning ? 'bg-blue-700 text-blue-200' : 'bg-slate-800 text-slate-500'}`}>
                        {isDone ? '✓' : stepNum}
                      </span>
                      <span className={`text-xs font-medium ${isDone ? 'text-slate-300' : isRunning ? 'text-blue-300' : isLocked ? 'text-slate-600' : 'text-slate-300'}`}>
                        {label}
                      </span>
                        {isRunning && (
                        <div className="w-3 h-3 border border-slate-600 border-t-blue-400 rounded-full animate-spin ml-auto shrink-0" />
                      )}
                    </div>

                    {isRunning && progress && progress.total > 0 && (
                      <div className="flex flex-col gap-0.5">
                        <div className="h-1 bg-slate-800 rounded overflow-hidden">
                          <div
                            className="h-full bg-blue-500 transition-all duration-300"
                            style={{ width: `${Math.round((progress.done / progress.total) * 100)}%` }}
                          />
                        </div>
                        <div className="text-[9px] text-slate-500 tabular-nums">
                          {progress.done.toLocaleString('de-DE')} / {progress.total.toLocaleString('de-DE')}
                          {progress.msg ? ` · ${progress.msg}` : ''}
                        </div>
                      </div>
                    )}

                    {isDone && galaxyId && (
                      <button
                        onClick={() => onViewGalaxy(galaxyId)}
                        className="text-[10px] text-cyan-500 hover:text-cyan-300 transition-colors text-left"
                      >
                        Im Viewer anzeigen →
                      </button>
                    )}

                    {isNext && stepNum > 1 && (
                      <button
                        onClick={() => handleRunStep(stepNum)}
                        className="mt-0.5 w-full text-xs py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-200 transition-colors"
                      >
                        ▶ Ausführen
                      </button>
                    )}
                  </div>
                )
              })}
            </div>

            {completedSteps === 4 && galaxyId && (
              <button
                onClick={() => onViewGalaxy(galaxyId)}
                className="px-4 py-2.5 rounded text-sm font-semibold bg-emerald-700 hover:bg-emerald-600 text-white transition-colors"
              >
                Fertig → Viewer
              </button>
            )}

            <button
              onClick={handleDiscard}
              disabled={discarding || stepRunning}
              className="mt-auto px-3 py-1.5 rounded text-xs border border-red-900 text-red-500 hover:bg-red-900/20 transition-colors disabled:opacity-40"
            >
              {discarding ? 'Wird gelöscht…' : 'Verwerfen'}
            </button>
          </>
        )}
      </div>
    </div>
  )
}
