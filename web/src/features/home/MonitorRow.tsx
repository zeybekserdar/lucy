import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import type { DayState, Monitor } from '../../api/types'
import { StatusIcon } from '../../components/StatusIcon'
import { UptimeBars } from '../../components/UptimeBars'
import { formatDuration, formatMs, formatRelativeDay, formatUptime } from '../../lib/format'
import { recentIncident } from './derive'
import styles from './MonitorRow.module.css'

export function MonitorRow({ monitor, now }: { monitor: Monitor; now: number }) {
  const { t, i18n } = useTranslation()
  const lng = i18n.language

  const incident = recentIncident(monitor, now)
  // Henüz sorgulanmadıysa ya da yerel bir adresse söylenecek bir şey yok.
  const { reason } = monitor.domain
  const domainUnknown = monitor.domain.daysLeft === null && Boolean(reason) && reason !== 'local'
  const note = incident
    ? t('list.incidentNote', {
        when: formatRelativeDay(incident.endedAt, now, lng),
        duration: formatDuration(incident.durationSec, t),
      })
    : domainUnknown
      ? t('list.domainUnknown')
      : null

  const problemDays = monitor.days.filter((d) => d !== 'ok' && d !== 'no_data').length
  const dayTitle = (index: number, state: DayState) => {
    const ago = monitor.days.length - 1 - index
    const when = ago === 0 ? t('bars.today') : t('bars.daysAgo', { count: ago })
    return `${when} · ${t(`status.${state}`)}`
  }

  return (
    <li className={styles.item}>
      <Link to={`/m/${monitor.id}`} className={styles.row}>
        <StatusIcon status="up" plain size={18} className={styles.icon} />
        <div className={styles.title}>
          <span className={styles.name}>{monitor.name}</span>
          {note && <span className={incident ? styles.note : styles.noteUnknown}>{note}</span>}
        </div>
        <div className={styles.bars}>
          <UptimeBars
            states={monitor.days}
            label={t('bars.summary30', { count: problemDays })}
            titleFor={dayTitle}
          />
        </div>
        <span className={styles.uptime}>{formatUptime(monitor.uptime.d30, lng)}</span>
        <span className={styles.ms}>{formatMs(monitor.responseMs)}</span>
      </Link>
    </li>
  )
}
