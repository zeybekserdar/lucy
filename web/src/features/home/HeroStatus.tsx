import { useTranslation } from 'react-i18next'
import type { MonitorStatus } from '../../api/types'
import { StatusIcon } from '../../components/StatusIcon'
import { formatAgo, formatDuration } from '../../lib/format'
import type { Hero } from './derive'
import styles from './HeroStatus.module.css'

const STATUS: Record<Hero['kind'], MonitorStatus> = {
  ok: 'up',
  down: 'down',
  offline: 'unknown',
  warn: 'warn',
  unknown: 'unknown',
}

export function HeroStatus({ hero }: { hero: Hero }) {
  const { t, i18n } = useTranslation()
  const lng = i18n.language
  const short = (sec: number) => formatAgo(sec, lng, 'narrow')

  const text = (): { title: string; meta: string; sub: string } => {
    switch (hero.kind) {
      case 'ok':
        return {
          title: t('hero.allGood'),
          meta: t('hero.monitors', { count: hero.total }),
          sub: t('hero.lastCheck', { ago: formatAgo(hero.lastCheckSec, lng, 'long') }),
        }
      case 'down':
        return {
          title: t('hero.down', { count: hero.names.length, name: hero.names[0] }),
          meta: formatDuration(hero.durationSec, t),
          sub: hero.others
            ? t('hero.othersUp', { count: hero.others, ago: short(hero.lastCheckSec) })
            : t('hero.noneUp', { ago: short(hero.lastCheckSec) }),
        }
      case 'offline':
        return {
          title: t('hero.offline'),
          meta: formatDuration(hero.durationSec, t),
          sub: t('hero.offlineSub'),
        }
      case 'warn':
      case 'unknown':
        return {
          title: t(`hero.${hero.kind}`, { count: hero.count }),
          meta: t('hero.monitors', { count: hero.total }),
          sub: t('hero.othersUp', { count: hero.others, ago: short(hero.lastCheckSec) }),
        }
    }
  }

  const { title, meta, sub } = text()
  return (
    <section className={styles.hero}>
      <StatusIcon status={STATUS[hero.kind]} size={30} className={styles.icon} />
      <div className={styles.text}>
        {/* Alt satır her yenilemede değiştiği için yalnızca başlık duyurulur. */}
        <div aria-live="polite">
          <h1 className={styles.title}>
            {title}
            <span className={styles.meta}> · {meta}</span>
          </h1>
        </div>
        <p className={styles.sub}>{sub}</p>
      </div>
    </section>
  )
}
