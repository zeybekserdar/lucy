import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { ButtonLink } from './Button'
import { Logo } from './Logo'
import styles from './TopBar.module.css'

export function TopBar({ showAdd = true }: { showAdd?: boolean }) {
  const { t } = useTranslation()
  return (
    <header className={styles.bar}>
      <Link to="/" aria-label={t('app.home')} className={styles.brand}>
        <Logo />
        <span className={styles.name}>Lucy</span>
      </Link>
      <div className={styles.spacer} />
      {showAdd && (
        <ButtonLink to="/new" variant="primary" aria-label={t('nav.addMonitor')} className={styles.add}>
          <svg width="14" height="14" viewBox="0 0 16 16" aria-hidden="true">
            <path d="M8 3v10M3 8h10" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
          </svg>
          <span className={styles.addLabel}>{t('nav.addMonitor')}</span>
        </ButtonLink>
      )}
      <ButtonLink to="/settings" variant="ghost">
        {t('nav.settings')}
      </ButtonLink>
    </header>
  )
}
