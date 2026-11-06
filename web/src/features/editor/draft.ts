import type { Condition, ConditionType, HttpMethod, MonitorConfig } from '../../api/types'

export interface DraftHeader {
  uid: number
  name: string
  value: string
}

export interface DraftCondition extends Condition {
  uid: number
}

export interface Draft {
  url: string
  name: string
  intervalSec: number
  alertThreshold: number
  method: HttpMethod
  timeoutSec: string
  headers: DraftHeader[]
  body: string
  conditions: DraftCondition[]
}

export const OPS: Record<ConditionType, string[]> = {
  status: ['between', 'equals'],
  response: ['below'],
  body: ['contains', 'not_contains'],
  json: ['equals', 'contains'],
}

let nextUid = 1
export const uid = () => nextUid++

export function newCondition(type: ConditionType): DraftCondition {
  const base = { uid: uid(), type, op: OPS[type][0] }
  switch (type) {
    case 'status':
      return { ...base, value: '200–299' }
    case 'response':
      return { ...base, value: '2000' }
    case 'body':
      return { ...base, value: '' }
    case 'json':
      return { ...base, path: '$.', value: '' }
  }
}

export function emptyDraft(): Draft {
  return {
    url: '',
    name: '',
    intervalSec: 60,
    alertThreshold: 3,
    method: 'GET',
    timeoutSec: '10',
    headers: [],
    body: '',
    conditions: [newCondition('status')],
  }
}

export function toDraft(config: MonitorConfig): Draft {
  return {
    url: config.url,
    name: config.name,
    intervalSec: config.intervalSec,
    alertThreshold: config.alertThreshold,
    method: config.method,
    timeoutSec: String(config.timeoutSec),
    headers: config.headers.map((header) => ({ ...header, uid: uid() })),
    body: config.body,
    conditions: config.conditions.map((condition) => ({ ...condition, uid: uid() })),
  }
}

export function toConfig(draft: Draft): MonitorConfig {
  return {
    url: draft.url.trim(),
    name: draft.name.trim(),
    intervalSec: draft.intervalSec,
    alertThreshold: draft.alertThreshold,
    method: draft.method,
    timeoutSec: Number(draft.timeoutSec),
    headers: draft.headers.map(({ name, value }) => ({ name, value })),
    body: draft.body,
    conditions: draft.conditions.map(({ type, op, path, value }) => ({ type, op, path, value })),
  }
}

export function hostOf(url: string): string {
  try {
    return new URL(url).host
  } catch {
    return ''
  }
}

const conditionKey = (condition: DraftCondition) => `cond:${condition.uid}`

function validateCondition(c: DraftCondition): string | null {
  const value = c.value.trim()
  switch (c.type) {
    case 'status': {
      const pattern = c.op === 'between' ? /^\d+\s*[-–]\s*\d+$/ : /^\d+$/
      if (!pattern.test(value)) return 'status_value'
      const [low, high] = value.split(/[-–]/).map(Number)
      return high !== undefined && low > high ? 'status_value' : null
    }
    case 'response':
      return /^[1-9]\d*(\s*ms)?$/.test(value) ? null : 'response_value'
    case 'body':
      return c.value ? null : 'value_required'
    case 'json':
      return c.path?.startsWith('$') ? null : 'json_path'
  }
}

/** Hatalı alanları kod olarak döndürür: url, timeoutSec, cond:<uid>. */
export function validate(draft: Draft): Record<string, string> {
  const errors: Record<string, string> = {}
  if (!/^https?:\/\/[^\s/]+/i.test(draft.url.trim())) errors.url = 'url_scheme'
  const timeout = Number(draft.timeoutSec)
  if (!Number.isInteger(timeout) || timeout < 1 || timeout >= draft.intervalSec) {
    errors.timeoutSec = 'timeout_range'
  }
  for (const condition of draft.conditions) {
    const code = validateCondition(condition)
    if (code) errors[conditionKey(condition)] = code
  }
  return errors
}

/** Sunucunun conditions.N alanlarını formun anahtarlarına çevirir. */
export function mapServerFields(fields: Record<string, string>, draft: Draft): Record<string, string> {
  const errors: Record<string, string> = {}
  for (const [field, code] of Object.entries(fields)) {
    const match = /^conditions\.(\d+)$/.exec(field)
    const condition = match ? draft.conditions[Number(match[1])] : undefined
    errors[condition ? conditionKey(condition) : field] = code
  }
  return errors
}

export function isAdvancedField(key: string): boolean {
  return key !== 'url'
}
