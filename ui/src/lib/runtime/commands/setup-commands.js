export const createSetupCommands = ({ sendSetupCommand, sendStartramCommand }) => {
  const beginSetup = () => {
    return sendSetupCommand('begin')
  }

  const setupPassword = pwd => {
    return sendSetupCommand('password', { password: pwd })
  }

  const setupSkip = () => {
    return sendSetupCommand('skip')
  }

  const setupStarTram = (key, region) => {
    return sendSetupCommand('startram', { key, region })
  }

  const startramBackupPassword = password => {
    return sendStartramCommand('set-backup-password', { password })
  }

  return {
    beginSetup,
    setupPassword,
    setupSkip,
    setupStarTram,
    startramBackupPassword
  }
}
