import { useTranslation } from 'react-i18next'
import styles from './Legend.module.css'

const ITEMS = [
  { key: 'status.ok', color: 'var(--ok)', height: 12 },
  { key: 'status.partial', color: 'var(--warn)', height: 12 },
  { key: 'status.down', color: 'var(--down)', height: 12 },
  { key: 'status.unknown', color: 'var(--unk)', height: 12 },
  { key: 'status.noDataLegend', color: 'var(--none)', height: 4 },
] as const

export function Legend() {
  const { t } = useTranslation()
  return (
    <div className={styles.legend}>
      {ITEMS.map((item) => (
        <span key={item.key} className={styles.item}>
          <span className={styles.swatch} style={{ background: item.color, height: item.height }} />
          <span className={styles.label}>{t(item.key)}</span>
        </span>
      ))}
    </div>
  )
}
