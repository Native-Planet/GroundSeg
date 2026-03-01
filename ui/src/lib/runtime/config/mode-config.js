const parseBooleanEnv = value => value === true || value === 'true'

const trimOrNull = value => {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed.length > 0 ? trimmed : null
}

export const runtimeModeConfig = Object.freeze({
  urbitMode: parseBooleanEnv(process.env.GS_URBIT_MODE),
  devPanel: parseBooleanEnv(process.env.GS_DEV_PANEL),
  customHostname: trimOrNull(process.env.GS_CUSTOM_HOSTNAME)
})
