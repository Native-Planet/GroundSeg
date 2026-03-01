export const createStartramCommands = ({ sendStartramCommand }) => {
  const startramGetRegions = () => {
    return sendStartramCommand('regions')
  }

  const startramGetServices = () => {
    return sendStartramCommand('services')
  }

  const startramRegister = (key, region) => {
    return sendStartramCommand('register', { key, region })
  }

  const startramToggle = () => {
    return sendStartramCommand('toggle')
  }

  const startramRestart = () => {
    return sendStartramCommand('restart')
  }

  const startramEndpoint = endpoint => {
    return sendStartramCommand('endpoint', { endpoint })
  }

  const startramCancel = (key, reset) => {
    return sendStartramCommand('cancel', { key, reset })
  }

  const setAllStartramReminder = remind => {
    return sendStartramCommand('reminder', { remind })
  }

  return {
    startramGetRegions,
    startramGetServices,
    startramRegister,
    startramToggle,
    startramRestart,
    startramEndpoint,
    startramCancel,
    setAllStartramReminder
  }
}
