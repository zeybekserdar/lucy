import type { ReactNode } from 'react'
import styles from './Shell.module.css'

export function Shell({ maxWidth = 880, children }: { maxWidth?: number; children: ReactNode }) {
  return (
    <div className={styles.root}>
      <div className={styles.column} style={{ maxWidth }}>
        {children}
      </div>
    </div>
  )
}
