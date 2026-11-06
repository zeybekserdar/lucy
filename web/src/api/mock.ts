import type { Dashboard, DayState, Monitor } from './types'

// Backend gelene kadar geçici veri. ?state=ok|incident|offline|empty|error
export type MockState = 'ok' | 'incident' | 'offline' | 'empty' | 'error'

const MINUTE = 60_000
const DAY = 24 * 60 * MINUTE

interface Seed {
  name: string
  path?: string
  marks?: Record<number, DayState>
  d30: number
  ms: number
  sslDays?: number
  domainDays?: number | null
  lastIncident?: { daysAgo: number; durationSec: number }
}

const seeds: Seed[] = [
  { name: 'api.app', path: '/health', marks: { 27: 'partial' }, d30: 99.86, ms: 142 },
  { name: 'blog.dev', marks: { 12: 'partial' }, d30: 99.98, ms: 182 },
  {
    name: 'foto.home',
    marks: { 21: 'partial', 28: 'down' },
    d30: 99.2,
    ms: 96,
    lastIncident: { daysAgo: 1, durationSec: 4 * 3600 + 12 * 60 },
  },
  { name: 'nas.home', marks: { 9: 'no_data', 10: 'no_data' }, d30: 100, ms: 12 },
  { name: 'vault.home', marks: { 17: 'partial' }, d30: 99.95, ms: 34, domainDays: null },
  { name: 'git.home', path: '/api/healthz', d30: 100, ms: 28 },
  { name: 'media.home', marks: { 4: 'partial' }, d30: 99.97, ms: 61 },
]

const iso = (t: number) => new Date(t).toISOString()

function toMonitor(seed: Seed, now: number): Monitor {
  const domainDays = seed.domainDays === undefined ? 210 : seed.domainDays
  return {
    id: seed.name,
    name: seed.name,
    url: `https://${seed.name}${seed.path ?? ''}`,
    method: 'GET',
    headers: [],
    body: '',
    timeoutSec: 10,
    intervalSec: 60,
    alertThreshold: 3,
    conditions: [],
    status: 'up',
    since: iso(now - 3 * DAY),
    lastCheck: iso(now - 12_000),
    responseMs: seed.ms,
    uptime: { d1: 100, d7: seed.d30, d30: seed.d30 },
    days: Array.from({ length: 30 }, (_, i) => seed.marks?.[i] ?? 'ok'),
    halfHours: Array.from({ length: 48 }, () => 'ok'),
    ssl: { daysLeft: seed.sslDays ?? 62, expiresAt: iso(now + 62 * DAY), issuer: "Let's Encrypt" },
    domain: {
      daysLeft: domainDays,
      expiresAt: domainDays === null ? null : iso(now + domainDays * DAY),
      reason: domainDays === null ? 'unsupported' : undefined,
    },
    lastIncident: seed.lastIncident && {
      endedAt: iso(now - seed.lastIncident.daysAgo * DAY),
      durationSec: seed.lastIncident.durationSec,
    },
  }
}

const startedAt = Date.now()

export function mockDashboard(state: MockState, now: number): Dashboard {
  if (state === 'error') throw new Error('mock: sunucuya ulaşılamadı')
  if (state === 'empty') return { monitors: [], network: { online: true } }

  const monitors = seeds.map((seed) => toMonitor(seed, now))
  const byName = (name: string) => monitors.find((m) => m.name === name)!

  if (state === 'incident') {
    const api = byName('api.app')
    const since = startedAt - 14 * MINUTE
    api.status = 'down'
    api.since = iso(since)
    api.lastCheck = iso(now - 8_000)
    api.responseMs = null
    api.failure = 'http:503 Service Unavailable'
    api.notifiedAt = iso(since + 2 * MINUTE)
    api.uptime = { d1: 99.03, d7: 99.5, d30: 98.71 }
    api.days[29] = 'down'
    api.halfHours[47] = 'down'

    const blog = byName('blog.dev')
    blog.status = 'warn'
    blog.ssl = { ...blog.ssl, daysLeft: 7, expiresAt: iso(now + 7 * DAY) }
  }

  if (state === 'offline') {
    for (const name of ['api.app', 'blog.dev']) {
      const m = byName(name)
      m.status = 'unknown'
      m.responseMs = null
      m.days[29] = 'unknown'
    }
    return { monitors, network: { online: false, offlineSince: iso(startedAt - 6 * MINUTE) } }
  }

  return { monitors, network: { online: true } }
}
