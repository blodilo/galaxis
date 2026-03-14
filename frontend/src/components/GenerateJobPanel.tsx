import { useEffect, useRef, useState } from 'react'
import { fetchJobStatus } from '../api/generate'
import type { GenerateJob, JobStatus } from '../types/generator'

interface Props {
  job: GenerateJob | null
  onDone: (galaxyId: string) => void
}

const STATUS_LABEL: Record<JobStatus, string> = {
  pending: 'Warteschlange…',
  running: 'Generierung läuft…',
  done:    'Fertig',
  error:   'Fehler',
}

const STATUS_COLOR: Record<JobStatus, string> = {
  pending: 'text-slate-400',
  running: 'text-blue-400',
  done:    'text-green-400',
  error:   'text-red-400',
}

export function GenerateJobPanel({ job, onDone }: Props) {
  const [current, setCurrent] = useState<GenerateJob | null>(job)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Sync new job prop from outside
  useEffect(() => {
    setCurrent(job)
  }, [job])

  // Poll while pending or running
  useEffect(() => {
    if (!current || current.status === 'done' || current.status === 'error') {
      if (intervalRef.current) clearInterval(intervalRef.current)
      return
    }

    intervalRef.current = setInterval(async () => {
      try {
        const updated = await fetchJobStatus(current.job_id)
        setCurrent(updated)
        if (updated.status === 'done' && updated.galaxy_id) {
          onDone(updated.galaxy_id)
          clearInterval(intervalRef.current!)
        }
        if (updated.status === 'error') {
          clearInterval(intervalRef.current!)
        }
      } catch {
        // ignore transient errors, keep polling
      }
    }, 2000)

    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [current?.job_id, current?.status])

  if (!current) return null

  const isSpinning = current.status === 'pending' || current.status === 'running'
  const elapsed = current.created_at
    ? Math.round((Date.now() - new Date(current.created_at).getTime()) / 1000)
    : 0

  return (
    <div className="border border-slate-700 rounded p-3 flex flex-col gap-2">
      <div className="flex items-center gap-2">
        {isSpinning && (
          <div className="w-4 h-4 border-2 border-slate-600 border-t-blue-400 rounded-full animate-spin shrink-0" />
        )}
        <span className={`text-sm font-medium ${STATUS_COLOR[current.status]}`}>
          {STATUS_LABEL[current.status]}
        </span>
        {isSpinning && (
          <span className="text-xs text-slate-600 ml-auto">{elapsed}s</span>
        )}
      </div>

      {current.error && (
        <div className="text-xs text-red-400 bg-red-900/20 border border-red-800/40 rounded px-2 py-1">
          {current.error}
        </div>
      )}

      {current.status === 'done' && current.galaxy_id && (
        <div className="text-xs text-green-300 bg-green-900/20 border border-green-800/40 rounded px-2 py-1">
          Galaxy ID: <span className="font-mono">{current.galaxy_id}</span>
        </div>
      )}

      <div className="text-[10px] text-slate-700 font-mono">
        Job: {current.job_id.slice(0, 8)}…
      </div>
    </div>
  )
}
