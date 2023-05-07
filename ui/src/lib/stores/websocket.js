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
  }
})

export const socket = writable()

export const disconnect = ws => {
  if (ws) { ws.close() }
}

export const connect = async (addr, cookie, info) => {
  let ws = new WebSocket(addr)
  ws.addEventListener('open', e => {
    updateMetadata("connected", e.returnValue)
    send(ws, info, cookie, {"category":"ping"}) 
  })
  ws.addEventListener('message', e => updateData(e.data))
  ws.addEventListener('error', e => console.log('error:', e))
  ws.addEventListener('close', e =>setTimeout(()=>{
    console.log("Websocket closed")
    updateMetadata("connected", false)
    console.log("Attempting to reconnect")
    connect(addr, cookie, info)
  }, 1000))
  socket.set(ws)
  updateMetadata("address", addr)
}

export const send = (ws, info, cookie, msg) => {
  if (info.metadata.connected) {
    msg = msg || {}
    let id = genRequestId(16)
    console.log(id + " attempting to send message.." )
    let sid = getCookie(cookie, 'sessionid')
    msg['id'] = id
    msg['sessionid'] = sid
    ws.send(JSON.stringify(msg))
    let category = msg['category']
    let payload = null
    if (category != 'ping') {
      payload = msg['payload']
    }
    return handleActivity(id, category, payload, info)
  } else {
    console.error("Not connected to websocket")
    return false
  }
}

const handleActivity = async (id, cat, load, info) => {
  let prefix = id + ":" + cat
  if (cat == "forms") {
    prefix = prefix + ":" + load.template + ":" + load.item
  } else if (cat != "ping") {
    prefix = prefix + ":" + load.module + ":" + load.action
  }

  if (!info.activity.hasOwnProperty(id)) {
    console.log(prefix + " checking broadcast..")
    setTimeout(()=>handleActivity(id, cat, load, info), 500)
  } else {
    return await removeActivity(prefix, id)
    return true
  }
}

const removeActivity = async (prefix, id) => {
  await socketInfo.update(i => {
    let act = i.activity[id]
    if (act.error == 0) {
      console.log(prefix + " send confirmed")
    } else {
      if (act.message.includes("auth-fail")) {
        console.log("jump to login")
        
      }
      console.warn(prefix + " sent but error: " + act.message + ", error code: " + act.error)
    }
    delete i.activity[id]
    return i
  })
  return true
}

const updateData = data => {
  data = JSON.parse(data)
  socketInfo.update(i => {
    let obj = deepMerge(i, data)
    return obj
  })
}

const updateMetadata = (item, val) => {
  socketInfo.update(i => {
    if (item == "address") {
      i.metadata.address = val
    }
    if (item == "connected") {
      i.metadata.connected = val
      if (val) {
        console.log("Websocket Successfully Connected")
      } else {
        console.error("Websocket Failed to connect")
      }
    }
    return i
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
