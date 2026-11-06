import { useTranslation } from 'react-i18next'
import { checkNow } from '../../api/client'
import { Button } from '../../components/Button'
import { Shell } from '../../components/Shell'
import { TopBar } from '../../components/TopBar'
import { deriveHero, groupMonitors } from './derive'
import { EmptyState } from './EmptyState'
import { FocusCard } from './FocusCard'
import { HeroStatus } from './HeroStatus'
import styles from './HomePage.module.css'
import { Legend } from './Legend'
import { MonitorRow } from './MonitorRow'
import { useDashboard } from './useDashboard'

export function HomePage() {
  const { t } = useTranslation()
  const { data, now, failed, reload } = useDashboard()

  if (!data) {
    return (
      <Shell>
        <TopBar showAdd={false} />
        {failed && (
          <div role="alert" className={styles.failure}>
            <h1 className={styles.failureTitle}>{t('load.failed')}</h1>
            <p className={styles.failureBody}>{t('load.failedBody')}</p>
            <Button onClick={() => void reload()}>{t('load.retry')}</Button>
          </div>
        )}
      </Shell>
    )
  }

  if (data.monitors.length === 0) {
    return (
      <Shell>
        <TopBar showAdd={false} />
        <EmptyState />
      </Shell>
    )
  }

  const { downs, softs, healthy } = groupMonitors(data.monitors)
  const hasProblems = downs.length + softs.length > 0

  async function onCheck(id: string) {
    await checkNow(id)
    await reload()
  }

  return (
    <Shell>
      <TopBar />
      <HeroStatus hero={deriveHero(data, now)} />
      {failed && (
        <p role="status" className={styles.stale}>
          {t('load.stale')}
        </p>
      )}
      {hasProblems && (
        <FocusCard downs={downs} softs={softs} network={data.network} now={now} onCheck={onCheck} />
      )}
      {healthy.length > 0 && (
        <>
          <h2 className={styles.listTitle}>
            {t(hasProblems ? 'list.working' : 'list.monitors', { count: healthy.length })}
          </h2>
          <ul className={styles.list}>
            {healthy.map((monitor) => (
              <MonitorRow key={monitor.id} monitor={monitor} now={now} />
            ))}
          </ul>
        </>
      )}
      <Legend />
    </Shell>
  )
}
