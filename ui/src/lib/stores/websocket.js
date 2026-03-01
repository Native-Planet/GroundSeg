import { get, writable } from 'svelte/store'
import { loadSession, saveSession, generateRandom } from './gs-crypto'
import { URBIT_MODE, connected, structure, firstLoad } from './data.js'
import { sendPoke } from './urbit.js'
import { createWebsocketCommandDomains } from './websocket-commands.js'
import { createRateLimitedJsonParser } from '../runtime/transport/json.js'
import { createWebSocketClient } from '../runtime/transport/websocket-client.js'

export const ready = writable(false)
export const logs = writable({})
export const wsPort = writable('3000')
export const isC2CMode = writable(false)
export const ssids = writable([])
export const loginError = writable('')
export const transportErrors = writable([])

const ACK_TIMEOUT_MS = 15000

let pendingRequests = new Map()
let socketClient

const pushTransportError = (kind, detail, metadata = {}) => {
  const normalizedDetail =
    detail instanceof Error ? detail.message : typeof detail === 'string' ? detail : JSON.stringify(detail)

  transportErrors.update(current => {
    const next = [
      ...current,
      {
        at: new Date().toISOString(),
        kind,
        detail: normalizedDetail,
        metadata
      }
    ]
    return next.slice(-100)
  })
}

export const safeParseSocketMessage = createRateLimitedJsonParser({
  label: 'websocket',
  onError: (error, context) => {
    pushTransportError('parse', error, { raw_preview: context.rawPreview })
  }
})

export const mergeContainerLogLine = (current, containerID, containerLine) => {
  const existing = current[containerID]
  const nextValue = existing ? `${existing}\n${containerLine}` : containerLine
  return {
    ...current,
    [containerID]: nextValue
  }
}

const resolvePendingRequest = (id, resolution) => {
  const entry = pendingRequests.get(id)
  if (!entry) {
    return
  }

  clearTimeout(entry.timeout)
  pendingRequests.delete(id)
  entry.resolve(resolution)
}

const flushPendingRequests = (reason, metadata = {}) => {
  for (const [id, entry] of pendingRequests.entries()) {
    clearTimeout(entry.timeout)
    entry.resolve({ ok: false, id, error: reason, ...metadata })
  }
  pendingRequests = new Map()
}

const trackPendingRequest = (id, payload) => {
  return new Promise(resolve => {
    const timeout = setTimeout(() => {
      pendingRequests.delete(id)
      resolve({
        ok: false,
        id,
        error: 'timeout',
        payloadType: payload?.type || 'unknown'
      })
    }, ACK_TIMEOUT_MS)

    pendingRequests.set(id, {
      resolve,
      timeout,
      payloadType: payload?.type || 'unknown'
    })
  })
}

// Initialize connection
export const connect = async url => {
  if (socketClient) {
    socketClient.disconnect(1000)
  }

  socketClient = createWebSocketClient({
    url,
    onOpen: () => handleOpen(),
    onMessage: message => {
      const parsed = safeParseSocketMessage(message.data)
      if (parsed) {
        handleMessage(parsed)
      }
    },
    onError: error => {
      pushTransportError('socket_error', error)
      console.error(error)
    },
    onClose: () => reconnect(url)
  })

  socketClient.connect()
}

