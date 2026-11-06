import type { DayState } from '../api/types'
import styles from './UptimeBars.module.css'

interface Props {
  states: DayState[]
  label: string
  /** row: listede 4 px çubuklar. strip: kartta kabı dolduran şerit. */
  variant?: 'row' | 'strip'
  titleFor?: (index: number, state: DayState) => string
}

export function UptimeBars({ states, label, variant = 'row', titleFor }: Props) {
  return (
    <div role="img" aria-label={label} className={`${styles.bars} ${styles[variant]}`}>
      {states.map((state, i) => (
        <span key={i} title={titleFor?.(i, state)} className={`${styles.bar} ${styles[state]}`} />
      ))}
    </div>
  )
}
