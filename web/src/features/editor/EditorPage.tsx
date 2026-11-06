import { useEffect, useRef, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ApiError, createMonitor, fetchMonitor, testMonitor, updateMonitor } from '../../api/client'
import type { CheckResult, ConditionType, HttpMethod } from '../../api/types'
import { Button, ButtonLink } from '../../components/Button'
import { Disclosure } from '../../components/Disclosure'
import { SegmentedControl } from '../../components/SegmentedControl'
import { Shell } from '../../components/Shell'
import { Stepper } from '../../components/Stepper'
import { FieldError, TextField } from '../../components/TextField'
import {
  emptyDraft,
  hostOf,
  isAdvancedField,
  mapServerFields,
  newCondition,
  OPS,
  toConfig,
  toDraft,
  uid,
  validate,
  type Draft,
  type DraftCondition,
} from './draft'
import styles from './EditorPage.module.css'
import { TestResult } from './TestResult'

const INTERVALS = [30, 60, 300, 900]
const METHODS: HttpMethod[] = ['GET', 'POST', 'PUT', 'HEAD']
const TYPES: ConditionType[] = ['status', 'response', 'body', 'json']

function RemoveIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 20 20" aria-hidden="true">
      <path d="M5 5l10 10M15 5 5 15" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" />
    </svg>
  )
}

