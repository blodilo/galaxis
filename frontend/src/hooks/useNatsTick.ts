// useNatsTick — subscribes to galaxis.tick.advance via NATS.
// Calls onTick(tickN) on each server tick.
// Falls back gracefully if NATS is unavailable.

import { useEffect, useRef, useState } from 'react'
import { getNATS, jc } from '../lib/nats'

export type NatsStatus = 'connecting' | 'live' | 'offline'

export function useNatsTick(onTick: (tickN: number) => void): NatsStatus {
  const [status, setStatus] = useState<NatsStatus>('connecting')
  const onTickRef = useRef(onTick)
  onTickRef.current = onTick

  useEffect(() => {
    let cancelled = false

    getNATS()
      .then(nc => {
        if (cancelled) return

        setStatus('live')
        const sub = nc.subscribe('galaxis.tick.advance')

        ;(async () => {
          for await (const msg of sub) {
            if (cancelled) break
            try {
              const data = jc.decode(msg.data) as { tick: number }
              onTickRef.current(data.tick)
            } catch {
              // malformed message — ignore
            }
          }
        })()

        return () => { sub.unsubscribe() }
      })
      .catch(() => {
        if (!cancelled) setStatus('offline')
      })

    return () => { cancelled = true }
  }, []) // deliberately empty — singleton connection, stable subscription

  return status
}
