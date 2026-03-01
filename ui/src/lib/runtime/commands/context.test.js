import { describe, expect, it, vi } from 'vitest'
import { createCommandContext } from './context.js'

describe('createCommandContext', () => {
  it('sends action commands through the provided transport', () => {
    const sendCommand = vi.fn()
    const context = createCommandContext(sendCommand)

    context.sendActionCommand('system', 'wifi-toggle')

    expect(sendCommand).toHaveBeenCalledWith({ type: 'system', action: 'wifi-toggle' })
  })

  it('validates required fields for structured commands', () => {
    const sendCommand = vi.fn()
    const context = createCommandContext(sendCommand)

    expect(() => {
      context.sendValidatedCommand({
        commandName: 'bootShip',
        type: 'new_ship',
        action: 'boot',
        payload: {
          patp: '~zod',
          key: 'ticket'
        },
        requiredFields: ['patp', 'key', 'keyType']
      })
    }).toThrow('bootShip is missing required field(s): keyType')
  })
})
