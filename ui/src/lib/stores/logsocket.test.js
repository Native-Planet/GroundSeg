import { describe, expect, it } from 'vitest'
import {
  appendLogMessage,
  removeLogType,
  replaceLogHistory,
  safeParseLogSocketMessage
} from './logsocket.js'

describe('logsocket helpers', () => {
  it('safely parses valid JSON payloads', () => {
    expect(safeParseLogSocketMessage('{"ok":true}')).toEqual({ ok: true })
  })

  it('returns null for invalid payloads', () => {
    expect(safeParseLogSocketMessage('{bad-json')).toBeNull()
  })

  it('updates log collections immutably', () => {
    const first = replaceLogHistory({}, 'system', ['line-1'])
    const second = appendLogMessage(first, 'system', 'line-2')
    const third = removeLogType(second, 'system')

    expect(first.system).toEqual(['line-1'])
    expect(second.system).toEqual(['line-1', 'line-2'])
    expect(third.system).toBeUndefined()
  })
})
