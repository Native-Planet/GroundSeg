const normalizePath = path => {
  if (!path) {
    return ''
  }

  return path.startsWith('/') ? path : `/${path}`
}

export const resolveWebsocketUrl = ({ hostname, port, path, customHostname }) => {
  const finalHostname = customHostname || hostname
  return `ws://${finalHostname}:${port}${normalizePath(path)}`
}
