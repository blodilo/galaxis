// Galaxis NATS browser client — nats.ws (Apache-2.0)
// Singleton connection: connected lazily on first use.

import { connect, JSONCodec } from 'nats.ws'
import type { NatsConnection, Subscription } from 'nats.ws'

export const jc = JSONCodec()

let nc: NatsConnection | null = null
let connecting: Promise<NatsConnection> | null = null

interface NatsConfig {
  url: string
  auth_required: boolean
  expires_in: number
}

async function fetchConfig(): Promise<NatsConfig> {
  const res = await fetch('/api/v1/auth/nats-token', { method: 'POST' })
  if (!res.ok) throw new Error(`nats-token: ${res.status}`)
  return res.json()
}

export async function getNATS(): Promise<NatsConnection> {
  if (nc && !nc.isClosed()) return nc
  if (connecting) return connecting

  connecting = (async () => {
    const cfg = await fetchConfig()
    const conn = await connect({
      servers: cfg.url,
      reconnect: true,
      maxReconnectAttempts: -1,
      reconnectTimeWait: 2000,
    })
    nc = conn
    connecting = null

    // Cleanup on close
    conn.closed().then(() => {
      if (nc === conn) nc = null
    })

    return conn
  })()

  return connecting
}

export async function subscribeOnce(
  subject: string,
  handler: (data: unknown) => void,
): Promise<Subscription> {
  const conn = await getNATS()
  const sub = conn.subscribe(subject)
  ;(async () => {
    for await (const msg of sub) {
      try {
        handler(jc.decode(msg.data))
      } catch {
        handler(null)
      }
    }
  })()
  return sub
}

export function isConnected(): boolean {
  return nc !== null && !nc.isClosed()
}
