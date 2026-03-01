export const createUrbitRuntimeCommands = ({ sendUrbitCommand }) => {
  const toggleBootStatus = patp => {
    return sendUrbitCommand('toggle-boot-status', { patp })
  }

  const toggleAutoReboot = patp => {
    return sendUrbitCommand('toggle-auto-reboot', { patp })
  }

  const toggleDevMode = patp => {
    return sendUrbitCommand('toggle-devmode', { patp })
  }

  const setMinIODomain = (patp, domain) => {
    return sendUrbitCommand('set-minio-domain', { patp, domain })
  }

  const setUrbitDomain = (patp, domain) => {
    return sendUrbitCommand('set-urbit-domain', { patp, domain })
  }

  const setNewMaxPierSize = (patp, size) => {
    return sendUrbitCommand('new-max-pier-size', { patp, value: size })
  }

  const toggleAutoBoot = patp => {
    return sendUrbitCommand('toggle-autoboot', { patp })
  }

  const toggleNetwork = patp => {
    return sendUrbitCommand('toggle-network', { patp })
  }

  const toggleMinIOLink = patp => {
    return sendUrbitCommand('toggle-minio-link', { patp })
  }

  const toggleUrbitPower = patp => {
    return sendUrbitCommand('toggle-power', { patp })
  }

  const deleteUrbitShip = patp => {
    return sendUrbitCommand('delete-ship', { patp })
  }

  const exportUrbitShip = patp => {
    return sendUrbitCommand('export-ship', { patp })
  }

  const exportUrbitBucket = patp => {
    return sendUrbitCommand('export-bucket', { patp })
  }

  const rebuildContainer = patp => {
    return sendUrbitCommand('rebuild-container', { patp })
  }

  return {
    toggleBootStatus,
    toggleAutoReboot,
    toggleDevMode,
    setMinIODomain,
    setUrbitDomain,
    setNewMaxPierSize,
    toggleAutoBoot,
    toggleNetwork,
    toggleMinIOLink,
    toggleUrbitPower,
    deleteUrbitShip,
    exportUrbitShip,
    exportUrbitBucket,
    rebuildContainer
  }
}
