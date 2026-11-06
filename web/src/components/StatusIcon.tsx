import type { MonitorStatus } from '../api/types'
import { GLYPHS } from './glyphs'

const COLORS: Record<MonitorStatus, { fill: string; stroke: string }> = {
  up: { fill: 'var(--ok)', stroke: 'var(--on-status)' },
  warn: { fill: 'var(--warn)', stroke: 'var(--on-warn)' },
  down: { fill: 'var(--down)', stroke: 'var(--on-status)' },
  unknown: { fill: 'var(--unk)', stroke: 'var(--on-status)' },
}

interface Props {
  status: MonitorStatus
  size?: number
  /** Listede çalışan monitörler dairesiz, sade bir onay işaretiyle görünür. */
  plain?: boolean
  className?: string
}

export function StatusIcon({ status, size = 20, plain = false, className }: Props) {
  const { fill, stroke } = plain ? { fill: 'transparent', stroke: 'var(--ok-ink)' } : COLORS[status]
  return (
    <svg width={size} height={size} viewBox="0 0 20 20" aria-hidden="true" className={className} style={{ flex: 'none' }}>
      <circle cx="10" cy="10" r="10" fill={fill} />
      <path
        d={GLYPHS[status]}
        fill="none"
        stroke={stroke}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}
