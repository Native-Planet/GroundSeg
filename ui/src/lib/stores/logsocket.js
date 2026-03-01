import { loadSession } from './gs-crypto'
import { writable } from 'svelte/store'
import { createRateLimitedJsonParser } from '../runtime/transport/json.js'
import { createWebSocketClient } from '../runtime/transport/websocket-client.js'

export const logs = writable({})
const sessionsByType = new Map()

export const safeParseLogSocketMessage = createRateLimitedJsonParser({
  label: 'log websocket'
})

export const replaceLogHistory = (current, type, history) => {
  return {
    ...current,
    [type]: history
  }
}

export const appendLogMessage = (current, type, line) => {
  const existing = Array.isArray(current[type]) ? current[type] : []
  return {
    ...current,
    [type]: [...existing, line]
  }
}

export const removeLogType = (current, logType) => {
  const { [logType]: _, ...rest } = current
  return rest
}

const handleMessage = msg => {
  if (msg.history) {
    logs.update(current => replaceLogHistory(current, msg.type, msg.log))
  } else {
    logs.update(current => appendLogMessage(current, msg.type, msg.log))
  }
}

// Initialize connection
export const connect = async (url, logType) => {
  const existingSession = sessionsByType.get(logType)
  if (existingSession) {
    existingSession.disconnect(1000)
  }

  const socketClient = createWebSocketClient({
    url,
    onOpen: () => {
      requestLogs(logType)
    },
    onMessage: message => {
      const parsed = safeParseLogSocketMessage(message.data)
      if (parsed) {
        handleMessage(parsed)
      }
    },
    onError: error => {
      console.error(error)
    },
    onClose: () => {
      sessionsByType.delete(logType)
    }
  })

  sessionsByType.set(logType, socketClient)
  socketClient.connect()
  return { ok: true }
}

export const disconnect = logType => {
  const session = sessionsByType.get(logType)
  if (!session) {
    return { ok: false, error: 'missing_session' }
  }

  session.disconnect(1000)
  sessionsByType.delete(logType)
  logs.update(current => removeLogType(current, logType))
  console.log(`${logType} logs disconnected`)
  return { ok: true }
}

export const requestLogs = async logType => {
  const session = sessionsByType.get(logType)
  if (!session) {
    return { ok: false, error: 'missing_session' }
  }

  if (!session.isOpen()) {
    return {
      ok: false,
      error: 'socket_not_open',
      readyState: session.getReadyState()
    }
  }

  const token = await loadSession()

  if (!token) {
    console.log(`invalid log session. Not send request: ${logType}`)
    return { ok: false, error: 'missing_token' }
  }

  const payload = {
    type: logType,
    token
  }

  console.log(`requesting ${logType} logs`)
  const sendResult = session.send(JSON.stringify(payload))

  if (!sendResult.ok) {
    return {
      ok: false,
      error: sendResult.reason,
      readyState: sendResult.readyState
    }
  }

  return { ok: true }
}
