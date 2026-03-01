import {
  createAuthCommands
} from '../runtime/commands/auth-commands.js'
import { createCommandContext } from '../runtime/commands/context.js'
import { createDevCommands } from '../runtime/commands/dev-commands.js'
import { createLogCommands } from '../runtime/commands/log-commands.js'
import { createNetworkCommands } from '../runtime/commands/network-commands.js'
import { createPenpaiCommands } from '../runtime/commands/penpai-commands.js'
import { createSetupCommands } from '../runtime/commands/setup-commands.js'
import { createShipCommands } from '../runtime/commands/ship-commands.js'
import { createStartramCommands } from '../runtime/commands/startram-commands.js'
import { createSupportCommands } from '../runtime/commands/support-commands.js'
import { createSystemCommands } from '../runtime/commands/system-commands.js'
import { createUrbitCommands } from '../runtime/commands/urbit-commands.js'

export const createWebsocketCommandDomains = ({ sendCommand, clearLoginError }) => {
  const context = createCommandContext(sendCommand)

  return {
    auth: createAuthCommands(context, { clearLoginError }),
    setup: createSetupCommands(context),
    system: createSystemCommands(context),
    startram: createStartramCommands(context),
    ship: createShipCommands(context),
    urbit: createUrbitCommands(context),
    support: createSupportCommands(context),
    logs: createLogCommands(context),
    network: createNetworkCommands(context),
    penpai: createPenpaiCommands(context),
    dev: createDevCommands(context)
  }
}

export const createWebsocketCommands = deps => {
  const domains = createWebsocketCommandDomains(deps)

  return {
    ...domains.auth,
    ...domains.setup,
    ...domains.system,
    ...domains.startram,
    ...domains.ship,
    ...domains.urbit,
    ...domains.support,
    ...domains.logs,
    ...domains.network,
    ...domains.penpai,
    ...domains.dev
  }
}
