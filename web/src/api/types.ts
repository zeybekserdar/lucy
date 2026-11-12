export type MonitorStatus = 'up' | 'warn' | 'down' | 'unknown'
export type DayState = 'ok' | 'partial' | 'down' | 'unknown' | 'no_data'

export interface Expiry {
  daysLeft: number | null
  expiresAt: string | null
  issuer?: string
  reason?: string
}

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'HEAD'
export type ConditionType = 'status' | 'response' | 'body' | 'json'

export interface Condition {
  type: ConditionType
  op: string
  path?: string
  value: string
}

export interface MonitorConfig {
  id?: string
  name: string
  url: string
  method: HttpMethod
  headers: { name: string; value: string }[]
  body: string
  timeoutSec: number
  intervalSec: number
  alertThreshold: number
  conditions: Condition[]
  /** Kesinti bildirimi telefonun sessiz modunu deler. */
  urgent?: boolean
}

export interface Monitor extends MonitorConfig {
  id: string
  status: MonitorStatus
  since: string | null
  lastCheck: string | null
  responseMs: number | null
  /** Veri yoksa null. */
  uptime: { d1: number | null; d7: number | null; d30: number | null }
  days: DayState[]
  halfHours: DayState[]
  ssl: Expiry
  domain: Expiry
  /** Dilden bağımsız neden: http:503 Service Unavailable, net:timeout, cond:body */
  failure?: string
  notifiedAt?: string
  lastIncident?: { endedAt: string; durationSec: number }
}

export interface ConditionResult {
  type: ConditionType
  op: string
  path?: string
  ok: boolean
  expected: string
  actual: string
  reason?: 'no_response' | 'not_json' | 'path_missing'
}

export interface CheckResult {
  ok: boolean
  method: HttpMethod
  url: string
  status: number
  statusText: string
  responseMs: number
  contentType: string
  error?: string
  conditions: ConditionResult[]
  at: string
}

export type ChartRange = '24h' | '7d' | '30d'

export interface HistoryEvent {
  kind: 'outage' | 'no_data'
  start: string
  /** Sürüyorsa null. */
  end: string | null
  cause?: string
}

export interface History {
  monitor: Monitor
  from: string
  to: string
  /** from ile to arasına eşit aralıkla dizilmiş ortalamalar; başarılı kontrol yoksa null. */
  points: (number | null)[]
  avgMs: number | null
  /** En yenisi başta, seçili aralıktan bağımsız olarak son 30 gün. */
  events: HistoryEvent[]
  failedChecks: number
  lastResult: CheckResult | null
}

export interface NtfySettings {
  server: string
  topic: string
  token: string
  /** Bildirime dokununca telefonda açılacak adres; boş olabilir. */
  publicUrl: string
}

export interface NtfyTestResult {
  ok: boolean
  /** Sunucu hiç yanıt vermediyse 0. */
  status: number
}

export interface ImportPreview {
  add: string[]
  update: string[]
  unchanged: number
  invalid: { name: string; fields: Record<string, string> }[]
  notifications: boolean
}

export interface AuthStatus {
  status: 'needsSetup' | 'loggedOut' | 'loggedIn'
  passwordChangedAt?: string
}

export interface Network {
  online: boolean
  offlineSince?: string
}

export interface Dashboard {
  monitors: Monitor[]
  network: Network
}
