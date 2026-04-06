// Shared UI primitives for Economy2 components.
import type { ReactNode } from 'react'

export function Spinner() {
  return (
    <div className="w-4 h-4 border-2 border-slate-700 border-t-emerald-400 rounded-full animate-spin" />
  )
}

export function SectionTitle({ children }: { children: ReactNode }) {
  return (
    <h3 className="text-xs tracking-widest text-slate-500 uppercase mb-2">{children}</h3>
  )
}

export function Card({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <div className={`bg-slate-900/50 border border-slate-800 rounded px-3 py-2 mb-1.5 ${className}`}>
      {children}
    </div>
  )
}

export function PrimaryButton({ onClick, children, disabled }: {
  onClick: () => void
  children: ReactNode
  disabled?: boolean
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="text-xs px-2 py-0.5 rounded border border-emerald-700 text-emerald-400
                 hover:bg-emerald-900/30 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
    >
      {children}
    </button>
  )
}

export function DangerButton({ onClick, children, disabled }: {
  onClick: () => void
  children: ReactNode
  disabled?: boolean
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="text-xs px-2 py-0.5 rounded border border-red-800 text-red-400
                 hover:bg-red-900/30 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
    >
      {children}
    </button>
  )
}

export function GhostButton({ onClick, children, disabled }: {
  onClick: () => void
  children: ReactNode
  disabled?: boolean
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="text-xs px-2 py-0.5 rounded border border-slate-700 text-slate-400
                 hover:border-slate-500 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
    >
      {children}
    </button>
  )
}

export function StatusBadge({ label, color }: { label: string; color: 'green' | 'yellow' | 'red' | 'blue' | 'slate' | 'cyan' }) {
  const cls: Record<typeof color, string> = {
    green:  'bg-emerald-900/60 text-emerald-400 border-emerald-800',
    yellow: 'bg-yellow-900/60 text-yellow-400 border-yellow-800',
    red:    'bg-red-900/60 text-red-400 border-red-800',
    blue:   'bg-blue-900/60 text-blue-400 border-blue-800',
    slate:  'bg-slate-800 text-slate-400 border-slate-700',
    cyan:   'bg-cyan-900/60 text-cyan-400 border-cyan-800',
  }
  return (
    <span className={`text-xs px-1.5 py-0.5 rounded border font-medium ${cls[color]}`}>
      {label}
    </span>
  )
}

export function StatusLamp({ status }: { status: string }) {
  const cls: Record<string, string> = {
    running:         'bg-emerald-500',
    paused_depleted: 'bg-orange-500',
    paused_input:    'bg-yellow-500',
    idle:            'bg-slate-600',
    building:        'bg-blue-500',
    destroyed:       'bg-red-700',
  }
  return (
    <span
      className={`inline-block w-2 h-2 rounded-full flex-shrink-0 ${cls[status] ?? 'bg-slate-700'}`}
      title={status}
    />
  )
}

export const ITEM_LABELS: Record<string, string> = {
  steel: 'Stahl', titansteel: 'Titanstahl',
  semiconductor_wafer: 'Halbleiter-Wafer', fusion_fuel: 'Fusionskraftstoff',
  base_component: 'Basisbauteil', nav_computer: 'Navigationscomputer',
  iron: 'Eisen', silicon: 'Silizium', titanium: 'Titan',
  rare_earth: 'Seltene Erden', helium_3: 'Helium-3', uranium: 'Uran',
  nickel: 'Nickel', molybdenum: 'Molybdän', aluminum: 'Aluminium', carbon: 'Kohlenstoff',
}

export function itemLabel(id: string): string {
  return ITEM_LABELS[id] ?? id
}

export const FACTORY_TYPE_LABELS: Record<string, string> = {
  extractor: 'Extraktor', refinery: 'Raffinerie', plant: 'Werk',
  assembly_plant: 'Montagehalle', construction_yard: 'Werft', construction: 'Bau',
}

export function factoryLabel(type: string): string {
  return FACTORY_TYPE_LABELS[type] ?? type
}
