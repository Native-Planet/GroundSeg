export const createSystemCommands = ({ sendSystemCommand }) => {
  const restartGroundSeg = () => {
    return sendSystemCommand('groundseg', { command: 'restart' })
  }

  const restartDevice = () => {
    return sendSystemCommand('power', { command: 'restart' })
  }

  const shutdownDevice = () => {
    return sendSystemCommand('power', { command: 'shutdown' })
  }

  const updateLinux = () => {
    return sendSystemCommand('update', { update: 'linux' })
  }

  const setSwap = val => {
    return sendSystemCommand('modify-swap', { value: val })
  }

  const toggleWifi = () => {
    return sendSystemCommand('wifi-toggle')
  }

  const connectWifi = (ssid, pwd) => {
    return sendSystemCommand('wifi-connect', { ssid, password: pwd })
  }

  const toggleExperimentalPenpai = () => {
    return sendSystemCommand('toggle-penpai-feature')
  }

  return {
    restartGroundSeg,
    restartDevice,
    shutdownDevice,
    updateLinux,
    setSwap,
    toggleWifi,
    connectWifi,
    toggleExperimentalPenpai
  }
}
