import { useTranslation } from 'react-i18next'
import styles from './Stepper.module.css'

interface Props {
  value: number
  min: number
  max: number
  onChange: (value: number) => void
}

export function Stepper({ value, min, max, onChange }: Props) {
  const { t } = useTranslation()
  return (
    <div className={styles.stepper}>
      <button
        type="button"
        aria-label={t('stepper.decrease')}
        disabled={value <= min}
        onClick={() => onChange(value - 1)}
        className={styles.step}
      >
        −
      </button>
      <span aria-live="polite" className={styles.value}>
        {value}
      </span>
      <button
        type="button"
        aria-label={t('stepper.increase')}
        disabled={value >= max}
        onClick={() => onChange(value + 1)}
        className={styles.step}
      >
        +
      </button>
    </div>
  )
}
