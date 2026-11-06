import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router-dom'
import { checkNow, deleteMonitor } from '../../api/client'
import type { ChartRange, CheckResult, History, HistoryEvent, Monitor, MonitorStatus } from '../../api/types'
import { Button, ButtonLink } from '../../components/Button'
import { GLYPHS } from '../../components/glyphs'
import { StatusIcon } from '../../components/StatusIcon'
import {
  describeFailure,
  formatAgo,
  formatClock,
  formatClockSeconds,
  formatDuration,
  formatMs,
  formatNearDay,
  formatRelativeDay,
  formatUptime,
} from '../../lib/format'
import styles from './DetailView.module.css'
import { ResponseChart } from './ResponseChart'
import { conditionActual, conditionExpected, conditionName, expiryLines } from './sections'
import { useHistory } from './useHistory'

const INK: Record<MonitorStatus, string> = {
  up: styles.inkOk,
  warn: styles.inkWarn,
  down: styles.inkDown,
  unknown: styles.inkUnknown,
}

interface Props {
  id: string
  onChanged: () => Promise<void>
  /** Verilirse panel içindedir: geri bağlantısı yerine kapatma düğmesi gösterilir. */
  onClose?: () => void
}

export function DetailView({ id, onChanged, onClose }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [range, setRange] = useState<ChartRange>('7d')
  const history = useHistory(id, range)
  const { data, now, failed, missing, reload } = history
  const isPage = !onClose

  useEffect(() => {
    if (isPage) window.scrollTo(0, 0)
  }, [isPage])

  async function remove(monitor: Monitor) {
    if (!window.confirm(t('monitor.deleteConfirm', { name: monitor.name }))) return
    await deleteMonitor(id)
    await onChanged()
    navigate('/')
  }

  async function check() {
    await checkNow(id)
    await Promise.all([reload(), onChanged()])
  }

  return (
    <div className={styles.stack}>
      <div className={styles.top}>
        {isPage && (
          <Link to="/" className={styles.back}>
            <svg width="18" height="18" viewBox="0 0 20 20" aria-hidden="true">
              <path
                d="M12.5 4.5 7 10l5.5 5.5"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
            {t('nav.back')}
          </Link>
        )}
        <span className={styles.spacer} />
        {data && (
          <ButtonLink to={`/m/${id}/edit`} variant="ghost">
            {t('monitor.edit')}
          </ButtonLink>
        )}
        {onClose && (
          <button type="button" aria-label={t('detail.close')} className={styles.close} onClick={onClose}>
            <svg width="14" height="14" viewBox="0 0 20 20" aria-hidden="true">
              <path d="M5 5l10 10M15 5 5 15" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" />
            </svg>
          </button>
        )}
      </div>

      {missing && <p className={styles.notice}>{t('monitor.notFound')}</p>}
      {!data && failed && !missing && (
        <div role="alert" className={styles.notice}>
          <p>{t('load.failed')}</p>
          <Button onClick={() => void reload()}>{t('load.retry')}</Button>
        </div>
      )}

      {data && (
        <>
          <Header history={data} now={now} onCheck={check} />
          {failed && !missing && (
            <p role="status" className={styles.stale}>
              {t('load.stale')}
            </p>
          )}
          <UptimeTriad monitor={data.monitor} />
          <ResponseChart history={data} shownRange={history.range} range={range} onRange={setRange} />
          <LastCheck result={data.lastResult} status={data.monitor.status} now={now} />
          <ExpiryCard monitor={data.monitor} />
          <EventTimeline events={data.events} now={now} />
          <div>
            <Button variant="ghost" className={styles.delete} onClick={() => void remove(data.monitor)}>
              {t('monitor.delete')}
            </Button>
          </div>
        </>
      )}
    </div>
  )
}

