export const createRateLimitedJsonParser = ({
  label,
  onError,
  logIntervalMs = 5000,
  previewLength = 220
}) => {
  let lastLoggedAt = 0

  return raw => {
    try {
      return JSON.parse(raw)
    } catch (error) {
      const now = Date.now()
      if (now - lastLoggedAt > logIntervalMs) {
        console.error(`Failed to parse ${label} message`, error)
        lastLoggedAt = now
      }

      if (onError) {
        onError(error, {
          rawPreview: String(raw).slice(0, previewLength)
        })
      }

      return null
    }
  }
}
