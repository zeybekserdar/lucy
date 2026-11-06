import { useEffect, useRef, useState, type ChangeEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError, fetchDashboard, importBackup, previewBackup } from '../../api/client'
import type { ImportPreview } from '../../api/types'
import { Button, ButtonAnchor } from '../../components/Button'
import styles from './Settings.module.css'

interface Pending {
  file: string
  yaml: string
  preview: ImportPreview
}

export function BackupGroup({ onImported }: { onImported: () => void }) {
  const { t } = useTranslation()
  const input = useRef<HTMLInputElement>(null)
  const [count, setCount] = useState<number | null>(null)
  const [pending, setPending] = useState<Pending | null>(null)
  const [error, setError] = useState('')
  const [done, setDone] = useState('')
  const [busy, setBusy] = useState<'read' | 'import' | null>(null)

  useEffect(() => {
    fetchDashboard().then(
      (dashboard) => setCount(dashboard.monitors.length),
      () => {},
    )
  }, [done])

  function explain(failure: unknown) {
    if (failure instanceof ApiError && failure.code === 'bad_yaml') return t('backup.badFile')
    if (failure instanceof ApiError && failure.code === 'bad_version') return t('backup.badVersion')
    return t('backup.failed')
  }

  async function choose(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = '' // aynı dosya düzeltilip yeniden seçilebilsin
    if (!file) return
    setBusy('read')
    setError('')
    setDone('')
    setPending(null)
    try {
      const yaml = await file.text()
      setPending({ file: file.name, yaml, preview: await previewBackup(yaml) })
    } catch (failure) {
      setError(explain(failure))
    } finally {
      setBusy(null)
    }
  }

  async function confirm() {
    if (!pending) return
    setBusy('import')
    setError('')
    try {
      const result = await importBackup(pending.yaml)
      setPending(null)
      setDone(t('backup.done', { added: result.add.length, updated: result.update.length }))
      onImported()
    } catch (failure) {
      setError(explain(failure))
    } finally {
      setBusy(null)
    }
  }

  const preview = pending?.preview
  const blocked = Boolean(preview && preview.invalid.length > 0)
  const nothing = Boolean(preview && !preview.add.length && !preview.update.length && !preview.notifications)

  return (
    <>
      <h2 className={styles.groupTitle}>{t('backup.title')}</h2>
      <section className={styles.group}>
        <div className={styles.row}>
          <div className={styles.rowText}>
            <div className={styles.rowLabel}>{t('backup.export')}</div>
            {count !== null && <div className={styles.rowHint}>{t('backup.exportHint', { count })}</div>}
          </div>
          <ButtonAnchor href="/api/backup.yaml" download className={styles.surfaceButton}>
            {t('backup.exportButton')}
          </ButtonAnchor>
        </div>
        <div className={styles.row}>
          <div className={styles.rowText}>
            <div className={styles.rowLabel}>{t('backup.import')}</div>
            <div className={styles.rowHint}>{t('backup.importHint')}</div>
          </div>
          <input
            ref={input}
            type="file"
            accept=".yaml,.yml,application/yaml,text/yaml"
            hidden
            aria-label={t('backup.import')}
            onChange={(event) => void choose(event)}
          />
          <Button className={styles.surfaceButton} disabled={busy !== null} onClick={() => input.current?.click()}>
            {t(busy === 'read' ? 'backup.reading' : 'backup.choose')}
          </Button>
        </div>
        {pending && preview && (
          <div className={styles.preview} aria-live="polite">
            <p className={styles.previewTitle}>{t('backup.previewTitle', { file: pending.file })}</p>
            <ul className={styles.previewList}>
              {preview.add.length > 0 && (
                <li>{t('backup.add', { count: preview.add.length, names: preview.add.join(', ') })}</li>
              )}
              {preview.update.length > 0 && (
                <li>{t('backup.update', { count: preview.update.length, names: preview.update.join(', ') })}</li>
              )}
              {preview.unchanged > 0 && <li>{t('backup.unchanged', { count: preview.unchanged })}</li>}
              {preview.notifications && <li>{t('backup.notifications')}</li>}
              {nothing && !blocked && <li>{t('backup.nothing')}</li>}
              {preview.invalid.map((entry) => (
                <li key={entry.name} className={styles.previewBad}>
                  {t('backup.invalid', { name: entry.name, fields: Object.keys(entry.fields).join(', ') })}
                </li>
              ))}
            </ul>
            {blocked && <p className={styles.rowHint}>{t('backup.invalidHint')}</p>}
            <div className={styles.previewActions}>
              <Button variant="primary" disabled={blocked || nothing || busy !== null} onClick={() => void confirm()}>
                {t(busy === 'import' ? 'backup.importing' : 'backup.confirm')}
              </Button>
              <Button variant="ghost" disabled={busy !== null} onClick={() => setPending(null)}>
                {t('backup.cancel')}
              </Button>
            </div>
          </div>
        )}
        {error && (
          <p role="alert" className={styles.groupError}>
            {error}
          </p>
        )}
        {done && (
          <p role="status" className={styles.groupDone}>
            {done}
          </p>
        )}
      </section>
    </>
  )
}
