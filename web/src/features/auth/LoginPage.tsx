import { useId, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError, login, setupPassword } from '../../api/client'
import { GLYPHS } from '../../components/glyphs'
import { Logo } from '../../components/Logo'
import { formatDuration } from '../../lib/format'
import { passwordIssue } from './password'
import styles from './LoginPage.module.css'

interface Props {
  mode: 'setup' | 'login'
  onDone: () => Promise<void>
}

export function LoginPage({ mode, onDone }: Props) {
  const { t } = useTranslation()
  const setup = mode === 'setup'
  const passwordId = useId()
  const repeatId = useId()
  const errorId = useId()

  const [password, setPassword] = useState('')
  const [repeat, setRepeat] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const issue = setup ? passwordIssue(password, repeat) : null
  const canSubmit = !busy && (setup ? issue === null : password !== '')

  function explain(failure: unknown): string {
    if (failure instanceof ApiError) {
      if (failure.status === 401) return t('auth.wrongPassword')
      if (failure.status === 409) return t('auth.alreadySet')
      if (failure.status === 429) {
        return t('auth.rateLimited', { time: formatDuration(failure.retryAfterSec, t) })
      }
    }
    return t('auth.failed')
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!canSubmit) return
    setBusy(true)
    setError('')
    try {
      await (setup ? setupPassword(password) : login(password))
      await onDone()
    } catch (failure) {
      setError(explain(failure))
      setPassword(setup ? password : '')
      setBusy(false)
    }
  }

  return (
    <main className={styles.page}>
      <form onSubmit={(event) => void submit(event)} noValidate className={styles.form}>
        <Logo size={64} />
        <h1 className={styles.title}>{setup ? t('auth.welcome') : 'Lucy'}</h1>
        <p className={styles.sub}>{setup ? t('auth.setupSub') : t('auth.loginSub')}</p>
        <div className={styles.fields}>
          <div className={styles.field}>
            <label htmlFor={passwordId} className={styles.label}>
              {setup ? t('auth.newPassword') : t('auth.password')}
            </label>
            <input
              id={passwordId}
              type="password"
              autoFocus
              autoComplete={setup ? 'new-password' : 'current-password'}
              placeholder="••••••••••••"
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? errorId : undefined}
              className={`${styles.input} ${error ? styles.invalid : ''}`}
              value={password}
              onChange={(event) => {
                setPassword(event.target.value)
                setError('')
              }}
            />
          </div>
          {setup && (
            <>
              <div className={styles.field}>
                <label htmlFor={repeatId} className={styles.label}>
                  {t('auth.repeatPassword')}
                </label>
                <input
                  id={repeatId}
                  type="password"
                  autoComplete="new-password"
                  className={styles.input}
                  value={repeat}
                  onChange={(event) => setRepeat(event.target.value)}
                />
              </div>
              <div aria-live="polite" className={issue ? styles.rule : styles.ruleOk}>
                {!issue && (
                  <svg width="14" height="14" viewBox="0 0 20 20" aria-hidden="true">
                    <path
                      d={GLYPHS.up}
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2.4"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                )}
                {issue ? t(`auth.${issue}`) : t('auth.rulesOk')}
              </div>
            </>
          )}
          {error && (
            <div id={errorId} role="alert" className={styles.error}>
              <svg width="16" height="16" viewBox="0 0 20 20" aria-hidden="true" className={styles.errorIcon}>
                <circle cx="10" cy="10" r="10" fill="var(--down)" />
                <path d={GLYPHS.warn} stroke="var(--on-status)" strokeWidth="2.2" strokeLinecap="round" />
              </svg>
              <span>{error}</span>
            </div>
          )}
          <button type="submit" disabled={!canSubmit} className={styles.submit}>
            {busy ? t('auth.working') : setup ? t('auth.setupCta') : t('auth.loginCta')}
          </button>
        </div>
        <p className={styles.foot}>{setup ? t('auth.setupFoot') : t('auth.loginFoot')}</p>
      </form>
    </main>
  )
}
