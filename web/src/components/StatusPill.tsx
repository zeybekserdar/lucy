import type { ReactNode } from 'react'
import type { MonitorStatus } from '../api/types'
import { GLYPHS } from './glyphs'
import styles from './StatusPill.module.css'

interface Props {
  status: MonitorStatus
  children: ReactNode
  glyph?: boolean
}

export function StatusPill({ status, children, glyph = true }: Props) {
  return (
    <span className={`${styles.pill} ${styles[status]} ${glyph ? styles.withGlyph : ''}`}>
      {glyph && (
        <svg width="12" height="12" viewBox="0 0 20 20" aria-hidden="true">
          <path
            d={GLYPHS[status]}
            fill="none"
            stroke="currentColor"
            strokeWidth="2.6"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      )}
      {children}
    </span>
  )
}
