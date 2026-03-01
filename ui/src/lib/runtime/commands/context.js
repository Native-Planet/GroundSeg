const isMissingRequiredValue = value => value === undefined || value === null

const assertRequiredFields = (commandName, payload, requiredFields = []) => {
  const missingFields = requiredFields.filter(field => isMissingRequiredValue(payload[field]))
  if (missingFields.length > 0) {
    throw new Error(`${commandName} is missing required field(s): ${missingFields.join(', ')}`)
  }
}

const normalizePayload = payload => {
  if (!payload || typeof payload !== 'object') {
    return {}
  }
  return payload
}

export const createCommandContext = sendCommand => {
  const sendTypeCommand = (type, payload = {}) => {
    return sendCommand({ type, ...normalizePayload(payload) })
  }

  const sendActionCommand = (type, action, payload = {}) => {
    return sendCommand({ type, action, ...normalizePayload(payload) })
  }

  const sendSystemCommand = (action, payload = {}) => {
    return sendActionCommand('system', action, payload)
  }

  const sendSetupCommand = (action, payload = {}) => {
    return sendActionCommand('setup', action, payload)
  }

  const sendStartramCommand = (action, payload = {}) => {
    return sendActionCommand('startram', action, payload)
  }

  const sendUrbitCommand = (action, payload = {}) => {
    return sendActionCommand('urbit', action, payload)
  }

  const sendDevCommand = (action, payload = {}) => {
    return sendActionCommand('dev', action, payload)
  }

  const sendValidatedCommand = ({ commandName, type, action, payload = {}, requiredFields = [] }) => {
    const normalizedPayload = normalizePayload(payload)
    assertRequiredFields(commandName, normalizedPayload, requiredFields)

    if (action) {
      return sendActionCommand(type, action, normalizedPayload)
    }

    return sendTypeCommand(type, normalizedPayload)
  }

  return {
    sendTypeCommand,
    sendActionCommand,
    sendSystemCommand,
    sendSetupCommand,
    sendStartramCommand,
    sendUrbitCommand,
    sendDevCommand,
    sendValidatedCommand
  }
}
