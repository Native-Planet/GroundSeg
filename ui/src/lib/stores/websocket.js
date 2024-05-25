import { get, writable } from 'svelte/store'
import { loadSession, saveSession, generateRandom } from './gs-crypto'
import { URBIT_MODE, connected, structure, firstLoad } from './data.js'
import { sendPoke } from './urbit.js'

export const ready = writable(false)
export const logs = writable({})
export const wsPort = writable("3000")
export const isC2CMode = writable(false)
export const ssids = writable([])
export const loginError = writable('')

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
  if (get(URBIT_MODE)) {
    sendPoke(payload)
  } else {
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
  console.log("alive")
  //console.log(data)
  if (data.type === "c2c") {
    if (Array.isArray(data.ssids)) {
      ssids.set(data.ssids)
      isC2CMode.set(true)
    }
  } else if (data.type === "activity") {
    handleActivity(data)
    ssids.set([])
    isC2CMode.set(false)
  } else if (data.type == "structure") {
    structure.set(data)
    ssids.set([])
    isC2CMode.set(false)
  } else if (data.type == 'login-failed') {
    loginError.set(data.message);
    setTimeout(() => {
      loginError.set('');
    }, 2000);
  } else if (data.hasOwnProperty('log')) {
    logs.update(l=>{
      let containerID = data.log.container_id
      let containerLine = data.log.line
      if (l.hasOwnProperty(containerID)) {
        l[containerID] = l[containerID] + "\n" + containerLine
      } else {
        l[containerID] = containerLine
      }
      return l
    })
  } else {
    console.log("server alive")
  }
  firstLoad.set(false)
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
  loginError.set('');
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

export const startramGetRegions = () => {
  let payload = {
    "type":"startram",
    "action":"regions"
  }
  send(payload)
}

export const startramGetServices = () => {
  let payload = {
    "type":"startram",
    "action":"services"
  }
  send(payload)
}

export const startramRegister = (key,region) => {
  let payload = {
    "type":"startram",
    "action":"register",
    "key":key,
    "region":region
  }
  send(payload)
}

export const startramToggle = () => {
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

export const setAllStartramReminder = remind => {
  let payload = {
    "type":"startram",
    "action":"reminder",
    "remind": remind
  }
  send(payload)
}

//
//  Upload Pier
//

export const openUploadEndpoint = (endpoint,remote,fix,selectedDrive) => {
  let payload = {
    "type":"pier_upload",
    "action":"open-endpoint",
    "endpoint": endpoint,
    "remote": remote,
    "fix": fix,
    "selectedDrive":selectedDrive,
  }
  send(payload)
}

export const modifyUploadEndpoint = (endpoint,remote,fix) => {
  let payload = {
    "type":"pier_upload",
    "action":"modify-endpoint",
    "endpoint": endpoint,
    "remote": remote,
    "fix": fix
  }
  send(payload)
}

export const closeUploadEndpoint = endpoint => {
  let payload = {
    "type":"pier_upload",
    "action":"close-endpoint",
    "endpoint": endpoint
  }
  send(payload)
}

export const resetImportShip = () => {
  let payload = {
    "type":"pier_upload",
    "action":"reset",
  }
  send(payload)
}

//
//  Boot New Ship
//

export const cancelNewShip = patp => {
  let payload = {
    "type":"new_ship",
    "action":"cancel",
    "patp":patp,
  }
  send(payload)
}

export const bootShip = (patp,key,remote,selectedDrive) => {
  let payload = {
    "type":"new_ship",
    "action":"boot",
    "patp":patp,
    "key":key,
    "remote":remote,
    "selectedDrive":selectedDrive
  }
  send(payload)
}

export const resetNewShip = () => {
  let payload = {
    "type":"new_ship",
    "action":"reset",
  }
  send(payload)
}

//
//  Urbits
//

export const registerServiceAgain = patp => {
  let payload = {
    "type":"urbit",
    "action":"register-service-again",
    "patp":patp,
  }
  send(payload)
}

export const toggleBootStatus = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-boot-status",
    "patp":patp
  }
  send(payload)
}

export const toggleDevMode = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-devmode",
    "patp":patp
  }
  send(payload)
}

export const setMinIODomain = (patp, domain) => {
  let payload = {
    "type":"urbit",
    "action":"set-minio-domain",
    "patp":patp,
    "domain": domain
  }
  send(payload)
}

export const setUrbitDomain = (patp, domain) => {
  let payload = {
    "type":"urbit",
    "action":"set-urbit-domain",
    "patp":patp,
    "domain": domain
  }
  send(payload)
}


export const setNewMaxPierSize = (patp, size) => {
  let payload = {
    "type":"urbit",
    "action":"new-max-pier-size",
    "patp":patp,
    "value": size
  }
  send(payload)
}

export const toggleAutoBoot = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-autoboot",
    "patp":patp
  }
  send(payload)
}

export const toggleNetwork = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-network",
    "patp":patp
  }
  send(payload)
}

export const toggleMinIOLink = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-minio-link",
    "patp":patp
  }
  send(payload)
}

export const toggleUrbitPower = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-power",
    "patp":patp
  }
  send(payload)
}

export const deleteUrbitShip = patp => {
  let payload = {
    "type":"urbit",
    "action":"delete-ship",
    "patp":patp
  }
  send(payload)
}

