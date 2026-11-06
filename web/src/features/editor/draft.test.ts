import { describe, expect, it } from 'vitest'
import { emptyDraft, hostOf, mapServerFields, newCondition, toConfig, toDraft, validate } from './draft'

const draft = (patch = {}) => ({ ...emptyDraft(), url: 'https://foto.home', ...patch })

describe('validate', () => {
  it('varsayılan taslağı kabul eder', () => {
    expect(validate(draft())).toEqual({})
  })

  it('şemasız adresi reddeder', () => {
    expect(validate(draft({ url: 'nas.home:5001' })).url).toBe('url_scheme')
    expect(validate(draft({ url: 'https://' })).url).toBe('url_scheme')
  })

  it('zaman aşımı aralıktan kısa olmalı', () => {
    expect(validate(draft({ timeoutSec: '60' })).timeoutSec).toBe('timeout_range')
    expect(validate(draft({ timeoutSec: 'abc' })).timeoutSec).toBe('timeout_range')
    expect(validate(draft({ timeoutSec: '59' })).timeoutSec).toBeUndefined()
  })

  it.each([
    ['status', { value: '200–299' }, null],
    ['status', { value: '200-299' }, null],
    ['status', { value: '299-200' }, 'status_value'],
    ['status', { value: 'iki yüz' }, 'status_value'],
    ['response', { value: '2000 ms' }, null],
    ['response', { value: '0' }, 'response_value'],
    ['body', { value: '' }, 'value_required'],
    ['json', { path: 'status' }, 'json_path'],
    ['json', { path: '$.status' }, null],
  ] as const)('%s %j → %s', (type, patch, code) => {
    const condition = { ...newCondition(type), ...patch }
    const errors = validate(draft({ conditions: [condition] }))
    expect(errors[`cond:${condition.uid}`] ?? null).toBe(code)
  })
})

describe('dönüşümler', () => {
  it('taslak ve yapılandırma birbirine döner', () => {
    const config = toConfig(draft({ headers: [{ uid: 1, name: 'X', value: 'y' }], timeoutSec: '5' }))
    expect(config.timeoutSec).toBe(5)
    expect(config.headers).toEqual([{ name: 'X', value: 'y' }])
    expect(toConfig(toDraft(config))).toEqual(config)
  })

  it('sunucu koşul hatasını doğru koşula bağlar', () => {
    const value = draft()
    const [first] = value.conditions
    expect(mapServerFields({ 'conditions.0': 'condition', url: 'url_scheme' }, value)).toEqual({
      [`cond:${first.uid}`]: 'condition',
      url: 'url_scheme',
    })
  })

  it('geçersiz adresten host çıkarmaz', () => {
    expect(hostOf('https://foto.home:8096/a')).toBe('foto.home:8096')
    expect(hostOf('foto')).toBe('')
  })
})
