import type { TFunction } from 'i18next'
import { describe, expect, it } from 'vitest'
import type { ConditionResult, Monitor } from '../../api/types'
import { conditionActual, conditionExpected, expiryLines } from './sections'

const t = ((key: string, values?: Record<string, unknown>) =>
  values ? `${key}:${JSON.stringify(values)}` : key) as unknown as TFunction

const condition = (over: Partial<ConditionResult>): ConditionResult => ({
  type: 'status',
  op: 'between',
  ok: true,
  expected: '200–299',
  actual: '200',
  ...over,
})

const monitor = (over: Partial<Monitor>): Monitor =>
  ({
    url: 'https://a.example.com/health',
    ssl: { daysLeft: null, expiresAt: null },
    domain: { daysLeft: null, expiresAt: null },
    ...over,
  }) as Monitor

describe('conditions', () => {
  it('shows the status range as it is', () => {
    expect(conditionExpected(condition({}), t)).toBe('200–299')
  })
  it('explains a body result in words', () => {
    expect(conditionActual(condition({ type: 'body', op: 'contains', actual: 'true' }), t)).toBe('detail.found')
    expect(conditionActual(condition({ type: 'body', op: 'contains', actual: 'false' }), t)).toBe('detail.notFound')
  })
  it('prefers the failure reason over the value', () => {
    expect(conditionActual(condition({ type: 'json', reason: 'path_missing', actual: '' }), t)).toBe('detail.pathMissing')
  })
})

describe('expiryLines', () => {
  const inDays = (days: number) => ({ daysLeft: days, expiresAt: '2027-01-07T00:00:00Z' })

  it('follows the design’s thresholds', () => {
    const status = (days: number) => expiryLines(monitor({ ssl: inDays(days) }), t, 'tr')[0].status
    expect(status(15)).toBe('up')
    expect(status(14)).toBe('warn')
    expect(status(0)).toBe('warn')
    expect(status(-1)).toBe('down')
  })

  it('leaves out the certificate of a plain http monitor', () => {
    const lines = expiryLines(monitor({ url: 'http://a.example.com' }), t, 'tr')
    expect(lines.map((l) => l.name)).toEqual(['detail.domain'])
  })

  it('names the host when a local domain has no record', () => {
    const lines = expiryLines(
      monitor({ url: 'https://nas.home', domain: { daysLeft: null, expiresAt: null, reason: 'local' } }),
      t,
      'tr',
    )
    expect(lines[1].status).toBe('unknown')
    expect(lines[1].sub).toContain('nas.home')
  })
})
