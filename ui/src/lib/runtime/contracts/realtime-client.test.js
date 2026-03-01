import { describe, expect, it, vi } from 'vitest'
import { createWebSocketClient } from '../transport/websocket-client.js'

describe('realtime client contract', () => {
  it('websocket client conforms to the realtime client contract shape', () => {
    const originalWebSocket = globalThis.WebSocket

    class FakeWebSocket {
      static OPEN = 1

      constructor() {
        this.readyState = FakeWebSocket.OPEN
      }

      send = vi.fn()
      close = vi.fn()
    }

    globalThis.WebSocket = FakeWebSocket

    try {
      const client = createWebSocketClient({ url: 'ws://localhost:3000/ws' })
      client.connect()

      expect(typeof client.connect).toBe('function')
      expect(typeof client.disconnect).toBe('function')
      expect(typeof client.send).toBe('function')
      expect(typeof client.isOpen).toBe('function')
      expect(typeof client.getReadyState).toBe('function')
      expect(client.send('hello')).toEqual({ ok: true })
    } finally {
      globalThis.WebSocket = originalWebSocket
    }
  })
})
