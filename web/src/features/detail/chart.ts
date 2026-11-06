import type { ChartRange, HistoryEvent } from '../../api/types'

const HOUR = 3_600_000
const DAY = 24 * HOUR
const MIN_BAND_PERCENT = 0.9
const LABEL_BAND_PERCENT = 8

export interface Band {
  kind: HistoryEvent['kind']
  startMs: number
  endMs: number
  left: number
  width: number
  showLabel: boolean
  labelRight: boolean
}

/** Olayları görünen aralığa kırpar; kesintiler veri yok bantlarının üstünde çizilir. */
export function toBands(events: HistoryEvent[], fromMs: number, toMs: number): Band[] {
  const span = toMs - fromMs
  const bands: Band[] = []
  for (const event of events) {
    const startMs = Math.max(new Date(event.start).getTime(), fromMs)
    const endMs = Math.min(event.end ? new Date(event.end).getTime() : toMs, toMs)
    if (endMs <= startMs) continue
    const width = Math.max(((endMs - startMs) / span) * 100, MIN_BAND_PERCENT)
    const left = Math.min(((startMs - fromMs) / span) * 100, 100 - width)
    const ongoing = event.kind === 'outage' && event.end === null
    bands.push({
      kind: event.kind,
      startMs,
      endMs,
      left,
      width,
      showLabel: width > LABEL_BAND_PERCENT || ongoing,
      labelRight: left > 70,
    })
  }
  return bands.sort((a, b) => Number(a.kind === 'outage') - Number(b.kind === 'outage'))
}

/** Ekseni okunur tutmak için en yüksek değerin biraz üstündeki yuvarlak sayı. */
export function niceMax(highest: number): number {
  const target = Math.max(highest * 1.15, 10)
  const power = 10 ** Math.floor(Math.log10(target))
  const step = [1, 1.5, 2, 2.5, 3, 4, 5, 7.5, 10].find((s) => s * power >= target) ?? 10
  return step * power
}

export const slotCenter = (index: number, count: number) => (index + 0.5) / count

export const yPercent = (value: number, yMax: number) => 100 - (Math.min(value, yMax) / yMax) * 100

export interface PathInput {
  points: (number | null)[]
  bands: Band[]
  fromMs: number
  toMs: number
  yMax: number
  intervalSec: number
}

export function bandAt(bands: Band[], timeMs: number): Band | undefined {
  // Sondan: kesintiler diziyi kapatır ve çakışmada onlar kazanır.
  return bands.findLast((band) => timeMs >= band.startMs && timeMs <= band.endMs)
}

/**
 * Bantların içinde ve uzun boşluklarda çizgi kesilir. Kontrol aralığı dilimden
 * uzunsa boş dilimler normaldir, o yüzden kısa boşluklar birleştirilir.
 */
export function buildPaths({ points, bands, fromMs, toMs, yMax, intervalSec }: PathInput) {
  const count = points.length
  const slotMs = (toMs - fromMs) / count
  const bridgeSlots = Math.max((3 * intervalSec * 1000) / slotMs, 1.5)
  let line = ''
  let area = ''
  let segment: [number, number][] = []
  let lastIndex = -1

  const flush = () => {
    if (segment.length === 1) {
      // Tek nokta da görünsün: yuvarlak uçlu, sıfıra yakın uzunlukta bir çizgi.
      const [x, y] = segment[0]
      line += `M${x} ${y}L${x + 0.01} ${y}`
    } else if (segment.length > 1) {
      const joined = segment.map(([x, y]) => `${x} ${y}`).join('L')
      line += `M${joined}`
      area += `M${segment[0][0]} 100L${joined}L${segment[segment.length - 1][0]} 100Z`
    }
    segment = []
  }

  points.forEach((value, index) => {
    if (bandAt(bands, fromMs + slotCenter(index, count) * (toMs - fromMs))) {
      flush()
      return
    }
    if (value === null) return
    if (lastIndex >= 0 && index - lastIndex > bridgeSlots) flush()
    segment.push([Number((slotCenter(index, count) * 1000).toFixed(1)), Number(yPercent(value, yMax).toFixed(1))])
    lastIndex = index
  })
  flush()
  return { line, area }
}

interface Words {
  now: string
  today: string
}

export function xLabels(range: ChartRange, toMs: number, lng: string, words: Words): string[] {
  const format = (ms: number, options: Intl.DateTimeFormatOptions) =>
    new Intl.DateTimeFormat(lng, options).format(new Date(ms))
  const steps = (count: number, stepMs: number, options: Intl.DateTimeFormatOptions) =>
    Array.from({ length: count }, (_, i) => format(toMs - (count - 1 - i) * stepMs, options)).slice(0, -1)
  if (range === '24h') return [...steps(5, 6 * HOUR, { hour: '2-digit', minute: '2-digit' }), words.now]
  if (range === '7d') return [...steps(8, DAY, { weekday: 'short' }), words.today]
  return [...steps(5, 7.5 * DAY, { day: 'numeric', month: 'short' }), words.today]
}

export function slotLabel(range: ChartRange, timeMs: number, lng: string): string {
  const options: Intl.DateTimeFormatOptions =
    range === '24h'
      ? { hour: '2-digit', minute: '2-digit' }
      : range === '7d'
        ? { weekday: 'short', hour: '2-digit', minute: '2-digit' }
        : { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }
  return new Intl.DateTimeFormat(lng, options).format(new Date(timeMs))
}
