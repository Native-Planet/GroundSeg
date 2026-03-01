import { describe, expect, it } from 'vitest'
import { alive, lastActivity } from './alive.js'

describe('alive route store', () => {
  it('exports readable and writable stores', () => {
    expect(alive).toBeDefined()
    expect(lastActivity).toBeDefined()
  })
})
