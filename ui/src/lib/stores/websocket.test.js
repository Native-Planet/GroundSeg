import { describe, expect, it } from 'vitest'
import { mergeContainerLogLine, safeParseSocketMessage } from './websocket.js'

describe('websocket helpers', () => {
  it('safely parses valid JSON messages', () => {
    expect(safeParseSocketMessage('{"type":"activity"}')).toEqual({ type: 'activity' })
  })

  it('returns null for invalid JSON messages', () => {
    expect(safeParseSocketMessage('invalid')).toBeNull()
  })

  it('merges container logs immutably', () => {
    const first = mergeContainerLogLine({}, 'abc', 'line-1')
    const second = mergeContainerLogLine(first, 'abc', 'line-2')

    expect(first.abc).toBe('line-1')
    expect(second.abc).toBe('line-1\nline-2')
  })
})
