import { writable } from 'svelte/store'
import GroundSegJS from "../../../../tools/groundseg-js"
import { loadSession, saveSession, generateRandom } from './gs-crypto'

// The websocket connection
export let SESSION;
export let PENDING = new Set();
export const structure = writable({});
export const connected = writable(undefined)
export const loginStatus = writable(null)

// Handle messages from API
let count = 0;
const listen = async () => {
  // Make sure session is connected
  if (!SESSION.connected) { 
    if (count % 10 == 0) {
      connect(SESSION.url)
      count = 0;
    }
  }

  // Update the main structure
  structure.set(SESSION.structure)
  connected.set(SESSION.connected)

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
  let newToken = (message === "NEW_TOKEN")
  let authorized = (message === "AUTHORIZED")
  if (newToken || authorized) {
    loginStatus.set("success")
    saveSession(act.token.token)
    verify()
  }

  let orchNotReady = (message === "ORCHESTRATOR_NOT_READY")
  let cfgNotReady = (message === "CONFIG_NOT_READY")
  if (orchNotReady || cfgNotReady) {
    console.log(cid, message)
    verify()
  }

  let authFailed = (message === "AUTH_FAILED")
  if (authFailed) {
    loginStatus.set(message)
  }
  if (cid) {
    console.log(cid, message)
    SESSION.deleteActivity(cid)
    PENDING.delete(cid)
  }
  count += 1
  setTimeout(()=>loginStatus.set(null),3000)
  setTimeout(listen, 500)
}

// Connect to API
export const connect = async url => {
  SESSION = new GroundSegJS(url)
  const connected = await SESSION.connect()
  if (connected) {
    verify()
    listen()
  } else {
    connect(url)
  }
}

//
//  Auth
//

// Verify (token category)
export const verify = async () => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.verify(id,token)
}

// Send Login
export const login = async password => {
  loginStatus.set("loading")
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.login(id,password,token)
}

//
//  Setup
//
export const beginSetup = async () => {
  let id = await generateRandom(16)
  let token = await loadSession()
  PENDING.add(id)
  SESSION.beginSetup(id,token)
}

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
