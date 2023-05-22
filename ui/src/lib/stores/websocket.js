//
// Store and API for Websocket payload
//
import { get, writable } from 'svelte/store'
import { generateRandom, saveSession, loadSession } from './gs-crypto'

// Main Structure
export const socketInfo = writable({
  "activity": {},
  "metadata": {
    "address": "",
    "connected": false,
  }
})

// The websocket connection
export const socket = writable()

// Disconnect from websocket connection
export const disconnect = ws => {
  if (ws) { ws.close() }
}

// Connect to websocket
export const connect = async (addr,info) => {
  // New Websocket
  let ws = new WebSocket(addr)

  // Handle open
  ws.addEventListener('open', e => {
    updateMetadata("connected", e.returnValue)
    let payload = {
      "payload":{
        "category":"token",
        "module":null,
        "action":null
      }
    }
    send(ws, info, payload) 
  })

  // Handle message
  ws.addEventListener('message', e => updateData(e.data))

  // Handle error
  ws.addEventListener('error', e => console.log('error:', e))

  // Handle connection close
  ws.addEventListener('close', e =>setTimeout(()=>{
    console.log("Websocket closed")
    updateMetadata("connected", false)
    console.log("Attempting to reconnect")
    connect(addr, info)
  }, 10000))

  // Update stores
  socket.set(ws)
  updateMetadata("address", addr)
}

// Send message to websocket
export const send = async (ws, info, msg={}) => {
  // Make sure websocket connection is open
  if (info.metadata.connected) {

    // Activity ID
    let id = await generateRandom(16)
    msg['id'] = id
    console.log(id + " attempting to send message.." )

    // Get current token
    let token = await loadSession()
    if (token !== null) {
      msg['token'] = token
    }

    // Send
    ws.send(JSON.stringify(msg))
    return handleActivity(id, msg, info)
  } else {
    console.error("Not connected to websocket")
    return false
  }
}

// Prefix
const actionPrefix = (id, msg) => {
  let payload = msg['payload']
  let cat = payload['category']
  let prefix = id + ":" + cat
  if (cat == "forms") {
    return prefix + ":" + payload.template + ":" + payload.item
  } else if (cat != "token"){
    return prefix + ":" + payload.module + ":" + payload.action
  }
  return prefix
}

// Handle Activity
const handleActivity = async (id, msg, info) => {
  // Prefix
  let prefix = await actionPrefix(id, msg)
  // Handle
  if (!info.activity.hasOwnProperty(id)) {
    console.log(prefix + " checking broadcast..")
    setTimeout(()=>handleActivity(id, msg, info), 500)
  } else {
    return await processActivity(prefix, id, info, msg['payload'])
  }
}

const processActivity = (prefix, id, info, payload) => {
  const act = info.activity[id]
  if (payload['category'] == "token") {
    if (act.status_code == 0) {
      console.log(prefix + " Token verified!")
    } else if ((act.status_code == 2) && (act.message == "NEW_TOKEN")) {
      console.log(prefix + " New Token recieved!")
      saveSession(act.token.token)
      console.log(prefix + " Login Required")
    }
  } else {
    removeActivity(prefix, id)
  }
  
}

// Remove Activity
const removeActivity = async (prefix, id) => {
  const info = get(socketInfo)
  const message = (info?.activity?.[id]?.message) || null
  await socketInfo.update(i => {
    let act = i.activity[id]
    if (act.error == 0) {
      if (act.message.includes("SETUP")) {
        i.metadata['setup'] = true
      }
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
  return await message
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