function Header({ history, now, onCheck }: { history: History; now: number; onCheck: () => Promise<void> }) {
  const { t, i18n } = useTranslation()
  const lng = i18n.language
  const { monitor, failedChecks } = history
  const [checking, setChecking] = useState(false)

  async function check() {
    setChecking(true)
    try {
      await onCheck()
    } finally {
      setChecking(false)
    }
  }

  const statusText = {
    up: t('status.ok'),
    warn: t('focus.warning'),
    unknown: t('status.unknown'),
    down: monitor.since
      ? `${t('status.down')} · ${formatDuration((now - new Date(monitor.since).getTime()) / 1000, t)}`
      : t('status.down'),
  }[monitor.status]

  const every = t('detail.every', { time: formatDuration(monitor.intervalSec, t) })
  const ago = monitor.lastCheck && formatAgo((now - new Date(monitor.lastCheck).getTime()) / 1000, lng, 'narrow')
  const meta =
    monitor.status === 'down'
      ? [
          monitor.since && t('detail.since', { when: startedAt(monitor.since, now, lng) }),
          failedChecks > 0 && t('detail.failedChecks', { count: failedChecks }),
          monitor.notifiedAt && t('detail.notified'),
        ]
      : [monitor.responseMs !== null && formatMs(monitor.responseMs), ago || t('detail.neverChecked'), every]

  return (
    <div className={styles.header}>
      <div>
        <h1 id="detail-title" className={styles.name}>
          {monitor.name}
        </h1>
        <div className={styles.url}>{monitor.url}</div>
      </div>
      <div className={styles.statusRow}>
        <span className={`${styles.status} ${INK[monitor.status]}`}>
          <StatusIcon status={monitor.status} size={18} />
          {statusText}
        </span>
        <span className={styles.meta}>{meta.filter(Boolean).join(' · ')}</span>
        <span className={styles.spacer} />
        <Button className={styles.check} onClick={() => void check()} disabled={checking}>
          {checking ? t('focus.checking') : t('focus.checkNow')}
        </Button>
      </div>
    </div>
  )
}

function startedAt(iso: string, now: number, lng: string) {
  const sameDay = new Date(iso).toDateString() === new Date(now).toDateString()
  const clock = formatClock(iso, lng)
  return sameDay ? clock : `${formatNearDay(iso, now, lng)} ${clock}`
}

function UptimeTriad({ monitor }: { monitor: Monitor }) {
  const { t, i18n } = useTranslation()
  const { uptime } = monitor
  const cells = [
    { label: t('detail.uptime24h'), value: uptime.d1, bad: monitor.status === 'down' },
    { label: t('detail.uptime7d'), value: uptime.d7, bad: false },
    { label: t('detail.uptime30d'), value: uptime.d30, bad: false },
  ]
  return (
    <div className={styles.triad}>
      {cells.map((cell) => (
        <div key={cell.label} className={styles.triadCell}>
          <div className={styles.triadLabel}>{cell.label}</div>
          <div className={`${styles.triadValue} ${cell.bad ? styles.inkDown : ''}`}>
            {formatUptime(cell.value, i18n.language)}
          </div>
        </div>
      ))}
    </div>
  )
}

