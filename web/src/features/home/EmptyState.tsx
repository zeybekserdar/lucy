import { useTranslation } from 'react-i18next'
import { ButtonLink } from '../../components/Button'
import { Logo } from '../../components/Logo'
import styles from './EmptyState.module.css'

export function EmptyState() {
  const { t } = useTranslation()
  return (
    <div className={styles.empty}>
      <Logo size={56} />
      <h1 className={styles.title}>{t('empty.title')}</h1>
      <p className={styles.body}>{t('empty.body')}</p>
      <div className={styles.actions}>
        <ButtonLink to="/new" variant="primary" size="md">
          {t('empty.add')}
        </ButtonLink>
        <ButtonLink to="/settings" size="md">
          {t('empty.import')}
        </ButtonLink>
      </div>
    </div>
  )
}
