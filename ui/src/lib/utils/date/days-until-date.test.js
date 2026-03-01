import { describe, expect, it } from 'vitest'
import { daysUntilDate } from './days-until-date.js'

describe('daysUntilDate', () => {
  it('returns positive values for future dates', () => {
    const days = daysUntilDate('2099-01-01T00:00:00.000Z')
    expect(days).toBeGreaterThan(0)
  })

  it('returns 0 for past dates', () => {
    const days = daysUntilDate('2000-01-01T00:00:00.000Z')
    expect(days).toBe(0)
  })
})
