import { writable } from 'svelte/store'
import { loadSession, saveSession, generateRandom } from './gs-crypto'

export const structure = writable({})
export const ready = writable(false)
export const connected = writable(false)

let PENDING = new Set();
let SESSION;

// Initialize connection
export const connect = async url => {
  SESSION = new WebSocket(url);
  SESSION.onopen = () => handleOpen();
  SESSION.onmessage = (message) => handleMessage(JSON.parse(message.data));
  SESSION.onerror = (error) => console.log(error);
  SESSION.onclose = () => reconnect(url);
}

// WebSocket send wrapper
export const send = async payload => {
  // generate an ID
  let id = await generateRandom(16)
  // add the ID to pending
  PENDING.add(id)
  // Grab token if exists
  let token = await loadSession()
  // Create the request
  let data = {"id":id,"payload":payload}
  // Add token to request if available
  if (token) {
    data['token'] = token
  }
  // Send the request
  console.log(id + ":" + payload.type + " sent")
  SESSION.send(JSON.stringify(data));
}

// Reconnection
export const reconnect = url => {
  // Set connected store to false
  connected.set(false)
  console.log("reconnecting to api")
  // Attempt to reconnect
  setTimeout(()=>connect(url),1000)
}

// Handle connection
export const handleOpen = () => {
  // Set connected store to true
  connected.set(true)
  // Verify session
  verify()
}

// Message Handler
export const handleMessage = data => {
  // Log the activity response and remove 
  // it from pending
  if (data.type === "activity") {
    handleActivity(data)
  } else {
    let now = new Date()
    console.log(now)
    console.log(data)
    structure.set(data)
  }
}

// Activity Handler
export const handleActivity = data => {
    // ack
    let res = data.id + " " + data.response
    // nack
    if (data.response === "nack") {
      res = res + ": " + data.error  
    }
    // GroundSeg hasn't fully started
    if (data.error === "NOT_READY") {
      ready.set(false)
    } else {
      ready.set(true)
    }
    // Set token
    if (data.hasOwnProperty('token')) {
      saveSession(data.token)
    }
    // display result
    console.log(res)
    // remove from pending
    PENDING.delete(data.id)
}

//
//  Auth
//

// Verify session
export const verify = async () => {
  let payload = {"type":"verify"}
  send(payload)
}

// Login
export const login = async password => {
  let payload = {
    "type":"login",
    "password":password
  }
  send(payload)
}

// Logout
export const logout = () => {
  let payload = {"type":"logout"}
  send(payload)
}

// Logout everywhere
export const logoutAll = () => {
  let payload = {
    "type":"logout",
    "action":"everywhere"
  }
  send(payload)
}

export const modifyPassword = (old,pwd) => {
  let payload = {
    "type":"password",
    "action":"modify",
    "old":old,
    "password":pwd
  }
  send(payload)
}

//
//  Setup
//

export const beginSetup = async () => {
  let payload = {"type":"setup","action":"begin"}
  send(payload)
}

export const setupPassword = async pwd => {
  let payload = {
    "type":"setup",
    "action":"password",
    "password":pwd
  }
  send(payload)
}

export const setupSkip = async () => {
  let payload = {"type":"setup","action":"skip"}
  send(payload)
}

export const setupStarTram = async (key,region) => {
  let payload = {
    "type":"setup",
    "action":"startram",
    "key":key,
    "region":region
  }
  send(payload)
}

//
//  System
//

export const restartGroundSeg = () => {
  let payload = {
    "type":"system",
    "action":"groundseg",
    "command":"restart"
  }
  send(payload)
}

export const restartDevice = () => {
  let payload = {
    "type":"system",
    "action":"power",
    "command":"restart"
  }
  send(payload)
}

export const shutdownDevice = () => {
  let payload = {
    "type":"system",
    "action":"power",
    "command":"shutdown"
  }
  send(payload)
}

export const updateLinux = () => {
  let payload = {
    "type":"system",
    "action":"update",
    "update":"linux"
  }
  send(payload)
}

export const setSwap = val => {
  let payload = {
    "type":"system",
    "action":"modify-swap",
    "value": val
  }
  send(payload)
}

export const toggleWifi = () => {
  let payload = {
    "type":"system",
    "action":"wifi-toggle"
  }
  send(payload)
}

export const connectWifi = (ssid,pwd) => {
  let payload = {
    "type":"system",
    "action":"wifi-connect",
    "ssid":ssid,
    "password":pwd
  }
  send(payload)
}

//
//  StarTram
//

export const startramGetRegions = async () => {
  let payload = {
    "type":"startram",
    "action":"regions"
  }
  send(payload)
}

export const startramRegister = async (key,region) => {
  let payload = {
    "type":"startram",
    "action":"register",
    "key":key,
    "region":region
  }
  send(payload)
}

export const startramToggle = async () => {
  let payload = {
    "type":"startram",
    "action":"toggle"
  }
  send(payload)
}

export const startramRestart = async () => {
  let payload = {
    "type":"startram",
    "action":"restart"
  }
  send(payload)
}

export const startramEndpoint = async endpoint => {
  let payload = {
    "type":"startram",
    "action":"endpoint",
    "endpoint":endpoint
  }
  send(payload)
}

export const startramCancel = async (key,reset) => {
  let payload = {
    "type":"startram",
    "action":"cancel",
    "key":key,
    "reset":reset
  }
  send(payload)
}
//
//  Upload Pier
//

export const freeUpload = () => {
  let payload = {
    "type":"pier_upload",
    "action":"free"
  }
  send(payload)
}

export const uploadMetadata = (patp,size,secret) => {
  let payload = {
    "type":"pier_upload",
    "action":"metadata",
    "patp":patp,
    "size":size,
    "secret":secret
  }
  send(payload)
}

//
//  Urbits
//

export const bootShip = (patp,key,remote) => {
  let payload = {
    "type":"new_ship",
    "action":"boot",
    "patp":patp,
    "key":key,
    "remote":remote
  }
  send(payload)
}

export const registerServiceAgain = patp => {
  let payload = {
    "type":"urbit",
    "action":"register-service-again",
    "patp":patp,
  }
  send(payload)
}

/*
export const urbitsAccessToggle = async ship => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.urbitsAccessToggle(id,ship,token)
}

export const urbitsMeldUrth = async ship => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.urbitsMeldUrth(id,ship,token)
}
*/

//
//  Support
//

export const submitReport = (contact,description,ships) => {
  let payload = {
    "type":"support",
    "action":"bug-report",
    "contact":contact,
    "description":description,
    "ships":ships
  }
  send(payload)
}

//
// Logs
//

// Stream the result of `wg show`
export const openWireguardLog = () => {
  let payload = {
    "type":"container-logs",
    "action":"open",
    "container":"wireguard"
  }
  send(payload)
}