// WebSocket send wrapper
export const sendCommand = async payload => {
  if (get(URBIT_MODE)) {
    sendPoke(payload)
    return { ok: true, mode: 'urbit' }
  }

  if (!socketClient) {
    pushTransportError('send_not_open', 'WebSocket client is missing', {
      ready_state: 'missing_client',
      payload_type: payload?.type || 'unknown'
    })
    return {
      ok: false,
      error: 'missing_client'
    }
  }

  if (!socketClient.isOpen()) {
    pushTransportError('send_not_open', 'WebSocket is not open', {
      ready_state: socketClient.getReadyState(),
      payload_type: payload?.type || 'unknown'
    })
    return {
      ok: false,
      error: 'not_open',
      readyState: socketClient.getReadyState()
    }
  }

  const id = await generateRandom(16)
  const token = await loadSession()
  const pendingAck = trackPendingRequest(id, payload)
  const data = {
    id,
    payload
  }

  if (token) {
    data.token = token
  }

  const serialized = JSON.stringify(data)
  const sendResult = socketClient.send(serialized)

  if (!sendResult.ok) {
    resolvePendingRequest(id, {
      ok: false,
      id,
      error: sendResult.reason,
      readyState: sendResult.readyState
    })

    pushTransportError(sendResult.reason, sendResult.error || sendResult.reason, {
      payload_type: payload?.type || 'unknown',
      ready_state: sendResult.readyState
    })

    return pendingAck
  }

  console.log(`${id}:${payload.type} sent`)
  return pendingAck
}

// Reconnection
export const reconnect = url => {
  connected.set(false)
  flushPendingRequests('disconnected')
  console.log('reconnecting to api')
  setTimeout(() => connect(url), 1000)
}

// Handle connection
export const handleOpen = () => {
  connected.set(true)
  verify()
}

// Message Handler
export const handleMessage = data => {
  if (data.type === 'c2c') {
    if (Array.isArray(data.ssids)) {
      ssids.set(data.ssids)
      isC2CMode.set(true)
    }
  } else if (data.type === 'activity') {
    handleActivity(data)
    ssids.set([])
    isC2CMode.set(false)
  } else if (data.type === 'structure') {
    structure.set(data)
    ssids.set([])
    isC2CMode.set(false)
  } else if (data.type === 'login-failed') {
    loginError.set(data.message)
    setTimeout(() => {
      loginError.set('')
    }, 2000)
  } else if (Object.prototype.hasOwnProperty.call(data, 'log')) {
    logs.update(current => mergeContainerLogLine(current, data.log.container_id, data.log.line))
  }
  firstLoad.set(false)
}

// Activity Handler
export const handleActivity = data => {
  let responseMessage = `${data.id} ${data.response}`

  if (data.response === 'nack') {
    responseMessage = `${responseMessage}: ${data.error}`
  }

  if (data.error === 'NOT_READY') {
    ready.set(false)
  } else {
    ready.set(true)
  }

  if (Object.prototype.hasOwnProperty.call(data, 'token')) {
    saveSession(data.token)
  }

  console.log(responseMessage)

  resolvePendingRequest(data.id, {
    ok: data.response !== 'nack',
    id: data.id,
    response: data.response,
    error: data.error || null,
    data
  })
}

export const websocketCommandDomains = createWebsocketCommandDomains({
  sendCommand,
  clearLoginError: () => loginError.set('')
})

const commandApi = {
  ...websocketCommandDomains.auth,
  ...websocketCommandDomains.setup,
  ...websocketCommandDomains.system,
  ...websocketCommandDomains.startram,
  ...websocketCommandDomains.ship,
  ...websocketCommandDomains.urbit,
  ...websocketCommandDomains.support,
  ...websocketCommandDomains.logs,
  ...websocketCommandDomains.network,
  ...websocketCommandDomains.penpai,
  ...websocketCommandDomains.dev
}

