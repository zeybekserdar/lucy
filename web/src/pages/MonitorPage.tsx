import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { deleteMonitor, fetchMonitor } from '../api/client'
import type { MonitorConfig } from '../api/types'
import { Button, ButtonLink } from '../components/Button'
import { Shell } from '../components/Shell'
import { TopBar } from '../components/TopBar'
import styles from './pages.module.css'

// Geçici: gerçek detay paneli gelene kadar düzenleme ve silmeye buradan ulaşılır.
export function MonitorPage() {
  const { t } = useTranslation()
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const [monitor, setMonitor] = useState<MonitorConfig | null>(null)
  const [missing, setMissing] = useState(false)

  useEffect(() => {
    fetchMonitor(id).then(setMonitor, () => setMissing(true))
  }, [id])

  async function remove() {
    if (!monitor || !window.confirm(t('monitor.deleteConfirm', { name: monitor.name }))) return
    await deleteMonitor(id)
    navigate('/')
  }

  return (
    <Shell maxWidth={720}>
      <TopBar />
      <Link to="/" className={styles.back}>
        ‹ {t('nav.back')}
      </Link>
      {missing && <p className={styles.note}>{t('monitor.notFound')}</p>}
      {monitor && (
        <div className={styles.stack}>
          <div>
            <h1 className={styles.monitorName}>{monitor.name}</h1>
            <p className={styles.monitorUrl}>{monitor.url}</p>
          </div>
          <p className={styles.note}>{t('monitor.detailSoon')}</p>
          <div className={styles.inline}>
            <ButtonLink to={`/m/${id}/edit`} variant="primary">
              {t('monitor.edit')}
            </ButtonLink>
            <Button className={styles.danger} onClick={() => void remove()}>
              {t('monitor.delete')}
            </Button>
          </div>
        </div>
      )}
    </Shell>
  )
}
