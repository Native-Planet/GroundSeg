import { describe, expect, it } from 'vitest'
import { resolveWebsocketUrl } from './url.js'

describe('resolveWebsocketUrl', () => {
  it('builds websocket urls from hostname and port', () => {
    const url = resolveWebsocketUrl({
      hostname: 'groundseg.local',
      port: '3000',
      path: '/ws'
    })

    expect(url).toBe('ws://groundseg.local:3000/ws')
  })

  it('prefers custom hostname when provided', () => {
    const url = resolveWebsocketUrl({
      hostname: 'ignored.local',
      customHostname: 'proxy.local',
      port: '3000',
      path: 'logs'
    })

    expect(url).toBe('ws://proxy.local:3000/logs')
  })
})
