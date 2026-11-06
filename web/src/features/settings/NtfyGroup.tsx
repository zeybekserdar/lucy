import { useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError, fetchNtfy, saveNtfy, testNtfy } from '../../api/client'
import type { NtfySettings } from '../../api/types'
import { Button } from '../../components/Button'
import { FieldError } from '../../components/TextField'
import { GLYPHS } from '../../components/glyphs'
import { formatClock } from '../../lib/format'
import styles from './Settings.module.css'
import { generateTopic } from './topic'

type Note = { ok: boolean; text: string }

export function NtfyGroup() {
  const { t, i18n } = useTranslation()
  const [saved, setSaved] = useState<NtfySettings | null>(null)
  const [draft, setDraft] = useState<NtfySettings>({ server: '', topic: '', token: '' })
  const [loadFailed, setLoadFailed] = useState(false)
  const [showToken, setShowToken] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [note, setNote] = useState<Note | null>(null)
  const [busy, setBusy] = useState<'save' | 'test' | null>(null)

  useEffect(() => {
    fetchNtfy().then(
      (settings) => {
        setSaved(settings)
        setDraft(settings)
      },
      () => setLoadFailed(true),
    )
  }, [])

  const dirty = saved !== null && (Object.keys(draft) as (keyof NtfySettings)[]).some((k) => draft[k] !== saved[k])

  const topicChanged = saved !== null && draft.topic !== '' && draft.topic !== saved.topic

  function change(key: keyof NtfySettings, value: string) {
    setDraft({ ...draft, [key]: value })
    setErrors({ ...errors, [key]: '' })
    setNote(null)
  }

  async function save(event: FormEvent) {
    event.preventDefault()
    if (!dirty || busy) return
    setBusy('save')
    setNote(null)
    try {
      await saveNtfy(draft)
      // Sunucu boşlukları ve sondaki / işaretini temizler.
      const settings = await fetchNtfy()
      setSaved(settings)
      setDraft(settings)
      setNote({ ok: true, text: t('ntfy.saved') })
    } catch (failure) {
      if (failure instanceof ApiError && Object.keys(failure.fields).length > 0) setErrors(failure.fields)
      else setNote({ ok: false, text: t('ntfy.saveFailed') })
    } finally {
      setBusy(null)
    }
  }

  async function test() {
    setBusy('test')
    setNote(null)
    try {
      const { ok, status } = await testNtfy(draft)
      if (ok) {
        const time = formatClock(new Date().toISOString(), i18n.language)
        setNote({ ok: true, text: t('ntfy.sent', { status, time }) })
      } else {
        setNote({ ok: false, text: status ? t('ntfy.rejected', { status }) : t('ntfy.unreachable') })
      }
    } catch (failure) {
      if (failure instanceof ApiError && Object.keys(failure.fields).length > 0) setErrors(failure.fields)
      else setNote({ ok: false, text: t('ntfy.failed') })
    } finally {
      setBusy(null)
    }
  }

  const errorText = (key: keyof NtfySettings) => (errors[key] ? t(`ntfy.error.${errors[key]}`) : '')

  return (
    <>
      <h2 className={styles.groupTitle}>{t('ntfy.title')}</h2>
      {loadFailed && (
        <p role="alert" className={styles.alert}>
          {t('load.failedBody')}
        </p>
      )}
      <form onSubmit={(event) => void save(event)} noValidate className={styles.group}>
        <label className={styles.fieldRow}>
          <span className={styles.fieldLabel}>{t('ntfy.server')}</span>
          <span className={styles.fieldControl}>
            <input
              className={styles.input}
              value={draft.server}
              disabled={!saved}
              inputMode="url"
              autoComplete="off"
              spellCheck={false}
              aria-invalid={errors.server ? true : undefined}
              onChange={(event) => change('server', event.target.value)}
            />
          </span>
          {errors.server && <FieldError>{errorText('server')}</FieldError>}
        </label>
        <div className={styles.fieldRow}>
          <label htmlFor="ntfy-topic" className={styles.fieldLabel}>
            {t('ntfy.topic')}
          </label>
          <span className={styles.fieldControl}>
            <input
              id="ntfy-topic"
              className={styles.input}
              value={draft.topic}
              disabled={!saved}
              autoComplete="off"
              spellCheck={false}
              aria-invalid={errors.topic ? true : undefined}
              onChange={(event) => change('topic', event.target.value)}
            />
            <button
              type="button"
              className={styles.dice}
              disabled={!saved}
              aria-label={t('ntfy.generate')}
              title={t('ntfy.generate')}
              onClick={() => change('topic', generateTopic())}
            >
              <svg width="18" height="18" viewBox="0 0 20 20" aria-hidden="true">
                <rect x="3" y="3" width="14" height="14" rx="3.5" fill="none" stroke="currentColor" strokeWidth="1.6" />
                <circle cx="7.2" cy="7.2" r="1.25" fill="currentColor" />
                <circle cx="12.8" cy="7.2" r="1.25" fill="currentColor" />
                <circle cx="10" cy="10" r="1.25" fill="currentColor" />
                <circle cx="7.2" cy="12.8" r="1.25" fill="currentColor" />
                <circle cx="12.8" cy="12.8" r="1.25" fill="currentColor" />
              </svg>
            </button>
          </span>
          {errors.topic && <FieldError>{errorText('topic')}</FieldError>}
          {topicChanged && !errors.topic && <p className={styles.fieldNote}>{t('ntfy.subscribeHint')}</p>}
        </div>
        <div className={styles.fieldRow}>
          <label htmlFor="ntfy-token" className={styles.fieldLabel}>
            {t('ntfy.token')}
            <span className={styles.optional}> · {t('editor.optional')}</span>
          </label>
          <span className={styles.fieldControl}>
            <input
              id="ntfy-token"
              className={styles.input}
              type={showToken ? 'text' : 'password'}
              value={draft.token}
              disabled={!saved}
              placeholder={t('ntfy.tokenNone')}
              autoComplete="off"
              spellCheck={false}
              onChange={(event) => change('token', event.target.value)}
            />
            <button type="button" className={styles.reveal} onClick={() => setShowToken(!showToken)}>
              {t(showToken ? 'ntfy.hide' : 'ntfy.show')}
            </button>
          </span>
        </div>
        <div className={styles.actions}>
          <Button className={styles.surfaceButton} onClick={() => void test()} disabled={!saved || busy !== null}>
            {t(busy === 'test' ? 'ntfy.testing' : 'ntfy.test')}
          </Button>
          {dirty && (
            <Button type="submit" variant="primary" disabled={busy !== null}>
              {t(busy === 'save' ? 'ntfy.saving' : 'ntfy.save')}
            </Button>
          )}
          {note && (
            <span role="status" className={`${styles.note} ${note.ok ? styles.noteOk : styles.noteBad}`}>
              <svg width="14" height="14" viewBox="0 0 20 20" aria-hidden="true" style={{ flex: 'none' }}>
                <path
                  d={note.ok ? GLYPHS.up : GLYPHS.down}
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.4"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
              {note.text}
            </span>
          )}
        </div>
      </form>
      <p className={styles.about}>{t('ntfy.about')}</p>
    </>
  )
}
