export const createPenpaiCommands = ({ sendActionCommand }) => {
  const togglePenpai = () => {
    return sendActionCommand('penpai', 'toggle')
  }

  const setPenpaiModel = model => {
    return sendActionCommand('penpai', 'set-model', { model })
  }

  const setPenpaiCores = cores => {
    return sendActionCommand('penpai', 'set-cores', { cores })
  }

  const removePenpai = () => {
    return sendActionCommand('penpai', 'remove')
  }

  return {
    togglePenpai,
    setPenpaiModel,
    setPenpaiCores,
    removePenpai
  }
}
