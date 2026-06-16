import { loadSession } from './gs-crypto'
import { writable } from 'svelte/store'

export let logs = writable({})

let SESSIONS = {}

const logsUrlFromSocket = url => {
  const target = new URL(url)
  target.protocol = target.protocol === 'wss:' ? 'https:' : 'http:'
  return target.toString()
}

// Initialize connection
export const connect = async (url, logType) => {
  disconnect(logType)
  const controller = new AbortController()
  SESSIONS[logType] = {
    controller,
    url: logsUrlFromSocket(url),
    retry: 1000,
    reconnectTimer: undefined
  }
  stream(logType)
}

export const disconnect = logType => {
  const session = SESSIONS[logType]
  if (session?.reconnectTimer) {
    clearTimeout(session.reconnectTimer)
  }
  session?.controller?.abort()
  delete SESSIONS[logType]
  logs.update(current => {
    delete current[logType]
    return current
  })
  console.log(logType + " logs disconnected")
}

const stream = async logType => {
  const session = SESSIONS[logType]
  if (!session) return

  const token = await loadSession()
  if (!token) {
    console.log("invalid log session. Not sending request: " + logType)
    scheduleReconnect(logType)
    return
  }

  try {
    console.log("requesting " + logType + " logs")
    const response = await fetch(session.url, {
      method: 'POST',
      headers: {
        'Accept': 'text/event-stream',
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        type: logType,
        token
      }),
      cache: 'no-store',
      signal: session.controller.signal
    })
    if (!response.ok || !response.body) {
      throw new Error(`log stream failed: ${response.status}`)
    }
    session.retry = 1000
    await readEventStream(response.body)
  } catch (error) {
    if (!session.controller.signal.aborted) {
      console.log(error)
    }
  } finally {
    if (SESSIONS[logType] === session && !session.controller.signal.aborted) {
      scheduleReconnect(logType)
    }
  }
}

const scheduleReconnect = logType => {
  const session = SESSIONS[logType]
  if (!session || session.reconnectTimer) return
  const retry = session.retry
  session.retry = Math.min(session.retry * 2, 10000)
  session.reconnectTimer = setTimeout(() => {
    session.reconnectTimer = undefined
    stream(logType)
  }, retry)
}

const readEventStream = async body => {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  for (;;) {
    const { value, done } = await reader.read()
    if (done) return
    buffer += decoder.decode(value, { stream: true })
    let boundary = buffer.indexOf('\n\n')
    while (boundary >= 0) {
      const raw = buffer.slice(0, boundary)
      buffer = buffer.slice(boundary + 2)
      dispatchEventFrame(raw)
      boundary = buffer.indexOf('\n\n')
    }
  }
}

const dispatchEventFrame = raw => {
  if (!raw.trim() || raw.trim().startsWith(':')) return
  const data = []
  for (const line of raw.split(/\r?\n/)) {
    if (!line || line.startsWith(':')) continue
    const separator = line.indexOf(':')
    const field = separator >= 0 ? line.slice(0, separator) : line
    let value = separator >= 0 ? line.slice(separator + 1) : ''
    if (value.startsWith(' ')) value = value.slice(1)
    if (field === 'data') data.push(value)
  }
  if (data.length < 1) return
  try {
    handleMessage(JSON.parse(data.join('\n')))
  } catch (error) {
    console.log(error)
  }
}

const handleMessage = msg => {
  if (msg.history) {
    logs.update(current => {
      current[msg.type] = msg.log
      return current
    })
  } else {
    logs.update(current => {
      if (!current[msg.type]) {
        current[msg.type] = []
      }
      current[msg.type].push(msg.log)
      return current
    })
  }
}
