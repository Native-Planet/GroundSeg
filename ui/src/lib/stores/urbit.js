import { writable } from 'svelte/store'
import { routeGroundsegBroadcast, parseBroadcastCord } from '../runtime/handlers/groundseg-events.js'
import { createUrbitSseClient } from '../runtime/transport/urbit-sse-client.js'

let realtimeClient = createUrbitSseClient('')

// broadcast json string
export const broadcast = writable('')

// login string
export const gallsegLoginInfo = writable({})

export const setRealtimeClient = client => {
  realtimeClient = client
}

export const sendPoke = async payload => {
  const result = await realtimeClient.sendAction(payload)
  if (result.ok) {
    console.log('poke succeeded')
  } else {
    console.log(result.error)
  }
  return result
}

// subscribe to path
export const subscribe = ship => {
  realtimeClient.subscribe({
    ship,
    onOpen: () => console.log('onOpen opened'),
    onRetry: () => console.log('onRetry called'),
    onError: error => console.error(`onError: ${error}`),
    onEvent: handleEvent,
    onQuit: handleQuit,
    onSubscriptionError: handleErr
  })
}

const handleEvent = event => {
  if (typeof event.cord !== 'string') {
    return
  }

  const parsedBroadcast = parseBroadcastCord(event.cord)
  if (!parsedBroadcast) {
    return
  }

  handleBroadcast(parsedBroadcast)
}

const handleQuit = () => {
  console.error('quit called')
}

const handleErr = () => {
  console.error('error called')
}

const handleBroadcast = parsedBroadcast => {
  console.log(parsedBroadcast)

  routeGroundsegBroadcast({
    broadcast: parsedBroadcast,
    onLoginActivity: loginActivity => {
      gallsegLoginInfo.set(loginActivity)
    }
  })
}

export { parseBroadcastCord }

export const sendHeartbeat = async () => {
  const result = await realtimeClient.sendHeartbeat()
  if (result.ok) {
    console.log('poke succeeded')
  } else {
    console.log(result.error)
  }
  return result
}