export function EditorPage() {
  const { t } = useTranslation()
  const { id } = useParams()
  const navigate = useNavigate()

  const [draft, setDraft] = useState<Draft | null>(id ? null : emptyDraft)
  const [title, setTitle] = useState('')
  const [loadFailed, setLoadFailed] = useState(false)
  const [advanced, setAdvanced] = useState(false)
  const [shown, setShown] = useState<Set<string>>(new Set())
  const [serverErrors, setServerErrors] = useState<Record<string, string>>({})
  const [submitted, setSubmitted] = useState(false)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [result, setResult] = useState<CheckResult | null>(null)
  const [formError, setFormError] = useState('')
  const summaryRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!id) return
    fetchMonitor(id).then(
      (config) => {
        setDraft(toDraft(config))
        setTitle(config.name)
      },
      () => setLoadFailed(true),
    )
  }, [id])

  const back = id ? `/m/${id}` : '/'
  const backLink = (
    <Link to={back} className={styles.back}>
      <svg width="18" height="18" viewBox="0 0 20 20" aria-hidden="true">
        <path
          d="M12.5 4.5 7 10l5.5 5.5"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
      {t('editor.cancelLink')}
    </Link>
  )

  if (!draft) {
    return (
      <Shell maxWidth={640}>
        <div className={styles.form}>
          <div className={styles.topRow}>{backLink}</div>
          {loadFailed && <p className={styles.muted}>{t('monitor.notFound')}</p>}
        </div>
      </Shell>
    )
  }

  const current = draft
  const update = (patch: Partial<Draft>) => {
    setDraft({ ...current, ...patch })
    setServerErrors({})
  }
  const touch = (key: string) => setShown((keys) => new Set(keys).add(key))

  const allErrors = { ...validate(current), ...serverErrors }
  const visible = Object.fromEntries(
    Object.entries(allErrors).filter(([key]) => submitted || shown.has(key) || key in serverErrors),
  )
  const message = (key: string) =>
    visible[key] && t(`editor.error.${visible[key]}`, { interval: current.intervalSec })
  const errorCount = Object.keys(visible).length

  /** Hata varsa gösterir, gizli bölümü açar ve özete götürür. */
  function revealErrors(errors: Record<string, string>): boolean {
    const keys = Object.keys(errors)
    if (keys.length === 0) return false
    setSubmitted(true)
    if (keys.some(isAdvancedField)) setAdvanced(true)
    requestAnimationFrame(() => summaryRef.current?.scrollIntoView({ block: 'center', behavior: 'smooth' }))
    return true
  }

  async function runTest() {
    setFormError('')
    if (revealErrors(validate(current))) return
    setTesting(true)
    setResult(null)
    try {
      setResult(await testMonitor(toConfig(current)))
    } catch {
      setFormError(t('editor.testFailedRequest'))
    } finally {
      setTesting(false)
    }
  }

  async function save(event: FormEvent) {
    event.preventDefault()
    setFormError('')
    if (revealErrors(validate(current))) return
    setSaving(true)
    try {
      const config = toConfig(current)
      if (id) await updateMonitor(id, config)
      else await createMonitor(config)
      navigate(back)
    } catch (error) {
      const fields = error instanceof ApiError ? mapServerFields(error.fields, current) : {}
      if (Object.keys(fields).length) {
        setServerErrors(fields)
        revealErrors(fields)
      } else {
        setFormError(t('editor.saveFailed'))
      }
    } finally {
      setSaving(false)
    }
  }

  const setCondition = (uidValue: number, patch: Partial<DraftCondition>) =>
    update({ conditions: current.conditions.map((c) => (c.uid === uidValue ? { ...c, ...patch } : c)) })

  const threshold = current.alertThreshold * current.intervalSec
  const thresholdTime =
    threshold < 60
      ? t('editor.seconds', { count: threshold })
      : t('editor.minutes', { count: Math.round(threshold / 60) })

  const summary = id
    ? [
        current.method,
        t('editor.advancedTimeout', { count: Number(current.timeoutSec) }),
        t('editor.advancedHeaders', { count: current.headers.length }),
        t('editor.advancedConditions', { count: current.conditions.length }),
      ].join(' · ')
    : t('editor.advancedDefault')

  const httpsUrl = `https://${current.url.trim().replace(/^[a-z]+:\/\//i, '')}`

  return (
    <Shell maxWidth={640}>
      <form onSubmit={(event) => void save(event)} noValidate className={styles.form}>
        <div className={styles.topRow}>{backLink}</div>
        <h1 className={styles.title}>
          {id ? t('editor.editTitle', { name: title }) : t('editor.newTitle')}
        </h1>

        {errorCount > 0 && (
          <div ref={summaryRef} role="alert" className={styles.alert}>
            <svg width="18" height="18" viewBox="0 0 20 20" aria-hidden="true" className={styles.alertIcon}>
              <circle cx="10" cy="10" r="10" fill="var(--down)" />
              <path d="M10 5.6v5.2M10 14.3v.1" stroke="var(--on-status)" strokeWidth="2.2" strokeLinecap="round" />
            </svg>
            <span>
              <strong className={styles.strong}>{t('editor.errors', { count: errorCount })}</strong>{' '}
              {t('editor.errorsHint')}
            </span>
          </div>
        )}

        <section className={styles.card}>
          <TextField
            label={t('editor.url')}
            mono
            type="url"
            inputMode="url"
            autoCapitalize="none"
            spellCheck={false}
            placeholder="https://"
            value={current.url}
            onChange={(event) => update({ url: event.target.value })}
            onBlur={() => touch('url')}
            error={message('url') || undefined}
            errorAction={
              current.url.trim() && (
                <Button size="xs" onClick={() => update({ url: httpsUrl })}>
                  {t('editor.useHttps')}
                </Button>
              )
            }
          />
          <TextField
            label={t('editor.name')}
            hint={t('editor.optional')}
            placeholder={hostOf(current.url) || 'foto.home'}
            value={current.name}
            onChange={(event) => update({ name: event.target.value })}
          />
          <div className={styles.field}>
            <span className={styles.label}>{t('editor.interval')}</span>
            <SegmentedControl
              label={t('editor.interval')}
              value={String(current.intervalSec)}
              onChange={(value) => update({ intervalSec: Number(value) })}
              options={INTERVALS.map((sec) => ({
                value: String(sec),
                label: sec < 60 ? t('unit.s', { n: sec }) : t('unit.m', { n: sec / 60 }),
              }))}
            />
          </div>
          <div className={styles.field}>
            <span className={styles.label}>{t('editor.threshold')}</span>
            <div className={styles.thresholdRow}>
              <Stepper
                value={current.alertThreshold}
                min={1}
                max={10}
                onChange={(alertThreshold) => update({ alertThreshold })}
              />
              <span className={styles.thresholdHint}>{t('editor.thresholdHint', { time: thresholdTime })}</span>
            </div>
          </div>
        </section>

        <Disclosure title={t('editor.advanced')} summary={summary} open={advanced} onOpenChange={setAdvanced}>
          <div className={styles.advanced}>
            <section className={styles.card}>
              <h2 className={styles.cardTitle}>{t('editor.request')}</h2>
              <div className={styles.requestRow}>
                <div className={styles.methodField}>
                  <span className={styles.label}>{t('editor.method')}</span>
                  <SegmentedControl
                    mono
                    label={t('editor.method')}
                    value={current.method}
                    onChange={(method) => update({ method })}
                    options={METHODS.map((method) => ({ value: method, label: method }))}
                  />
                </div>
                <TextField
                  className={styles.timeoutField}
                  label={t('editor.timeout')}
                  inputMode="numeric"
                  suffix={t('editor.secondsUnit')}
                  value={current.timeoutSec}
                  onChange={(event) => update({ timeoutSec: event.target.value })}
                  onBlur={() => touch('timeoutSec')}
                  error={message('timeoutSec') || undefined}
                />
              </div>
              <div className={styles.headers}>
                <span className={styles.label}>{t('editor.headers')}</span>
                {current.headers.map((header) => (
                  <div key={header.uid} className={styles.headerRow}>
                    <input
                      aria-label={t('editor.headerName')}
                      className={styles.smallInput}
                      value={header.name}
                      onChange={(event) =>
                        update({
                          headers: current.headers.map((h) =>
                            h.uid === header.uid ? { ...h, name: event.target.value } : h,
                          ),
                        })
                      }
                    />
                    <input
                      aria-label={t('editor.headerValue')}
                      className={styles.smallInput}
                      value={header.value}
                      onChange={(event) =>
                        update({
                          headers: current.headers.map((h) =>
                            h.uid === header.uid ? { ...h, value: event.target.value } : h,
                          ),
                        })
                      }
                    />
                    <button
                      type="button"
                      aria-label={t('editor.headerRemove')}
                      className={styles.remove}
                      onClick={() => update({ headers: current.headers.filter((h) => h.uid !== header.uid) })}
                    >
                      <RemoveIcon />
                    </button>
                  </div>
                ))}
                <button
                  type="button"
                  className={styles.addRow}
                  onClick={() => update({ headers: [...current.headers, { uid: uid(), name: '', value: '' }] })}
                >
                  {t('editor.headerAdd')}
                </button>
              </div>
              <label className={styles.field}>
                <span className={styles.label}>{t('editor.body')}</span>
                <textarea
                  rows={3}
                  className={styles.textarea}
                  placeholder={
                    current.method === 'GET' || current.method === 'HEAD'
                      ? t('editor.bodyNone', { method: current.method })
                      : '{ "ping": true }'
                  }
                  value={current.body}
                  onChange={(event) => update({ body: event.target.value })}
                />
              </label>
            </section>

            <section className={`${styles.card} ${styles.conditions}`}>
              <div>
                <h2 className={styles.cardTitle}>{t('editor.conditions')}</h2>
                <p className={styles.cardHint}>{t('editor.conditionsHint')}</p>
              </div>
              {current.conditions.map((condition) => {
                const key = `cond:${condition.uid}`
                const error = message(key)
                const invalid = error ? styles.invalid : ''
                return (
                  <div key={condition.uid} className={styles.condition}>
                    <div className={styles.conditionRow}>
                      <select
                        aria-label={t('editor.conditionType')}
                        className={`${styles.select} ${styles.typeSelect}`}
                        value={condition.type}
                        onChange={(event) => {
                          const next = newCondition(event.target.value as ConditionType)
                          update({
                            conditions: current.conditions.map((c) => (c.uid === condition.uid ? next : c)),
                          })
                        }}
                      >
                        {TYPES.map((type) => (
                          <option key={type} value={type}>
                            {t(`editor.type.${type}`)}
                          </option>
                        ))}
                      </select>
                      {condition.type === 'json' && (
                        <input
                          aria-label={t('editor.conditionPath')}
                          aria-invalid={error ? true : undefined}
                          className={`${styles.smallInput} ${styles.grow} ${invalid}`}
                          value={condition.path ?? ''}
                          onChange={(event) => setCondition(condition.uid, { path: event.target.value })}
                          onBlur={() => touch(key)}
                        />
                      )}
                      <select
                        aria-label={t('editor.conditionOp')}
                        className={styles.select}
                        value={condition.op}
                        onChange={(event) => setCondition(condition.uid, { op: event.target.value })}
                      >
                        {OPS[condition.type].map((op) => (
                          <option key={op} value={op}>
                            {t(`editor.op.${op}`)}
                          </option>
                        ))}
                      </select>
                      <input
                        aria-label={t('editor.conditionValue')}
                        aria-invalid={error && condition.type !== 'json' ? true : undefined}
                        className={`${styles.smallInput} ${styles.grow} ${condition.type === 'json' ? '' : invalid}`}
                        value={condition.value}
                        onChange={(event) => setCondition(condition.uid, { value: event.target.value })}
                        onBlur={() => touch(key)}
                      />
                      <button
                        type="button"
                        aria-label={t('editor.conditionRemove')}
                        className={styles.remove}
                        onClick={() =>
                          update({ conditions: current.conditions.filter((c) => c.uid !== condition.uid) })
                        }
                      >
                        <RemoveIcon />
                      </button>
                    </div>
                    {error && <FieldError>{error}</FieldError>}
                  </div>
                )
              })}
              <button
                type="button"
                className={styles.addRow}
                onClick={() => update({ conditions: [...current.conditions, newCondition('body')] })}
              >
                {t('editor.conditionAdd')}
              </button>
            </section>
          </div>
        </Disclosure>

        {result && <TestResult result={result} />}
        {formError && (
          <p role="alert" className={styles.formError}>
            {formError}
          </p>
        )}

        <div className={styles.actions}>
          <Button size="md" disabled={testing} onClick={() => void runTest()}>
            {testing ? t('editor.testing') : t('editor.test')}
          </Button>
          <span className={styles.spacer} />
          <ButtonLink to={back} variant="ghost" size="md">
            {t('editor.cancel')}
          </ButtonLink>
          <Button type="submit" variant="primary" size="md" disabled={saving}>
            {saving ? t('editor.saving') : id ? t('editor.save') : t('editor.add')}
          </Button>
        </div>
      </form>
    </Shell>
  )
}
