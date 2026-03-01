import { describe, expect, it, vi } from 'vitest'
import { createAuthCommands } from './auth-commands.js'
import { createDevCommands } from './dev-commands.js'
import { createLogCommands } from './log-commands.js'
import { createNetworkCommands } from './network-commands.js'
import { createPenpaiCommands } from './penpai-commands.js'
import { createSetupCommands } from './setup-commands.js'
import { createShipCommands } from './ship-commands.js'
import { createStartramCommands } from './startram-commands.js'
import { createSupportCommands } from './support-commands.js'
import { createSystemCommands } from './system-commands.js'
import { createUrbitBackupCommands } from './urbit-backup-commands.js'
import { createUrbitCommands } from './urbit-commands.js'
import { createUrbitMaintenanceCommands } from './urbit-maintenance-commands.js'
import { createUrbitRuntimeCommands } from './urbit-runtime-commands.js'

const makeContext = () => {
  return {
    sendTypeCommand: vi.fn(),
    sendActionCommand: vi.fn(),
    sendSystemCommand: vi.fn(),
    sendSetupCommand: vi.fn(),
    sendStartramCommand: vi.fn(),
    sendUrbitCommand: vi.fn(),
    sendDevCommand: vi.fn(),
    sendValidatedCommand: vi.fn()
  }
}

describe('domain command modules', () => {
  it('auth commands clear login errors and send auth payloads', () => {
    const context = makeContext()
    const clearLoginError = vi.fn()
    const commands = createAuthCommands(context, { clearLoginError })

    commands.login('secret')

    expect(clearLoginError).toHaveBeenCalledTimes(1)
    expect(context.sendTypeCommand).toHaveBeenCalledWith('login', { password: 'secret' })
  })

  it('setup commands map to setup and startram actions', () => {
    const context = makeContext()
    const commands = createSetupCommands(context)

    commands.setupStarTram('key', 'us-east')
    commands.startramBackupPassword('pw')

    expect(context.sendSetupCommand).toHaveBeenCalledWith('startram', { key: 'key', region: 'us-east' })
    expect(context.sendStartramCommand).toHaveBeenCalledWith('set-backup-password', { password: 'pw' })
  })

  it('system commands route through system transport helpers', () => {
    const context = makeContext()
    const commands = createSystemCommands(context)

    commands.toggleWifi()

    expect(context.sendSystemCommand).toHaveBeenCalledWith('wifi-toggle')
  })

  it('startram commands dispatch registration actions', () => {
    const context = makeContext()
    const commands = createStartramCommands(context)

    commands.startramRegister('key', 'region')

    expect(context.sendStartramCommand).toHaveBeenCalledWith('register', {
      key: 'key',
      region: 'region'
    })
  })

  it('ship commands use validated command sending for boot', () => {
    const context = makeContext()
    const commands = createShipCommands(context)

    commands.bootShip('~zod', 'ticket', 'master-ticket', true, 'system-drive')

    expect(context.sendValidatedCommand).toHaveBeenCalledWith({
      commandName: 'bootShip',
      type: 'new_ship',
      action: 'boot',
      payload: {
        patp: '~zod',
        keyType: 'master-ticket',
        key: 'ticket',
        remote: true,
        selectedDrive: 'system-drive'
      },
      requiredFields: ['patp', 'keyType', 'key']
    })
  })

  it('urbit commands map to urbit action payloads', () => {
    const context = makeContext()
    const commands = createUrbitCommands(context)

    commands.toggleBackups('~zod')

    expect(context.sendUrbitCommand).toHaveBeenCalledWith('toggle-backup', { patp: '~zod' })
  })

  it('urbit command subdomains expose backup/runtime/maintenance helpers', () => {
    const context = makeContext()
    const backupCommands = createUrbitBackupCommands(context)
    const runtimeCommands = createUrbitRuntimeCommands(context)
    const maintenanceCommands = createUrbitMaintenanceCommands(context)

    backupCommands.localBackup('~zod')
    runtimeCommands.toggleUrbitPower('~zod')
    maintenanceCommands.setUrbitLoom('~zod', 31)

    expect(context.sendUrbitCommand).toHaveBeenCalledWith('local-backup', { patp: '~zod' })
    expect(context.sendUrbitCommand).toHaveBeenCalledWith('toggle-power', { patp: '~zod' })
    expect(context.sendUrbitCommand).toHaveBeenCalledWith('loom', { patp: '~zod', value: 31 })
  })

  it('support commands include cpu_profile wire key', () => {
    const context = makeContext()
    const commands = createSupportCommands(context)

    commands.submitReport('me', 'desc', ['~zod'], true, false)

    expect(context.sendActionCommand).toHaveBeenCalledWith('support', 'bug-report', {
      contact: 'me',
      description: 'desc',
      ships: ['~zod'],
      cpu_profile: true,
      penpai: false
    })
  })

  it('log and network commands target expected request envelopes', () => {
    const context = makeContext()
    const logCommands = createLogCommands(context)
    const networkCommands = createNetworkCommands(context)

    logCommands.toggleLog('groundseg', 'subscribe')
    networkCommands.submitNetwork('ssid', 'pw')

    expect(context.sendActionCommand).toHaveBeenCalledWith('logs', 'subscribe', {
      container_id: 'groundseg'
    })
    expect(context.sendTypeCommand).toHaveBeenCalledWith('c2c', {
      ssid: 'ssid',
      password: 'pw'
    })
  })

  it('penpai and dev commands map to dedicated domains', () => {
    const context = makeContext()
    const penpaiCommands = createPenpaiCommands(context)
    const devCommands = createDevCommands(context)

    penpaiCommands.setPenpaiCores(4)
    devCommands.devRestoreTlon('~zod', true)

    expect(context.sendActionCommand).toHaveBeenCalledWith('penpai', 'set-cores', { cores: 4 })
    expect(context.sendDevCommand).toHaveBeenCalledWith('restore-tlon', {
      patp: '~zod',
      remote: true
    })
  })
})
