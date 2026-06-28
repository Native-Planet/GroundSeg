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

let PENDING = new Set()
let OUTBOX = []
let SESSION
let EVENTS_URL = ''
let wsConnected = false
let eventConnected = false
let eventConnecting = false
let reconnectTimer
let eventReconnectTimer
let eventController
let eventTokenKey = ''
let eventRetry = 1000

const maxEventRetry = 10000

const updateConnected = () => {
  connected.set(wsConnected && (eventConnected || !eventTokenKey))
}

const eventsUrlFromSocket = url => {
  const target = new URL(url)
  target.protocol = target.protocol === 'wss:' ? 'https:' : 'http:'
  target.pathname = '/events'
  target.search = ''
  target.hash = ''
  return target.toString()
}

// Initialize connection
export const connect = async url => {
  EVENTS_URL = eventsUrlFromSocket(url)
  if (SESSION && [WebSocket.CONNECTING, WebSocket.OPEN].includes(SESSION.readyState)) {
    SESSION.onclose = null
    SESSION.close(1000)
  }
  SESSION = new WebSocket(url)
  SESSION.onopen = () => handleOpen()
  SESSION.onmessage = message => {
    try {
      handleMessage(JSON.parse(message.data))
    } catch (error) {
      console.log(error)
    }
  }
  SESSION.onerror = error => console.log(error)
  SESSION.onclose = () => reconnect(url)
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
    sendSocket(JSON.stringify(data))
  }
}

const sendSocket = data => {
  if (SESSION?.readyState === WebSocket.OPEN) {
    SESSION.send(data)
  } else {
    OUTBOX.push(data)
  }
}

const flushOutbox = () => {
  while (OUTBOX.length > 0 && SESSION?.readyState === WebSocket.OPEN) {
    SESSION.send(OUTBOX.shift())
  }
}

// Reconnection
export const reconnect = url => {
  wsConnected = false
  updateConnected()
  console.log("reconnecting to api")
  if (reconnectTimer) return
  reconnectTimer = setTimeout(() => {
    reconnectTimer = undefined
    connect(url)
  }, 1000)
}

// Handle connection
export const handleOpen = () => {
  wsConnected = true
  updateConnected()
  // Verify session
  verify()
  flushOutbox()
  connectEvents()
}

// Message Handler
export const handleMessage = data => {
  // Log the activity response and remove 
  // it from pending
  if (data.type === "activity") {
    console.log("activity")
    handleActivity(data)
    firstLoad.set(false)
  } else if (data.type == 'login-failed') {
    loginError.set(data.message);
    setTimeout(() => {
      loginError.set('');
    }, 2000);
    firstLoad.set(false)
  } else {
    handleStreamMessage(data)
  }
}

