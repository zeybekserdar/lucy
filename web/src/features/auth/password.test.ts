import { describe, expect, it } from 'vitest'
import { passwordIssue } from './password'

describe('passwordIssue', () => {
  it('kısa şifreyi önce bildirir', () => {
    expect(passwordIssue('kısa', 'başka')).toBe('tooShort')
  })

  it('eşleşmeyen alanları bildirir', () => {
    expect(passwordIssue('doğru-at-pili', 'doğru-at-pil')).toBe('mismatch')
  })

  it('geçerli şifreyi kabul eder', () => {
    expect(passwordIssue('doğru-at-pili', 'doğru-at-pili')).toBeNull()
  })

  it('Türkçe harfleri tek karakter sayar', () => {
    expect(passwordIssue('şşşşşşşşşşşş', 'şşşşşşşşşşşş')).toBeNull()
    expect(passwordIssue('şşşşşşşşşşş', 'şşşşşşşşşşş')).toBe('tooShort')
  })
})
