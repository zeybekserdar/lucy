import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import type { Monitor, Network } from '../../api/types'
import { Button, ButtonLink } from '../../components/Button'
import { StatusIcon } from '../../components/StatusIcon'
import { StatusPill } from '../../components/StatusPill'
import { UptimeBars } from '../../components/UptimeBars'
import { describeFailure, formatAgo, formatClock, formatDuration } from '../../lib/format'
import { softReason } from './derive'
import styles from './FocusCard.module.css'

interface DownProps {
  monitor: Monitor
  now: number
  onCheck: (id: string) => Promise<void>
}

function DownBlock({ monitor, now, onCheck }: DownProps) {
  const { t, i18n } = useTranslation()
  const lng = i18n.language
  const [checking, setChecking] = useState(false)

  const seconds = (iso: string | null) => (iso ? (now - new Date(iso).getTime()) / 1000 : 0)
  const details = [
    monitor.failure && describeFailure(monitor.failure, t),
    t('focus.lastCheck', { ago: formatAgo(seconds(monitor.lastCheck), lng, 'narrow') }),
    monitor.notifiedAt && t('focus.notified', { time: formatClock(monitor.notifiedAt, lng) }),
  ].filter(Boolean)
  const problems = monitor.halfHours.filter((s) => s !== 'ok' && s !== 'no_data').length

  async function check() {
    setChecking(true)
    try {
      await onCheck(monitor.id)
    } finally {
      setChecking(false)
    }
  }

  return (
    <div className={styles.down}>
      <div className={styles.head}>
        <StatusIcon status="down" size={24} />
        <span className={styles.name}>{monitor.name}</span>
        <StatusPill status="down" glyph={false}>
          {t('status.down')}
        </StatusPill>
        <span className={styles.spacer} />
        <span className={styles.duration}>{formatDuration(seconds(monitor.since), t)}</span>
      </div>
      <p className={styles.details}>{details.join(' · ')}</p>
      <div>
        <UptimeBars
          variant="strip"
          states={monitor.halfHours}
          label={t('bars.summary24', { count: problems })}
        />
        <div className={styles.axis}>
          <span>{t('focus.from24h')}</span>
          <span>{t('focus.now')}</span>
        </div>
      </div>
      <div className={styles.actions}>
        <ButtonLink to={`/m/${monitor.id}`} variant="primary">
          {t('focus.details')}
        </ButtonLink>
        <Button onClick={() => void check()} disabled={checking}>
          {checking ? t('focus.checking') : t('focus.checkNow')}
        </Button>
      </div>
    </div>
  )
}

function SoftRow({ monitor, network }: { monitor: Monitor; network: Network }) {
  const { t } = useTranslation()
  const { key, ...params } = softReason(monitor, network)
  const tone = monitor.status === 'warn' ? styles.warnInk : styles.unkInk
  return (
    <Link to={`/m/${monitor.id}`} className={styles.soft}>
      <StatusIcon status={monitor.status} />
      <div className={styles.softText}>
        <span className={styles.softName}>{monitor.name}</span>
        <span className={tone}> — {t(key, params)}</span>
      </div>
      <span aria-hidden="true" className={styles.chevron}>
        ›
      </span>
    </Link>
  )
}

interface Props {
  downs: Monitor[]
  softs: Monitor[]
  network: Network
  now: number
  onCheck: (id: string) => Promise<void>
}

export function FocusCard({ downs, softs, network, now, onCheck }: Props) {
  return (
    <div className={styles.card}>
      {downs.map((monitor) => (
        <DownBlock key={monitor.id} monitor={monitor} now={now} onCheck={onCheck} />
      ))}
      {softs.map((monitor) => (
        <SoftRow key={monitor.id} monitor={monitor} network={network} />
      ))}
    </div>
  )
}
