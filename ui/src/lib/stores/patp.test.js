import { describe, expect, it } from 'vitest'
import { checkPatp, sigRemove } from './patp.js'

describe('patp helpers', () => {
  it('removes leading sigils', () => {
    expect(sigRemove('~zod')).toBe('zod')
  })

  it('accepts valid patp values', () => {
    expect(checkPatp('zod')).toBe(true)
    expect(checkPatp('marzod')).toBe(true)
  })

  it('rejects malformed patp values', () => {
    expect(checkPatp('')).toBe(false)
    expect(checkPatp('not-a-patp')).toBe(false)
  })
})
