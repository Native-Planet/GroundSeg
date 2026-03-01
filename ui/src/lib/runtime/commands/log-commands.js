export const createLogCommands = ({ sendActionCommand }) => {
  const toggleLog = (name, action) => {
    return sendActionCommand('logs', action, { container_id: name })
  }

  return {
    toggleLog
  }
}