export const verify = commandApi.verify
export const login = commandApi.login
export const logout = commandApi.logout
export const logoutAll = commandApi.logoutAll
export const modifyPassword = commandApi.modifyPassword
export const beginSetup = commandApi.beginSetup
export const setupPassword = commandApi.setupPassword
export const setupSkip = commandApi.setupSkip
export const setupStarTram = commandApi.setupStarTram
export const startramBackupPassword = commandApi.startramBackupPassword
export const restartGroundSeg = commandApi.restartGroundSeg
export const restartDevice = commandApi.restartDevice
export const shutdownDevice = commandApi.shutdownDevice
export const updateLinux = commandApi.updateLinux
export const setSwap = commandApi.setSwap
export const toggleWifi = commandApi.toggleWifi
export const connectWifi = commandApi.connectWifi
export const startramGetRegions = commandApi.startramGetRegions
export const startramGetServices = commandApi.startramGetServices
export const startramRegister = commandApi.startramRegister
export const startramToggle = commandApi.startramToggle
export const startramRestart = commandApi.startramRestart
export const startramEndpoint = commandApi.startramEndpoint
export const startramCancel = commandApi.startramCancel
export const setAllStartramReminder = commandApi.setAllStartramReminder
export const openUploadEndpoint = commandApi.openUploadEndpoint
export const modifyUploadEndpoint = commandApi.modifyUploadEndpoint
export const closeUploadEndpoint = commandApi.closeUploadEndpoint
export const resetImportShip = commandApi.resetImportShip
export const cancelNewShip = commandApi.cancelNewShip
export const bootShip = commandApi.bootShip
export const resetNewShip = commandApi.resetNewShip
export const toggleBackups = commandApi.toggleBackups
export const toggleStartramBackups = commandApi.toggleStartramBackups
export const scheduleLocalBackup = commandApi.scheduleLocalBackup
export const localBackup = commandApi.localBackup
export const startramBackup = commandApi.startramBackup
export const restoreTlonBackup = commandApi.restoreTlonBackup
export const registerServiceAgain = commandApi.registerServiceAgain
export const restoreBackup = commandApi.restoreBackup
export const toggleBootStatus = commandApi.toggleBootStatus
export const toggleAutoReboot = commandApi.toggleAutoReboot
export const toggleDevMode = commandApi.toggleDevMode
export const setMinIODomain = commandApi.setMinIODomain
export const setUrbitDomain = commandApi.setUrbitDomain
export const setNewMaxPierSize = commandApi.setNewMaxPierSize
export const toggleAutoBoot = commandApi.toggleAutoBoot
export const toggleNetwork = commandApi.toggleNetwork
export const toggleMinIOLink = commandApi.toggleMinIOLink
export const toggleUrbitPower = commandApi.toggleUrbitPower
export const deleteUrbitShip = commandApi.deleteUrbitShip
export const exportUrbitShip = commandApi.exportUrbitShip
export const exportUrbitBucket = commandApi.exportUrbitBucket
export const rebuildContainer = commandApi.rebuildContainer
export const setUrbitLoom = commandApi.setUrbitLoom
export const setUrbitSnapTime = commandApi.setUrbitSnapTime
export const setPackSchedule = commandApi.setPackSchedule
export const pausePackSchedule = commandApi.pausePackSchedule
export const toggleUrbitAlias = commandApi.toggleUrbitAlias
export const marsPack = commandApi.marsPack
export const urthPackMeld = commandApi.urthPackMeld
export const urbitChop = commandApi.urbitChop
export const urbitRollChop = commandApi.urbitRollChop
export const toggleChopAfterVereUpdate = commandApi.toggleChopAfterVereUpdate
export const setStartramReminder = commandApi.setStartramReminder
export const submitReport = commandApi.submitReport
export const toggleLog = commandApi.toggleLog
export const submitNetwork = commandApi.submitNetwork
export const toggleExperimentalPenpai = commandApi.toggleExperimentalPenpai
export const togglePenpai = commandApi.togglePenpai
export const setPenpaiModel = commandApi.setPenpaiModel
export const setPenpaiCores = commandApi.setPenpaiCores
export const removePenpai = commandApi.removePenpai
export const installPenpaiCompanion = commandApi.installPenpaiCompanion
export const uninstallPenpaiCompanion = commandApi.uninstallPenpaiCompanion
export const installGallseg = commandApi.installGallseg
export const deleteStartramService = commandApi.deleteStartramService
export const uninstallGallseg = commandApi.uninstallGallseg
export const resetSetup = commandApi.resetSetup
export const printMounts = commandApi.printMounts
export const devStartramReminder = commandApi.devStartramReminder
export const devStartramReminderToggle = commandApi.devStartramReminderToggle
export const devBackupTlon = commandApi.devBackupTlon
export const devRemoteBackupTlon = commandApi.devRemoteBackupTlon
export const devRestoreTlon = commandApi.devRestoreTlon
