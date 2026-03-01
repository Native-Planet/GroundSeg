export const createNetworkCommands = ({ sendTypeCommand }) => {
  const submitNetwork = (ssid, password) => {
    return sendTypeCommand('c2c', { ssid, password })
  }

  return {
    submitNetwork
  }
}
