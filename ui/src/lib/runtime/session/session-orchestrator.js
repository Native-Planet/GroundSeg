import { resolveWebsocketUrl } from '../transport/url.js'

export const startRuntimeSession = ({
  pageUrl,
  wsPort,
  urbitModeEnabled,
  customHostname,
  connectWebsocket,
  subscribeUrbit,
  ship
}) => {
  if (urbitModeEnabled) {
    subscribeUrbit(ship)
    return {
      transport: 'urbit',
      url: null
    }
  }

  const url = resolveWebsocketUrl({
    hostname: pageUrl.hostname,
    port: wsPort,
    path: '/ws',
    customHostname
  })

  connectWebsocket(url)

  return {
    transport: 'websocket',
    url
  }
}
