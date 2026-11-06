import type { TFunction } from 'i18next'

export function formatUptime(value: number | null, lng: string): string {
  if (value === null) return '—'
  const digits = Number.isInteger(value) ? 0 : 2
  const number = new Intl.NumberFormat(lng, {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value)
  return `${number}%`
}

export function formatMs(ms: number | null): string {
  return ms === null ? '—' : `${ms} ms`
}

/** Sunucunun dilden bağımsız hata kodunu okunur metne çevirir. */
export function describeFailure(failure: string, t: TFunction): string {
  const [kind, ...rest] = failure.split(':')
  const detail = rest.join(':')
  if (kind === 'http') return `HTTP ${detail}`
  return t(`failure.${kind}.${detail}`, { defaultValue: t('failure.net.network') })
}

export function formatDuration(totalSec: number, t: TFunction): string {
  const sec = Math.max(0, Math.round(totalSec))
  if (sec < 60) return t('unit.s', { n: sec })
  const minutes = Math.floor(sec / 60)
  if (minutes < 60) return t('unit.m', { n: minutes })
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    const rest = minutes % 60
    return rest ? `${t('unit.h', { n: hours })} ${t('unit.m', { n: rest })}` : t('unit.h', { n: hours })
  }
  const days = Math.floor(hours / 24)
  const rest = hours % 24
  return rest ? `${t('unit.d', { n: days })} ${t('unit.h', { n: rest })}` : t('unit.d', { n: days })
}

export function formatAgo(sec: number, lng: string, style: 'long' | 'narrow'): string {
  const rtf = new Intl.RelativeTimeFormat(lng, { style })
  const value = Math.max(0, Math.round(sec))
  const text = (() => {
    if (value < 60) return rtf.format(-value, 'second')
    if (value < 3600) return rtf.format(-Math.floor(value / 60), 'minute')
    if (value < 86400) return rtf.format(-Math.floor(value / 3600), 'hour')
    if (value < 30 * 86400) return rtf.format(-Math.floor(value / 86400), 'day')
    return rtf.format(-Math.floor(value / (30 * 86400)), 'month')
  })()
  // Tasarım "8 sn önce" yazıyor, Intl ise "8 sn. önce" üretiyor.
  return style === 'narrow' ? text.replace(/\.(?= )/, '') : text
}

export function formatClock(iso: string, lng: string): string {
  return new Intl.DateTimeFormat(lng, { hour: '2-digit', minute: '2-digit' }).format(new Date(iso))
}

export function formatRelativeDay(iso: string, now: number, lng: string): string {
  const startOfDay = (t: number) => new Date(t).setHours(0, 0, 0, 0)
  const days = Math.round((startOfDay(new Date(iso).getTime()) - startOfDay(now)) / 86_400_000)
  const text = new Intl.RelativeTimeFormat(lng, { numeric: 'auto' }).format(days, 'day')
  return text.charAt(0).toLocaleUpperCase(lng) + text.slice(1)
}
