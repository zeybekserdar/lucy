import { describe, expect, it } from 'vitest'
import { generateTopic } from './topic'

describe('generateTopic', () => {
  it('fits what the server accepts as a topic', () => {
    expect(generateTopic()).toMatch(/^lucy-[0-9a-f]{24}$/)
  })
  it('never repeats', () => {
    expect(new Set(Array.from({ length: 200 }, generateTopic)).size).toBe(200)
  })
})
