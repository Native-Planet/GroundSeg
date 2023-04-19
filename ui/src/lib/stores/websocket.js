//
// Store and API for Websocket payload
//

import { writable } from 'svelte/store'
import { genRequestId, getCookie } from '/src/lib/scripts/session.js'

export const socketInfo = writable({
  "activity": {},
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

export const removeActivity = id => {
  socketInfo.update(i => {
    delete i.activity[id]
    return i
  })
}


export const send = (ws, cookie, msg) => {
  msg = msg || {}
  let id = genRequestId(16)
  let sid = getCookie(cookie, 'sessionid')
  msg['id'] = id
  msg['sessionid'] = sid
  ws.send(JSON.stringify(msg))
  return id
}

export const connect = async (addr, cookie) => {
  let ws = new WebSocket(addr)
  let connected = false
  ws.addEventListener('open', e => {
    updateMetadata("connected", e.returnValue)
    send(ws,cookie,{"category":"ping"}) 
  })
  ws.addEventListener('message', e => updateData(e.data))
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
  data = JSON.parse(data)
  socketInfo.update(i => {
    let obj = deepMerge(i, data)
    return obj
  })
}

const deepMerge = (target, source) => {
  for (const key in source) {
    if (typeof source[key] === 'object' && !Array.isArray(source[key]) && source[key] !== null) {
      if (!target.hasOwnProperty(key)) {
        target[key] = {};
      }
      deepMerge(target[key], source[key])
    } else {
      target[key] = source[key]
    }
  }
  return target
}
