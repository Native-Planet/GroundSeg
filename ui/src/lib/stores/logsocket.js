import { loadSession } from './gs-crypto'
import { get, writable } from 'svelte/store'

export let logs = writable({})
let SESSION = {};

// Initialize connection
export const connect = async (url, logType) => {
  SESSION[logType] = new WebSocket(url);
  SESSION[logType].onopen = () => handleOpen(logType);
  SESSION[logType].onmessage = (message) => handleMessage(JSON.parse(message.data));
  SESSION[logType].onerror = (error) => console.log(error);
  SESSION[logType].onclose = () => {};
}

export const disconnect = logType => {
  SESSION[logType].close(1000)
  SESSION[logType] = undefined
  logs.update(current => {
    delete current[logType]
    return current
  })
  console.log(logType + " logs disconnected")
}

const handleOpen = logType => {
  send(logType)
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

export const send = async logType => {
  // Grab token if exists
  let token = await loadSession()
  // Create the request if token is available
  if (token) {
    let data = {
      "type": logType,
      "token": token,
    }
    // Send the request
    console.log("requesting " + logType + " logs")
    SESSION[logType].send(JSON.stringify(data));
  } else {
    console.log("invalid log session. Not send request: " + logType)
  }
}

