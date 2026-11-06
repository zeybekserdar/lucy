import { useMemo, useState, type PointerEvent } from 'react'
import { useTranslation } from 'react-i18next'
import type { ChartRange, History } from '../../api/types'
import { bandAt, buildPaths, niceMax, slotCenter, slotLabel, toBands, xLabels, yPercent } from './chart'
import styles from './ResponseChart.module.css'

const RANGES: ChartRange[] = ['24h', '7d', '30d']

interface Props {
  history: History
  /** history'nin ait olduğu aralık. */
  shownRange: ChartRange
  range: ChartRange
  onRange: (range: ChartRange) => void
}

export function ResponseChart({ history, shownRange, range, onRange }: Props) {
  const { t, i18n } = useTranslation()
  const lng = i18n.language
  const { points, events, avgMs, from, to, monitor } = history
  const [hover, setHover] = useState<number | null>(null)

  const chart = useMemo(() => {
    const fromMs = new Date(from).getTime()
    const toMs = new Date(to).getTime()
    const values = points.filter((p) => p !== null)
    const highest = values.length ? Math.max(...values) : 0
    const yMax = niceMax(highest)
    const bands = toBands(events, fromMs, toMs)
    const paths = buildPaths({ points, bands, fromMs, toMs, yMax, intervalSec: monitor.intervalSec })
    return { fromMs, toMs, highest, yMax, bands, paths, empty: values.length === 0 }
  }, [points, events, from, to, monitor.intervalSec])
  const { fromMs, toMs, highest, yMax, bands, paths, empty } = chart

  const labels = xLabels(shownRange, toMs, lng, { now: t('detail.now'), today: t('detail.today') })

  function onPointer(event: PointerEvent<HTMLDivElement>) {
    const rect = event.currentTarget.getBoundingClientRect()
    const ratio = (event.clientX - rect.left) / rect.width
    setHover(Math.min(Math.max(Math.floor(ratio * points.length), 0), points.length - 1))
  }

  // Aralık değişince dilim sayısı da değişir, eski imleç konumu geçersiz olabilir.
  const slot = hover !== null && hover < points.length ? hover : null
  const position = slot === null ? 0 : slotCenter(slot, points.length)
  const slotMs = fromMs + position * (toMs - fromMs)
  const slotBand = slot === null ? undefined : bandAt(bands, slotMs)
  const slotValue = slot === null || slotBand ? null : points[slot]

  return (
    <section className={styles.card}>
      <div className={styles.head}>
        <h2 className={styles.title}>{t('detail.chartTitle')}</h2>
        {avgMs !== null && <span className={styles.average}>{t('detail.average', { ms: avgMs })}</span>}
        <span className={styles.spacer} />
        <div role="group" aria-label={t('detail.rangeLabel')} className={styles.ranges}>
          {RANGES.map((value) => (
            <button
              key={value}
              type="button"
              aria-pressed={value === range}
              className={styles.range}
              onClick={() => onRange(value)}
            >
              {t(`detail.range.${value}`)}
            </button>
          ))}
        </div>
      </div>

      <div className={styles.grid}>
        <div className={styles.yAxis} aria-hidden="true">
          <span className={styles.yTop}>{yMax} ms</span>
          <span>{Math.round(yMax / 2)}</span>
          <span className={styles.yBottom}>0</span>
        </div>
        <div
          role="img"
          aria-label={empty ? t('detail.chartEmpty') : t('detail.chartSummary', { ms: avgMs, max: highest })}
          className={styles.plot}
          onPointerMove={onPointer}
          onPointerDown={onPointer}
          onPointerLeave={() => setHover(null)}
        >
          <div className={styles.gridTop} />
          <div className={styles.gridMid} />
          {bands.map((band) => (
            <div
              key={`${band.kind}-${band.startMs}`}
              className={band.kind === 'outage' ? styles.outage : styles.noData}
              style={{ left: `${band.left}%`, width: `${band.width}%` }}
            >
              {band.showLabel && (
                <span className={`${styles.bandLabel} ${band.labelRight ? styles.bandLabelRight : ''}`}>
                  {t(band.kind === 'outage' ? 'detail.outage' : 'detail.noData')}
                </span>
              )}
            </div>
          ))}
          <svg viewBox="0 0 1000 100" preserveAspectRatio="none" aria-hidden="true" className={styles.svg}>
            <path d={paths.area} fill="var(--ink)" fillOpacity="0.06" />
            <path
              d={paths.line}
              fill="none"
              stroke="var(--ink)"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
              vectorEffect="non-scaling-stroke"
            />
          </svg>
          {empty && bands.length === 0 && <p className={styles.empty}>{t('detail.chartEmpty')}</p>}
          {slot !== null && (
            <>
              <div className={styles.cursor} style={{ left: `${position * 100}%` }} />
              {slotValue !== null && (
                <div
                  className={styles.dot}
                  style={{ left: `${position * 100}%`, top: `${yPercent(slotValue, yMax)}%` }}
                />
              )}
              <div
                className={`${styles.tip} ${position > 0.6 ? styles.tipLeft : ''}`}
                style={{ left: `${position * 100}%` }}
              >
                <span className={styles.tipTime}>{slotLabel(shownRange, slotMs, lng)}</span>
                <span className={styles.tipValue}>
                  {slotBand
                    ? t(slotBand.kind === 'outage' ? 'detail.outage' : 'detail.noDataTitle')
                    : slotValue !== null
                      ? `${slotValue} ms`
                      : t('detail.noValue')}
                </span>
              </div>
            </>
          )}
        </div>
        <span />
        <div className={styles.xAxis} aria-hidden="true">
          {labels.map((label, index) => (
            <span key={index}>{label}</span>
          ))}
        </div>
      </div>

      <div className={styles.legend}>
        <span className={styles.legendItem}>
          <span className={styles.keyLine} />
          {t('detail.chartTitle')}
        </span>
        <span className={styles.legendItem}>
          <span className={styles.keyOutage} />
          {t('detail.outage')}
        </span>
        <span className={styles.legendItem}>
          <span className={styles.keyNoData} />
          {t('detail.noDataLegend')}
        </span>
      </div>
    </section>
  )
}
