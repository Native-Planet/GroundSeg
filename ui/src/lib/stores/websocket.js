import { writable } from 'svelte/store'
//import GroundSegJS from "../../../../tools/groundseg-js"
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

// Verify session
export const login = async password => {
  let payload = {"type":"login"}
  send(payload)
}


//
//  Setup
//
export const beginSetup = async () => {
  let payload = {"type":"setup","action":"begin"}
  send(payload)
}

/*
//
//  Urbits
//

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

// 
// Update Form
//

export const updateForm = async (template,item,value) => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.updateForm(id,template,item,value,token)
}

//
//  StarTram
//

// Register StarTram
export const starTramRegister = async () => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.starTramRegister(id,token)
}

// Toggle StarTram
export const starTramToggle = async c => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  c == "running"
    ? SESSION.starTramStop(id,token)
    : SESSION.starTramStart(id,token)
}

// Restart StarTram
export const starTramRestart = async () => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.starTramRestart(id,token)
}

// Modify endpoint
export const starTramEndpoint = async () => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.starTramEndpoint(id,token)
}

// Cancel StarTram Subscription
export const starTramCancel = async () => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.starTramCancel(id,token)
}
*/
