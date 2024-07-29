import { loadSession } from './gs-crypto'
let SESSION;

// Initialize connection
export const connect = async (url, logType) => {
  SESSION = new WebSocket(url);
  SESSION.onopen = () => handleOpen(logType);
  SESSION.onmessage = (message) => handleMessage(JSON.parse(message.data));
  SESSION.onerror = (error) => console.log(error);
  SESSION.onclose = () => reconnect(url);
}

export const disconnect = () => {
  SESSION.close(1000)
  SESSION = undefined
  console.log("disconnected")
}

const handleOpen = logType => {
  send(logType)
}

const handleMessage = msg => {
  console.log("msg: " + JSON.stringify(msg))
}

const reconnect = url => {
  console.log("disconnected")
  //connect(url)
}

export const send = async logType => {
  // Grab token if exists
  let token = await loadSession()
  // Create the request if token is available
  if (token) {
    let data = {
      "type": logType,
      "token": token
    }
    // Send the request
    console.log("requesting " + logType + " logs")
    SESSION.send(JSON.stringify(data));
  } else {
    console.log("invalid log session. Not send request: " + logType)
  }
}

