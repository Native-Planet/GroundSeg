export const createUrbitBackupCommands = ({ sendUrbitCommand }) => {
  const toggleBackups = patp => {
    return sendUrbitCommand('toggle-backup', { patp })
  }

  const toggleStartramBackups = patp => {
    return sendUrbitCommand('toggle-startram-backup', { patp })
  }

  const scheduleLocalBackup = (patp, backupTime) => {
    return sendUrbitCommand('schedule-local-backup', { patp, backupTime })
  }

  const localBackup = patp => {
    return sendUrbitCommand('local-backup', { patp })
  }

  const startramBackup = patp => {
    return sendUrbitCommand('startram-backup', { patp })
  }

  const restoreTlonBackup = (patp, remote, timestamp, md5, bakType) => {
    return sendUrbitCommand('restore-tlon-backup', { patp, remote, timestamp, md5, bakType })
  }

  const registerServiceAgain = patp => {
    return sendUrbitCommand('register-service-again', { patp })
  }

  const restoreBackup = (patp, backupFile) => {
    return sendUrbitCommand('restore-backup', { patp, file: backupFile })
  }

  const setStartramReminder = (patp, remind) => {
    return sendUrbitCommand('startram-reminder', { patp, remind })
  }

  return {
    toggleBackups,
    toggleStartramBackups,
    scheduleLocalBackup,
    localBackup,
    startramBackup,
    restoreTlonBackup,
    registerServiceAgain,
    restoreBackup,
    setStartramReminder
  }
}
