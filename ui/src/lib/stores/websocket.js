import { writable } from 'svelte/store'

export const socketInfo = writable({
  "metadata": {
    "address": "",
    "connected": false,
  },
  "urbits": {},
  "system": {}
})
export const socket = writable()

export const disconnect = ws => {
  if (ws) { ws.close() }
}

export const send = (ws, msg) => {
  console.log("Request sent: " + JSON.stringify(msg))
  ws.send(JSON.stringify(msg))
}

export const connect = async addr => {
  let ws = new WebSocket(addr)
  let connected = false
  ws.addEventListener('open', e => updateMetadata("connected", e.returnValue))
  ws.addEventListener('message', e => updateData(e))
  ws.addEventListener('error', e => console.log('error:', e))
  ws.addEventListener('close', e => console.log('closed:', e))
  socket.set(ws)
  updateMetadata("address", addr)
}

const updateMetadata = (item, val) => {
  socketInfo.update(i => {
    if (item == "address") {
      i.metadata.address = val
    }
    if (item == "connected") {
      i.metadata.connected = val
      console.log("Connected: " + val)
    }
    return i
  })
}

const updateData = data => {
  console.log(data)
}
