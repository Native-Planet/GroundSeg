export const createDevCommands = ({ sendDevCommand }) => {
  const resetSetup = () => {
    return sendDevCommand('reset-setup')
  }

  const printMounts = () => {
    return sendDevCommand('print-mounts')
  }

  const devStartramReminder = () => {
    return sendDevCommand('startram-reminder')
  }

  const devStartramReminderToggle = reminded => {
    return sendDevCommand('startram-reminder-toggle', { reminded })
  }

  const devBackupTlon = () => {
    return sendDevCommand('backup-tlon')
  }

  const devRemoteBackupTlon = () => {
    return sendDevCommand('remote-backup-tlon')
  }

  const devRestoreTlon = (patp, remote) => {
    return sendDevCommand('restore-tlon', { patp, remote })
  }

  return {
    resetSetup,
    printMounts,
    devStartramReminder,
    devStartramReminderToggle,
    devBackupTlon,
    devRemoteBackupTlon,
    devRestoreTlon
  }
}