function LastCheck({ result, status, now }: { result: CheckResult | null; status: MonitorStatus; now: number }) {
  const { t, i18n } = useTranslation()
  const lng = i18n.language
  if (!result) {
    return (
      <section className={styles.card}>
        <h2 className={styles.cardTitle}>{t('detail.lastCheck')}</h2>
        <p className={styles.cardNote}>{t('detail.lastCheckNone')}</p>
      </section>
    )
  }
  const code = result.error
    ? describeFailure(`net:${result.error}`, t)
    : `${result.status} ${result.statusText}`.trim()
  const failed = result.conditions.filter((c) => !c.ok).length
  const day = formatRelativeDay(result.at, now, lng).toLocaleLowerCase(lng)
  const meta = [
    `${result.responseMs} ms`,
    `${day} ${formatClockSeconds(result.at, lng)}`,
    failed > 0 && t('result.someFailed', { count: failed, total: result.conditions.length }),
  ]
  return (
    <section className={styles.card}>
      <h2 className={styles.cardTitle}>{t('detail.lastCheck')}</h2>
      <div className={styles.checkHead}>
        <span className={`${styles.code} ${result.ok ? INK[status === 'warn' ? 'up' : status] : styles.inkDown}`}>{code}</span>
        <span className={styles.checkMeta}>{meta.filter(Boolean).join(' · ')}</span>
      </div>
      {result.conditions.map((condition, index) => (
        <div key={index} className={styles.condition}>
          <svg width="18" height="18" viewBox="0 0 20 20" aria-hidden="true" className={styles.conditionGlyph}>
            <path
              d={condition.ok ? GLYPHS.up : GLYPHS.down}
              fill="none"
              stroke={condition.ok ? 'var(--ok-ink)' : 'var(--down-ink)'}
              strokeWidth="2.2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
          <div className={styles.conditionText}>
            <span className={styles.conditionName}>{conditionName(condition, t)}</span>
            <span className={styles.conditionExpected}>
              {t('detail.expected', { value: conditionExpected(condition, t) })}
            </span>
            <span className={`${styles.conditionActual} ${condition.ok ? styles.inkOk : styles.inkDown}`}>
              {conditionActual(condition, t)}
            </span>
          </div>
        </div>
      ))}
    </section>
  )
}

function ExpiryCard({ monitor }: { monitor: Monitor }) {
  const { t, i18n } = useTranslation()
  return (
    <section className={styles.card}>
      <h2 className={styles.cardTitleTight}>{t('detail.expiryTitle')}</h2>
      {expiryLines(monitor, t, i18n.language).map((line) => (
        <div key={line.name} className={styles.expiry}>
          <div className={styles.expiryText}>
            <div className={styles.expiryName}>{line.name}</div>
            <div className={styles.expirySub}>{line.sub}</div>
          </div>
          <span className={`${styles.expiryValue} ${INK[line.status]}`}>
            <StatusIcon status={line.status} size={16} />
            {line.value}
          </span>
        </div>
      ))}
    </section>
  )
}

function EventTimeline({ events, now }: { events: HistoryEvent[]; now: number }) {
  const { t, i18n } = useTranslation()
  const lng = i18n.language

  function when(event: HistoryEvent) {
    const start = `${formatNearDay(event.start, now, lng)} ${formatClock(event.start, lng)}`
    if (!event.end) return t('detail.since', { when: start })
    const sameDay = new Date(event.start).toDateString() === new Date(event.end).toDateString()
    const end = formatClock(event.end, lng)
    return `${start} – ${sameDay ? end : `${formatNearDay(event.end, now, lng)} ${end}`}`
  }

  return (
    <section className={styles.events}>
      <h2 className={styles.eventsTitle}>{t('detail.eventsTitle')}</h2>
      {events.length === 0 && <p className={styles.eventsEmpty}>{t('detail.eventsEmpty')}</p>}
      <ol>
        {events.map((event) => {
          const outage = event.kind === 'outage'
          const endMs = event.end ? new Date(event.end).getTime() : now
          const duration = formatDuration((endMs - new Date(event.start).getTime()) / 1000, t)
          return (
            <li key={`${event.kind}-${event.start}`} className={styles.event}>
              <div className={styles.eventMarks}>
                <span className={outage ? styles.dotOutage : styles.dotNoData} />
                <span className={styles.rail} />
              </div>
              <div className={styles.eventBody}>
                <div className={styles.eventHead}>
                  <span className={outage ? styles.inkDown : styles.eventKindNoData}>
                    {t(outage ? 'detail.outage' : 'detail.noData')}
                  </span>
                  <span className={styles.eventDuration}>
                    {event.end ? duration : t('detail.ongoing', { duration })}
                  </span>
                </div>
                <div className={styles.eventWhen}>
                  {when(event)} ·{' '}
                  {outage ? describeFailure(event.cause ?? '', t) : t('detail.noDataReason')}
                </div>
              </div>
            </li>
          )
        })}
      </ol>
    </section>
  )
}
