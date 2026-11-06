import { describe, expect, it } from 'vitest'
import type { HistoryEvent } from '../../api/types'
import { bandAt, buildPaths, niceMax, toBands, xLabels } from './chart'

const HOUR = 3_600_000
const from = Date.UTC(2026, 10, 1)
const to = from + 10 * HOUR
const iso = (ms: number) => new Date(ms).toISOString()

const outage = (start: number, end: number | null): HistoryEvent => ({
  kind: 'outage',
  start: iso(start),
  end: end === null ? null : iso(end),
})

describe('toBands', () => {
  it('clips events to the visible range', () => {
    const [band] = toBands([outage(from - HOUR, from + HOUR)], from, to)
    expect(band.left).toBe(0)
    expect(band.width).toBeCloseTo(10)
  })

  it('drops events outside the range', () => {
    expect(toBands([outage(from - 2 * HOUR, from - HOUR)], from, to)).toEqual([])
  })

  it('keeps a very short outage visible inside the plot', () => {
    const [band] = toBands([outage(to - 1000, to)], from, to)
    expect(band.width).toBe(0.9)
    expect(band.left + band.width).toBeCloseTo(100)
  })

  it('runs an ongoing outage up to now and labels it', () => {
    const [band] = toBands([outage(to - HOUR / 10, null)], from, to)
    expect(band.endMs).toBe(to)
    expect(band.showLabel).toBe(true)
  })

  it('lets an outage win over a stretch without data', () => {
    const bands = toBands(
      [outage(from + HOUR, from + 2 * HOUR), { kind: 'no_data', start: iso(from), end: iso(to) }],
      from,
      to,
    )
    expect(bandAt(bands, from + 1.5 * HOUR)?.kind).toBe('outage')
    expect(bandAt(bands, from + 5 * HOUR)?.kind).toBe('no_data')
  })
})

describe('buildPaths', () => {
  const base = { bands: [], fromMs: from, toMs: to, yMax: 100, intervalSec: 60 }

  it('draws one segment for continuous data', () => {
    const { line, area } = buildPaths({ ...base, points: [10, 20, 30, 40] })
    expect(line.match(/M/g)).toHaveLength(1)
    expect(area).toContain('Z')
  })

  it('breaks the line inside a band', () => {
    const bands = toBands([outage(from + 4 * HOUR, from + 6 * HOUR)], from, to)
    const points = Array.from({ length: 10 }, () => 50)
    expect(buildPaths({ ...base, bands, points }).line.match(/M/g)).toHaveLength(2)
  })

  it('breaks the line over a long empty stretch', () => {
    const { line } = buildPaths({ ...base, points: [10, 20, null, null, null, 30, 40, 50, 60, 70] })
    expect(line.match(/M/g)).toHaveLength(2)
  })

  it('bridges empty slots when checks are rarer than slots', () => {
    const { line } = buildPaths({ ...base, intervalSec: 3600, points: [10, null, 20, null, 30, null, 40, null, 50, null] })
    expect(line.match(/M/g)).toHaveLength(1)
  })

  it('still shows a lone point', () => {
    const { line, area } = buildPaths({ ...base, points: [null, null, 30, null, null, null, null, null] })
    expect(line).not.toBe('')
    expect(area).toBe('')
  })

  it('stays empty without data', () => {
    expect(buildPaths({ ...base, points: [null, null] })).toEqual({ line: '', area: '' })
  })
})

describe('niceMax', () => {
  it('rounds up above the highest value', () => {
    expect(niceMax(96)).toBe(150)
    expect(niceMax(1700)).toBe(2000)
  })
  it('has a floor for very fast services', () => {
    expect(niceMax(1)).toBe(10)
  })
})

describe('xLabels', () => {
  const words = { now: 'Şimdi', today: 'Bugün' }
  it('ends with the present and matches the design’s counts', () => {
    expect(xLabels('24h', to, 'tr', words)).toHaveLength(5)
    expect(xLabels('7d', to, 'tr', words)).toHaveLength(8)
    expect(xLabels('30d', to, 'tr', words)).toHaveLength(5)
    expect(xLabels('24h', to, 'tr', words).at(-1)).toBe('Şimdi')
    expect(xLabels('7d', to, 'tr', words).at(-1)).toBe('Bugün')
  })
})
