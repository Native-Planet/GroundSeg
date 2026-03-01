export const WEBSOCKET_OPEN_STATE = 1

export const createWebSocketClient = ({
  url,
  onOpen,
  onMessage,
  onError,
  onClose
}) => {
  let session = null

  const connect = () => {
    session = new WebSocket(url)

    session.onopen = () => {
      if (onOpen) {
        onOpen()
      }
    }

    session.onmessage = message => {
      if (onMessage) {
        onMessage(message)
      }
    }

    session.onerror = error => {
      if (onError) {
        onError(error)
      }
    }

    session.onclose = event => {
      if (onClose) {
        onClose(event)
      }
    }

    return session
  }

  const isOpen = () => {
    return Boolean(session) && session.readyState === WEBSOCKET_OPEN_STATE
  }

  const send = data => {
    if (!session) {
      return { ok: false, reason: 'missing', readyState: 'missing' }
    }

    if (!isOpen()) {
      return { ok: false, reason: 'not_open', readyState: session.readyState }
    }

    try {
      session.send(data)
      return { ok: true }
    } catch (error) {
      return { ok: false, reason: 'send_failed', error, readyState: session.readyState }
    }
  }

  const disconnect = (code = 1000) => {
    if (session) {
      session.close(code)
    }
  }

  const getReadyState = () => {
    if (!session) {
      return 'missing'
    }

    return session.readyState
  }

  return {
    connect,
    disconnect,
    send,
    isOpen,
    getReadyState,
    getSession: () => session
  }
}
