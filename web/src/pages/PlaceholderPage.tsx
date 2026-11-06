import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Shell } from '../components/Shell'
import { TopBar } from '../components/TopBar'
import styles from './pages.module.css'

export function PlaceholderPage() {
  const { t } = useTranslation()
  return (
    <Shell maxWidth={680}>
      <TopBar />
      <Link to="/" className={styles.back}>
        ‹ {t('nav.back')}
      </Link>
      <p className={styles.note}>{t('placeholder.body')}</p>
    </Shell>
  )
}
