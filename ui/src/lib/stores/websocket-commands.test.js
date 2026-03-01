import { describe, expect, it, vi } from 'vitest'
import { createWebsocketCommandDomains, createWebsocketCommands } from './websocket-commands.js'

describe('websocket command factory', () => {
  it('builds domain-specific command groups', () => {
    const sendCommand = vi.fn()
    const clearLoginError = vi.fn()
    const domains = createWebsocketCommandDomains({ sendCommand, clearLoginError })

    domains.auth.login('secret')
    domains.system.toggleWifi()
    domains.urbit.toggleBackups('~zod')

    expect(clearLoginError).toHaveBeenCalledTimes(1)
    expect(sendCommand).toHaveBeenCalledWith({ type: 'login', password: 'secret' })
    expect(sendCommand).toHaveBeenCalledWith({ type: 'system', action: 'wifi-toggle' })
    expect(sendCommand).toHaveBeenCalledWith({ type: 'urbit', action: 'toggle-backup', patp: '~zod' })
  })

  it('provides a compatibility facade for legacy callers', () => {
    const sendCommand = vi.fn()
    const clearLoginError = vi.fn()
    const commands = createWebsocketCommands({ sendCommand, clearLoginError })

    commands.login('secret')
    commands.toggleWifi()
    commands.toggleBackups('~zod')
    commands.bootShip('~zod', 'ticket', 'master-ticket', true, 'system-drive')

    expect(sendCommand).toHaveBeenCalledTimes(4)
    expect(sendCommand).toHaveBeenCalledWith({
      type: 'new_ship',
      action: 'boot',
      patp: '~zod',
      keyType: 'master-ticket',
      key: 'ticket',
      remote: true,
      selectedDrive: 'system-drive'
    })
  })
})
