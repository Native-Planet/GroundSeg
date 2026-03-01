export const createShipCommands = ({ sendActionCommand, sendValidatedCommand }) => {
  const openUploadEndpoint = (endpoint, remote, fix, selectedDrive) => {
    return sendActionCommand('pier_upload', 'open-endpoint', {
      endpoint,
      remote,
      fix,
      selectedDrive
    })
  }

  const modifyUploadEndpoint = (endpoint, remote, fix) => {
    return sendActionCommand('pier_upload', 'modify-endpoint', {
      endpoint,
      remote,
      fix
    })
  }

  const closeUploadEndpoint = endpoint => {
    return sendActionCommand('pier_upload', 'close-endpoint', { endpoint })
  }

  const resetImportShip = () => {
    return sendActionCommand('pier_upload', 'reset')
  }

  const cancelNewShip = patp => {
    return sendActionCommand('new_ship', 'cancel', { patp })
  }

  const bootShip = (patp, key, keyType, remote, selectedDrive) => {
    return sendValidatedCommand({
      commandName: 'bootShip',
      type: 'new_ship',
      action: 'boot',
      payload: {
        patp,
        keyType,
        key,
        remote,
        selectedDrive
      },
      requiredFields: ['patp', 'keyType', 'key']
    })
  }

  const resetNewShip = () => {
    return sendActionCommand('new_ship', 'reset')
  }

  return {
    openUploadEndpoint,
    modifyUploadEndpoint,
    closeUploadEndpoint,
    resetImportShip,
    cancelNewShip,
    bootShip,
    resetNewShip
  }
}
