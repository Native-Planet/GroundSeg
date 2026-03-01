import { createUrbitBackupCommands } from './urbit-backup-commands.js'
import { createUrbitMaintenanceCommands } from './urbit-maintenance-commands.js'
import { createUrbitRuntimeCommands } from './urbit-runtime-commands.js'

export const createUrbitCommands = context => {
  return {
    ...createUrbitBackupCommands(context),
    ...createUrbitRuntimeCommands(context),
    ...createUrbitMaintenanceCommands(context)
  }
}