export const handleStreamMessage = data => {
  console.log("event alive")
  if (data.type === "c2c") {
    if (Array.isArray(data.ssids)) {
      ssids.set(data.ssids)
      isC2CMode.set(true)
    }
  } else if (data.type == "structure") {
    structure.set(data)
    ssids.set([])
    isC2CMode.set(false)
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

export const connectEvents = async () => {
  if (!EVENTS_URL) return
  let token = await loadSession()
  if (!token) return

  const tokenKey = token.id + ":" + token.token
  if (eventController && !eventController.signal.aborted) {
    if (eventTokenKey === tokenKey && (eventConnected || eventConnecting)) return
    eventController.abort()
  }
  if (eventReconnectTimer) {
    clearTimeout(eventReconnectTimer)
    eventReconnectTimer = undefined
  }
  eventTokenKey = tokenKey
  eventController = new AbortController()
  streamEvents(token, eventController)
}

const streamEvents = async (token, controller) => {
  eventConnecting = true
  try {
    const response = await fetch(EVENTS_URL, {
      method: 'POST',
      headers: {
        'Accept': 'text/event-stream',
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ token }),
      cache: 'no-store',
      signal: controller.signal
    })
    if (!response.ok || !response.body) {
      throw new Error(`event stream failed: ${response.status}`)
    }
    eventConnected = true
    eventConnecting = false
    eventRetry = 1000
    updateConnected()
    await readEventStream(response.body)
  } catch (error) {
    if (!controller.signal.aborted) {
      console.log(error)
    }
  } finally {
    if (eventController === controller) {
      eventConnecting = false
      if (!controller.signal.aborted) {
        eventConnected = false
        updateConnected()
        scheduleEventReconnect()
      }
    }
  }
}

const scheduleEventReconnect = () => {
  if (eventReconnectTimer) return
  const retry = eventRetry
  eventRetry = Math.min(eventRetry * 2, maxEventRetry)
  eventReconnectTimer = setTimeout(() => {
    eventReconnectTimer = undefined
    connectEvents()
  }, retry)
}

const readEventStream = async body => {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  for (;;) {
    const { value, done } = await reader.read()
    if (done) return
    buffer += decoder.decode(value, { stream: true })
    let boundary = buffer.indexOf('\n\n')
    while (boundary >= 0) {
      const raw = buffer.slice(0, boundary)
      buffer = buffer.slice(boundary + 2)
      dispatchEventFrame(raw)
      boundary = buffer.indexOf('\n\n')
    }
  }
}

const dispatchEventFrame = raw => {
  if (!raw.trim() || raw.trim().startsWith(':')) return
  const data = []
  for (const line of raw.split(/\r?\n/)) {
    if (!line || line.startsWith(':')) continue
    const separator = line.indexOf(':')
    const field = separator >= 0 ? line.slice(0, separator) : line
    let value = separator >= 0 ? line.slice(separator + 1) : ''
    if (value.startsWith(' ')) value = value.slice(1)
    if (field === 'data') data.push(value)
    if (field === 'retry') {
      const retry = Number.parseInt(value, 10)
      if (!Number.isNaN(retry)) eventRetry = retry
    }
  }
  if (data.length < 1) return
  try {
    handleStreamMessage(JSON.parse(data.join('\n')))
  } catch (error) {
    console.log(error)
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
      saveSession(data.token).then(connectEvents)
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

export const startramBackupPassword = password => {
  let payload = {
    "type":"startram",
    "action":"set-backup-password",
    "password":password
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

export const checkUpdates = () => {
  let payload = {
    "type":"system",
    "action":"update",
    "update":"check"
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
//  Hermes
//

export const hermesInstall = config => {
  send({
    "type":"hermes",
    "action":"install",
    ...config
  })
}

export const hermesUpdate = config => {
  send({
    "type":"hermes",
    "action":"update",
    ...config
  })
}

export const hermesToggle = config => {
  send({
    "type":"hermes",
    "action":"toggle",
    ...config
  })
}

export const hermesSave = config => {
  send({
    "type":"hermes",
    "action":"save",
    ...config
  })
}

export const hermesRestart = () => {
  send({
    "type":"hermes",
    "action":"restart"
  })
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

export const bootShip = (patp,key,keyType,remote,selectedDrive,command="") => {
  let payload = {
    "type":"new_ship",
    "action":"boot",
    "patp":patp,
    "keyType":keyType,
    "key":key,
    "remote":remote,
    "selectedDrive":selectedDrive,
    "command":command
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

//
// Pier backups and restoration
//

export const toggleBackups = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-backup",
    "patp":patp
  }
  send(payload)
}

export const toggleStartramBackups = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-startram-backup",
    "patp":patp
  }
  send(payload)
}

export const scheduleLocalBackup = (patp, backupTime) => {
  let payload = {
    "type":"urbit",
    "action":"schedule-local-backup",
    "patp":patp,
    "backupTime":backupTime 
  }
  send(payload)
}

export const localBackup = patp => {
  let payload = {
    "type":"urbit",
    "action":"local-backup",
    "patp":patp
  }
  send(payload)
}

export const startramBackup = patp => {
  let payload = {
    "type":"urbit",
    "action":"startram-backup",
    "patp":patp
  }
  send(payload)
}

export const restoreTlonBackup = (patp, remote, timestamp, md5, bakType) => {
  let payload = {
    "type": "urbit",
    "action": "restore-tlon-backup",
    "patp": patp,
    "remote": remote,
    "timestamp": timestamp,
    "md5": md5,
    "bakType": bakType
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

export const restoreBackup = (patp, backupFile) => {
  let payload = {
    "type":"urbit",
    "action":"restore-backup",
    "patp":patp,
    "file": "this must fail"
  }
  send(payload)
}

/***/
export const toggleBootStatus = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-boot-status",
    "patp":patp
  }
  send(payload)
}

export const toggleAutoReboot = patp => {
  let payload = {
    "type":"urbit",
    "action":"toggle-auto-reboot",
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

export const setRustFSDomain = (patp, domain) => {
  setMinIODomain(patp, domain)
}

export const removeRustFSDomain = patp => {
  let payload = {
    "type":"urbit",
    "action":"remove-minio-domain",
    "patp":patp,
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

export const toggleRustFSLink = patp => {
  toggleMinIOLink(patp)
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

export const setUrbitSnapTime = (patp, value) => {
  let payload = {
    "type":"urbit",
    "action":"snaptime",
    "patp":patp,
    "value": value
  }
  send(payload)
}

export const setUrbitExtraArgs = (patp, extraArgs) => {
  let payload = {
    "type":"urbit",
    "action":"extra-args",
    "patp":patp,
    "extraArgs": extraArgs
  }
  send(payload)
}

export const setVereTag = (patp, vereTag) => {
  let payload = {
    "type":"urbit",
    "action":"vere-tag",
    "patp":patp,
    "vereTag": vereTag
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

export const submitReport = (contact,description,ships,cpuProfile) => {
  let payload = {
    "type":"support",
    "action":"bug-report",
    "contact":contact,
    "description":description,
    "ships":ships,
    "cpu_profile":cpuProfile
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

export const devBackupTlon = () => {
  let payload = {
    "type": "dev",
    "action": "backup-tlon",
  }
  send(payload)
}

export const devRemoteBackupTlon = () => {
  let payload = {
    "type": "dev",
    "action": "remote-backup-tlon",
  }
  send(payload)
}

export const devRestoreTlon = (patp, remote) => {
  let payload = {
    "type": "dev",
    "action": "restore-tlon",
    "patp": patp,
    "remote": remote
  }
  send(payload)
}
