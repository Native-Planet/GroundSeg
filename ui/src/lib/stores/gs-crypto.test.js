import { describe, expect, it } from 'vitest'
import { generateRandom, toBase64 } from './gs-crypto.js'

describe('gs-crypto helpers', () => {
  it('encodes bytes to base64', () => {
    const input = new Uint8Array([104, 105])
    expect(toBase64(input)).toBe('aGk=')
  })

  it('generates requested-length hex tokens', () => {
    const token = generateRandom(32)
    expect(token).toHaveLength(32)
    expect(token).toMatch(/^[0-9a-f]+$/)
  })
})
