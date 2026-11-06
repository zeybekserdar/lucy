import type { Dashboard, Monitor, MonitorStatus, Network } from '../../api/types'

const ORDER: Record<MonitorStatus, number> = { down: 0, warn: 1, unknown: 2, up: 3 }
const EXPIRY_WARN_DAYS = 14
const isExpiring = (days: number | null) => days !== null && days <= EXPIRY_WARN_DAYS

const secondsSince = (iso: string, now: number) => (now - new Date(iso).getTime()) / 1000

export function sortMonitors(monitors: Monitor[]): Monitor[] {
  return [...monitors].sort(
    (a, b) => ORDER[a.status] - ORDER[b.status] || a.name.localeCompare(b.name),
  )
}

export function groupMonitors(monitors: Monitor[]) {
  const sorted = sortMonitors(monitors)
  return {
    downs: sorted.filter((m) => m.status === 'down'),
    softs: sorted.filter((m) => m.status === 'warn' || m.status === 'unknown'),
    healthy: sorted.filter((m) => m.status === 'up'),
  }
}

export type Hero =
  | { kind: 'ok'; total: number; lastCheckSec: number }
  | { kind: 'down'; names: string[]; durationSec: number; others: number; lastCheckSec: number }
  | { kind: 'offline'; durationSec: number }
  | { kind: 'warn' | 'unknown'; count: number; total: number; others: number; lastCheckSec: number }

export function deriveHero({ monitors, network }: Dashboard, now: number): Hero {
  const total = monitors.length
  const checks = monitors.flatMap((m) => (m.lastCheck ? [new Date(m.lastCheck).getTime()] : []))
  const lastCheckSec = checks.length ? (now - Math.max(...checks)) / 1000 : 0
  const { downs, softs } = groupMonitors(monitors)

  if (downs.length) {
    return {
      kind: 'down',
      names: downs.map((m) => m.name),
      durationSec: Math.max(...downs.map((m) => (m.since ? secondsSince(m.since, now) : 0))),
      others: total - downs.length,
      lastCheckSec,
    }
  }
  if (!network.online) {
    return {
      kind: 'offline',
      durationSec: network.offlineSince ? secondsSince(network.offlineSince, now) : 0,
    }
  }
  const warns = softs.filter((m) => m.status === 'warn').length
  if (warns) return { kind: 'warn', count: warns, total, others: total - softs.length, lastCheckSec }
  if (softs.length) {
    return { kind: 'unknown', count: softs.length, total, others: total - softs.length, lastCheckSec }
  }
  return { kind: 'ok', total, lastCheckSec }
}

export type SoftReason =
  | { key: 'focus.sslExpiring' | 'focus.domainExpiring'; count: number }
  | { key: 'focus.sslExpired' | 'focus.domainExpired' | 'focus.warning' | 'focus.unknown' | 'focus.unknownOffline' }

export function softReason(monitor: Monitor, network: Network): SoftReason {
  if (monitor.status === 'unknown') {
    return { key: network.online ? 'focus.unknown' : 'focus.unknownOffline' }
  }
  const { ssl, domain } = monitor
  if (isExpiring(ssl.daysLeft)) {
    const count = ssl.daysLeft!
    return count <= 0 ? { key: 'focus.sslExpired' } : { key: 'focus.sslExpiring', count }
  }
  if (isExpiring(domain.daysLeft)) {
    const count = domain.daysLeft!
    return count <= 0 ? { key: 'focus.domainExpired' } : { key: 'focus.domainExpiring', count }
  }
  return { key: 'focus.warning' }
}

/** Yakın tarihli (bugün veya dün) kesinti, satırdaki notta gösterilir. */
export function recentIncident(monitor: Monitor, now: number) {
  const incident = monitor.lastIncident
  if (!incident) return null
  return secondsSince(incident.endedAt, now) < 2 * 86_400 ? incident : null
}