export const exportUrbitShip = patp => {
  let payload = {
    "type":"urbit",
    "action":"export-ship",
    "patp":patp
  }
  send(payload)
}

export const exportUrbitBucket = patp => {
  let payload = {
    "type":"urbit",
    "action":"export-bucket",
    "patp":patp
  }
  send(payload)
}

export const rebuildContainer = patp => {
  let payload = {
    "type":"urbit",
    "action":"rebuild-container",
    "patp": patp
  }
  send(payload)
}

export const setUrbitLoom = (patp, value) => {
  let payload = {
    "type":"urbit",
    "action":"loom",
    "patp":patp,
    "value": value
  }
  send(payload)
}

export const setPackSchedule = (patp, frequency, intervalType, time, day, date) => {
  let payload = {
    "type":"urbit",
    "action":"schedule-pack",
    "patp":patp,
    "frequency": frequency,
    "intervalType": intervalType,
    "time": time,
    "day": day,
    "date": date
  }
  send(payload)
}

export const pausePackSchedule = patp => {
  let payload = {
    "type":"urbit",
    "action":"pause-pack-schedule",
    "patp":patp,
  }
  send(payload)
}

export const toggleUrbitAlias = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-alias",
    "patp":patp,
  }
  send(payload)
}

export const marsPack = patp => {
  let payload = {
    "type":"urbit",
    "action":"pack",
    "patp":patp,
  }
  send(payload)
}

export const urthPackMeld = patp => {
  let payload = {
    "type":"urbit",
    "action":"pack-meld",
    "patp":patp,
  }
  send(payload)
}

export const urbitChop = patp => {
  let payload = {
    "type":"urbit",
    "action":"chop",
    "patp":patp,
  }
  send(payload)
}

export const urbitRollChop = patp => {
  let payload = {
    "type":"urbit",
    "action":"roll-chop",
    "patp":patp
  }
  send(payload)
}

export const toggleChopAfterVereUpdate = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-chop-on-vere-update",
    "patp":patp,
  }
  send(payload)
}

export const setStartramReminder = (patp, remind) => {
  let payload = {
    "type":"urbit",
    "action":"startram-reminder",
    "patp":patp,
    "remind": remind
  }
  send(payload)
}

//
//  Support
//

export const submitReport = (contact,description,ships,cpuProfile,penpai) => {
  let payload = {
    "type":"support",
    "action":"bug-report",
    "contact":contact,
    "description":description,
    "ships":ships,
    "cpu_profile":cpuProfile,
    "penpai":penpai
  }
  send(payload)
}

//
//  Logs
//

export const toggleLog = (name,action) => {
  let payload = {
    "type":"logs",
    "action":action,
    "container_id": name,
  }
  send(payload)
}

//
//  C2C
//

export const submitNetwork = (ssid,password) => {
  let payload = {
    "type":"c2c",
    "ssid":ssid,
    "password": password
  }
  send(payload)
}

//
// Penpai
//

export const toggleExperimentalPenpai = () => {
  let payload = {
    "type":"system",
    "action": "toggle-penpai-feature",
  }
  send(payload)
}

export const togglePenpai = () => {
  let payload = {
    "type":"penpai",
    "action": "toggle",
  }
  send(payload)
}

export const setPenpaiModel = model => {
  let payload = {
    "type":"penpai",
    "action": "set-model",
    "model": model
  }
  send(payload)
}

export const setPenpaiCores = cores => {
  let payload = {
    "type":"penpai",
    "action": "set-cores",
    "cores": cores
  }
  send(payload)
}

export const removePenpai = () => {
  let payload = {
    "type":"penpai",
    "action": "remove"
  }
  send(payload)
}

export const installPenpaiCompanion = patp => {
  let payload = {
    "type":"urbit",
    "action":"install-penpai-companion",
    "patp":patp,
  }
  send(payload)
}

export const uninstallPenpaiCompanion = patp => {
  let payload = {
    "type":"urbit",
    "action":"uninstall-penpai-companion",
    "patp":patp,
  }
  send(payload)
}

export const installGallseg = patp => {
  let payload = {
    "type":"urbit",
    "action":"install-gallseg",
    "patp":patp,
  }
  send(payload)
}

export const deleteStartramService = (patp, service) => {
  let payload = {
    "type":"urbit",
    "action":"delete-service",
    "patp": patp,
    "service":service
  }
  send(payload)
}

export const uninstallGallseg = patp => {
  let payload = {
    "type":"urbit",
    "action":"uninstall-gallseg",
    "patp":patp,
  }
  send(payload)
}

//
//  Dev
//   

export const resetSetup = () => {
  let payload = {
    "type":"dev",
    "action":"reset-setup",
  }
  send(payload)
}

export const printMounts = () => {
  let payload = {
    "type":"dev",
    "action":"print-mounts",
  }
  send(payload)
}

export const devStartramReminder = () => {
  let payload = {
    "type":"dev",
    "action":"startram-reminder",
  }
  send(payload)
}

export const devStartramReminderToggle = b => {
  let payload = {
    "type":"dev",
    "action":"startram-reminder-toggle",
    "reminded": b
  }
  send(payload)
}

export const transloadPier = (patp, path, remote, fix, selectedDrive) => {
  let payload = {
    "type":"pier_transload",
    "action":"transload",
    "patp": patp,
    "path": path,
    "remote": remote,
    "fix": fix,
    "selectedDrive":selectedDrive,
  }
  send(payload)
}