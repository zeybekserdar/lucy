import type { TFunction } from 'i18next'
import type { ConditionResult, Expiry, Monitor, MonitorStatus } from '../../api/types'
import { formatDate } from '../../lib/format'

export function conditionName(c: ConditionResult, t: TFunction): string {
  return c.type === 'json' ? t('detail.jsonPath', { path: c.path }) : t(`editor.type.${c.type}`)
}

export function conditionExpected(c: ConditionResult, t: TFunction): string {
  const value = c.expected
  if (c.type === 'status') return value
  if (c.type === 'response') return t('detail.expect.below', { value })
  return t(`detail.expect.${c.op}`, { value, defaultValue: value })
}

export function conditionActual(c: ConditionResult, t: TFunction): string {
  if (c.reason === 'no_response') return t('detail.noResponse')
  if (c.reason === 'not_json') return t('detail.notJson')
  if (c.reason === 'path_missing') return t('detail.pathMissing')
  if (c.type === 'response') return `${c.actual} ms`
  if (c.type === 'body') return t(c.actual === 'true' ? 'detail.found' : 'detail.notFound')
  return c.actual
}

export interface ExpiryLine {
  name: string
  sub: string
  value: string
  status: MonitorStatus
}

function known(expiry: Expiry, days: number, t: TFunction, lng: string): Omit<ExpiryLine, 'name'> {
  const date = t('detail.expiresOn', { date: formatDate(expiry.expiresAt!, lng) })
  const sub = expiry.issuer ? `${date} · ${expiry.issuer}` : date
  if (days < 0) return { sub, value: t('detail.expired'), status: 'down' }
  return { sub, value: t('detail.days', { count: days }), status: days <= 14 ? 'warn' : 'up' }
}

export function expiryLines(monitor: Monitor, t: TFunction, lng: string): ExpiryLine[] {
  const url = new URL(monitor.url)
  const lines: ExpiryLine[] = []
  const unknown = { value: t('status.unknown'), status: 'unknown' as const }
  if (url.protocol === 'https:') {
    const { ssl } = monitor
    lines.push({
      name: t('detail.ssl'),
      ...(ssl.daysLeft === null ? { ...unknown, sub: t('detail.sslUnknown') } : known(ssl, ssl.daysLeft, t, lng)),
    })
  }
  const { domain } = monitor
  const reason = t(`detail.domainReason.${domain.reason || 'pending'}`, { host: url.hostname })
  lines.push({
    name: t('detail.domain'),
    ...(domain.daysLeft === null ? { ...unknown, sub: reason } : known(domain, domain.daysLeft, t, lng)),
  })
  return lines
}
