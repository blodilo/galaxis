/**
 * CicOverlay — Wurzel-Container für alle DOM-HUD-Elemente im CIC.
 *
 * Strategie (ADR-012):
 *  - Das Overlay liegt absolut über dem Three.js-Canvas (inset-0, z-10).
 *  - pointer-events: none als Default → Canvas-Klicks und Raycaster bleiben unberührt.
 *  - Interaktive Kind-Elemente setzen pointer-events: auto lokal auf sich selbst.
 */

interface CicOverlayProps {
  children?: React.ReactNode
}

export function CicOverlay({ children }: CicOverlayProps) {
  return (
    <div
      className="absolute inset-0 z-10"
      style={{ pointerEvents: 'none' }}
    >
      {children}
    </div>
  )
}
