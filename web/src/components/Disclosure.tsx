import { useId, type ReactNode } from 'react'
import styles from './Disclosure.module.css'

interface Props {
  title: string
  summary?: string
  open: boolean
  onOpenChange: (open: boolean) => void
  children: ReactNode
}

export function Disclosure({ title, summary, open, onOpenChange, children }: Props) {
  const panelId = useId()
  return (
    <>
      <button
        type="button"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => onOpenChange(!open)}
        className={styles.trigger}
      >
        <span className={styles.text}>
          <span className={styles.title}>{title}</span>
          {summary && <span className={styles.summary}>{summary}</span>}
        </span>
        <svg width="16" height="16" viewBox="0 0 20 20" aria-hidden="true" className={`${styles.chevron} ${open ? styles.open : ''}`}>
          <path
            d="M7 4.5 12.5 10 7 15.5"
            fill="none"
            stroke="var(--ink2)"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </button>
      <div id={panelId} hidden={!open}>
        {children}
      </div>
    </>
  )
}
