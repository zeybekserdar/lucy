import { useId, type InputHTMLAttributes, type ReactNode } from 'react'
import styles from './TextField.module.css'

export function FieldError({ id, children, action }: { id?: string; children: ReactNode; action?: ReactNode }) {
  return (
    <span id={id} className={styles.error}>
      <span className={styles.errorText}>
        <svg width="14" height="14" viewBox="0 0 20 20" aria-hidden="true" style={{ flex: 'none' }}>
          <circle cx="10" cy="10" r="10" fill="var(--down)" />
          <path d="M10 5.6v5.2M10 14.3v.1" stroke="var(--on-status)" strokeWidth="2.4" strokeLinecap="round" />
        </svg>
        {children}
      </span>
      {action}
    </span>
  )
}

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label: string
  /** Etiketin yanında soluk görünen ek, örneğin "isteğe bağlı". */
  hint?: string
  error?: string
  errorAction?: ReactNode
  mono?: boolean
  /** Alanın içinde sağda görünen birim, örneğin "saniye". */
  suffix?: string
}

export function TextField({ label, hint, error, errorAction, mono, suffix, className, ...input }: Props) {
  const id = useId()
  const errorId = `${id}-error`
  return (
    <div className={`${styles.field} ${className ?? ''}`}>
      <label htmlFor={id} className={styles.label}>
        {label}
        {hint && <span className={styles.hint}> · {hint}</span>}
      </label>
      <div className={`${styles.box} ${error ? styles.invalid : ''}`}>
        <input
          id={id}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? errorId : undefined}
          className={`${styles.input} ${mono ? styles.mono : ''}`}
          {...input}
        />
        {suffix && <span className={styles.suffix}>{suffix}</span>}
      </div>
      {error && (
        <FieldError id={errorId} action={errorAction}>
          {error}
        </FieldError>
      )}
    </div>
  )
}
