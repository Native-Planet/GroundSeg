import { writable } from 'svelte/store'
import GroundSegJS from "../../../../tools/groundseg-js"
import { loadSession, saveSession, generateRandom } from './gs-crypto'

// The websocket connection
export let SESSION;
export let PENDING = new Set();
export const structure = writable({});

// Handle messages from API
const listen = async () => {
  // Update the main structure
  structure.set(SESSION.structure)
  // Activity Checker
  let act,cid;
  for (let id of PENDING) {
    if (SESSION.activity.activity.hasOwnProperty(id)) {
      act = await SESSION.activity.activity[id]
      cid = await id
      break
    }
  }
  let message = (act?.message) || null
  if ((message === "NEW_TOKEN") || (message === "AUTHORIZED")) {
    saveSession(act.token.token)
    verify()
  }
  if (cid) {
    SESSION.deleteActivity(cid)
    PENDING.delete(cid)
  }
  setTimeout(listen, 500)
}

// Connect to API
export const connect = async url => {
  SESSION = new GroundSegJS(url, structure)
  const connected = await SESSION.connect()
  if (connected) {
    verify()
    listen()
  }
}

// Verify (token category)
export const verify = async () => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.verify(id,token)
}

// Send Login
export const login = async password => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.login(id,password,token)
}

export const send = async msg => {
  // Activity ID
  let id = await generateRandom(16)
  msg['id'] = id

  // Get current token
  let token = await loadSession()
  if (token !== null) {
    msg['token'] = token
  }

  // Send
  SESSION.send(msg)
  PENDING.add(id)
}

//
// Store and API for Websocket payload
/*
import { get, writable } from 'svelte/store'

// Main Structure
export const socketInfo = writable({
  "activity": {},
  "metadata": {
    "address": "",
    "connected": false,
  }
})


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
    return handleActivity(ws, id, msg, info)
  } else {
    console.error("Not connected to websocket")
    return false
  }
}

// Handle Activity
const handleActivity = async (ws, id, msg, info) => {
  // Prefix
  let prefix = await actionPrefix(id, msg)
  // Handle
  if (!info.activity.hasOwnProperty(id)) {
    console.log(prefix + " checking broadcast..")
    setTimeout(()=>handleActivity(id, msg, info), 500)
  } else {
    return await processActivity(ws, prefix, id, info, msg['payload'])
  }
}

const processActivity = (ws, prefix, id, info, payload) => {

  const handleToken = act => {
    if (act.status_code == 0) {
      console.log(prefix + " Token verified!")
    } else if ((act.status_code == 2) && (act.message == "NEW_TOKEN")) {
      console.log(prefix + " New Token recieved!")
      saveSession(act.token.token)
      let payload = {
        "payload":{
          "category":"token",
          "module":null,
          "action":null
        }
      }
      send(ws, info, payload) 
    }
  }

  const act = info.activity[id]

  if (payload['category'] == "token") {
    console.log(act)
    handleToken(act)
  } else {
    removeActivity(prefix, id)
  }
  
}
*/
