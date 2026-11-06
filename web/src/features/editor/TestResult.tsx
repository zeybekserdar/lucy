import { useTranslation } from 'react-i18next'
import type { CheckResult, ConditionResult } from '../../api/types'
import { StatusIcon } from '../../components/StatusIcon'
import { GLYPHS } from '../../components/glyphs'
import { describeFailure } from '../../lib/format'
import styles from './TestResult.module.css'

function useConditionText() {
  const { t } = useTranslation()
  return (c: ConditionResult) => {
    if (c.reason === 'no_response') return t('result.noResponse')
    if (c.reason === 'not_json') return t('result.notJson', { actual: c.actual || '—' })
    if (c.reason === 'path_missing') return t('result.pathMissing', { path: c.path })
    const values = { actual: c.actual, expected: c.expected, path: c.path }
    switch (c.type) {
      case 'status':
        return t('result.status', values)
      case 'response':
        return t('result.response', values)
      case 'json':
        return c.ok ? `${c.path} = ${c.actual}` : t('result.json', values)
      case 'body': {
        const found = c.actual === 'true'
        if (c.op === 'contains') return t(found ? 'result.bodyFound' : 'result.bodyMissing', values)
        return t(found ? 'result.bodyForbidden' : 'result.bodyAbsent', values)
      }
    }
  }
}

export function TestResult({ result }: { result: CheckResult }) {
  const { t, i18n } = useTranslation()
  const text = useConditionText()
  const failed = result.conditions.filter((c) => !c.ok).length
  const total = result.conditions.length
  const time = new Intl.DateTimeFormat(i18n.language, { timeStyle: 'medium' }).format(new Date(result.at))
  const outcome = result.error
    ? describeFailure(`net:${result.error}`, t)
    : `${result.status} ${result.statusText}`.trim()
  const summary = [outcome, `${result.responseMs} ms`, result.contentType].filter(Boolean).join(' · ')

  return (
    <section aria-live="polite" className={styles.card}>
      <div className={`${styles.head} ${result.ok ? styles.passed : styles.failed}`}>
        <StatusIcon status={result.ok ? 'up' : 'down'} />
        <span className={styles.title}>{t(result.ok ? 'result.passed' : 'result.failed')}</span>
        <span className={styles.meta}>
          {result.ok
            ? t('result.allMet', { count: total })
            : t('result.someFailed', { count: failed, total })}{' '}
          · {time}
        </span>
      </div>
      <div className={styles.request}>
        {result.method} {result.url} → {summary}
      </div>
      {result.conditions.map((condition, i) => (
        <div key={i} className={styles.row}>
          <svg width="16" height="16" viewBox="0 0 20 20" aria-hidden="true" className={styles.glyph}>
            <path
              d={condition.ok ? GLYPHS.up : GLYPHS.down}
              fill="none"
              stroke={condition.ok ? 'var(--ok-ink)' : 'var(--down-ink)'}
              strokeWidth="2.4"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
          <span className={styles.rowText}>
            <span className={styles.rowName}>{t(`editor.type.${condition.type}`)}</span>
            <span className={styles.rowDetail}> — {text(condition)}</span>
          </span>
        </div>
      ))}
    </section>
  )
}
