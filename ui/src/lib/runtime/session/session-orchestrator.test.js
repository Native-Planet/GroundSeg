import { describe, expect, it, vi } from 'vitest'
import { startRuntimeSession } from './session-orchestrator.js'

describe('startRuntimeSession', () => {
  it('connects websocket in non-urbit mode', () => {
    const connectWebsocket = vi.fn()
    const subscribeUrbit = vi.fn()

    const result = startRuntimeSession({
      pageUrl: new URL('https://example.com/system'),
      wsPort: '3000',
      urbitModeEnabled: false,
      customHostname: null,
      connectWebsocket,
      subscribeUrbit
    })

    expect(connectWebsocket).toHaveBeenCalledWith('ws://example.com:3000/ws')
    expect(subscribeUrbit).not.toHaveBeenCalled()
    expect(result.transport).toBe('websocket')
  })

  it('subscribes urbit in urbit mode', () => {
    const connectWebsocket = vi.fn()
    const subscribeUrbit = vi.fn()

    const result = startRuntimeSession({
      pageUrl: new URL('https://example.com/system'),
      wsPort: '3000',
      urbitModeEnabled: true,
      customHostname: null,
      connectWebsocket,
      subscribeUrbit,
      ship: '~zod'
    })

    expect(connectWebsocket).not.toHaveBeenCalled()
    expect(subscribeUrbit).toHaveBeenCalledWith('~zod')
    expect(result.transport).toBe('urbit')
  })
})
