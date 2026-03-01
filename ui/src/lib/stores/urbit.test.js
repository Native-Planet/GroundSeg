import { describe, expect, it } from 'vitest'
import { parseBroadcastCord } from './urbit.js'

describe('urbit helpers', () => {
  it('parses valid broadcast payloads', () => {
    expect(parseBroadcastCord('{"type":"init"}')).toEqual({ type: 'init' })
  })

  it('returns null for invalid cord values', () => {
    expect(parseBroadcastCord('{invalid-json')).toBeNull()
  })
})
