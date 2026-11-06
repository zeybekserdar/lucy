import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError, changePassword } from '../../api/client'
import { Button } from '../../components/Button'
import { TextField } from '../../components/TextField'
import { passwordIssue } from '../auth/password'
import { useAuth } from '../auth/context'
import { formatAgo, formatDuration } from '../../lib/format'
import styles from './SecurityGroup.module.css'

export function SecurityGroup() {
  const { t, i18n } = useTranslation()
  const { passwordChangedAt, refresh, signOut } = useAuth()
  const [now] = useState(() => Date.now())

  const [open, setOpen] = useState(false)
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [repeat, setRepeat] = useState('')
  const [currentError, setCurrentError] = useState('')
  const [formError, setFormError] = useState('')
  const [busy, setBusy] = useState(false)
  const [updated, setUpdated] = useState(false)

  const issue = passwordIssue(next, repeat)
  const showIssue = issue && (next !== '' || repeat !== '')

  function reset() {
    setCurrent('')
    setNext('')
    setRepeat('')
    setCurrentError('')
    setFormError('')
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (issue || !current || busy) return
    setBusy(true)
    setCurrentError('')
    setFormError('')
    try {
      await changePassword(current, next)
      await refresh()
      reset()
      setOpen(false)
      setUpdated(true)
    } catch (failure) {
      if (failure instanceof ApiError && failure.fields.current) setCurrentError(t('security.wrongCurrent'))
      else if (failure instanceof ApiError && failure.status === 429) {
        setFormError(t('auth.rateLimited', { time: formatDuration(failure.retryAfterSec, t) }))
      } else setFormError(t('auth.failed'))
    } finally {
      setBusy(false)
    }
  }

  const changedSec = passwordChangedAt ? (now - new Date(passwordChangedAt).getTime()) / 1000 : null

  return (
    <>
      <h2 className={styles.title}>{t('security.title')}</h2>
      <section className={styles.group}>
        <div className={styles.row}>
          <div className={styles.rowText}>
            <div className={styles.rowLabel}>{t('security.password')}</div>
            {changedSec !== null && (
              <div className={styles.rowHint}>
                {t('security.changedAgo', { ago: formatAgo(changedSec, i18n.language, 'long') })}
              </div>
            )}
          </div>
          <Button
            aria-expanded={open}
            onClick={() => {
              reset()
              setUpdated(false)
              setOpen(!open)
            }}
          >
            {open ? t('security.cancel') : t('security.change')}
          </Button>
        </div>
        {open && (
          <form onSubmit={(event) => void submit(event)} noValidate className={styles.form}>
            <TextField
              label={t('security.current')}
              type="password"
              autoComplete="current-password"
              value={current}
              onChange={(event) => {
                setCurrent(event.target.value)
                setCurrentError('')
              }}
              error={currentError || undefined}
            />
            <TextField
              label={t('security.next')}
              type="password"
              autoComplete="new-password"
              value={next}
              onChange={(event) => setNext(event.target.value)}
            />
            <TextField
              label={t('security.repeat')}
              type="password"
              autoComplete="new-password"
              value={repeat}
              onChange={(event) => setRepeat(event.target.value)}
            />
            {showIssue && (
              <p aria-live="polite" className={styles.rowHint}>
                {t(`auth.${issue}`)}
              </p>
            )}
            {formError && (
              <p role="alert" className={styles.formError}>
                {formError}
              </p>
            )}
            <Button type="submit" variant="primary" disabled={busy || Boolean(issue) || !current} className={styles.submit}>
              {t('security.update')}
            </Button>
          </form>
        )}
        {updated && (
          <p role="status" className={styles.updated}>
            {t('security.updated')}
          </p>
        )}
        <div className={styles.row}>
          <span className={styles.rowLabel}>{t('security.logout')}</span>
          <Button variant="ghost" className={styles.logout} onClick={() => void signOut()}>
            {t('security.logoutButton')}
          </Button>
        </div>
      </section>
    </>
  )
}
