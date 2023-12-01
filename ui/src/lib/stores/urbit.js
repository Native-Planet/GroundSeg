import Urbit from '@urbit/http-api';
import { writable } from 'svelte/store';
import { connected, structure, firstLoad } from './data.js'

// urbit object
const urbit = new Urbit("")

// broadcast json string
export const broadcast = writable("")

// subscribe to path
export const subscribe = ship => {
  urbit.ship = ship
  urbit.onOpen =  ()=> console.log("onOpen opened")
  urbit.onRetry = ()=> console.log("onRetry called")
  urbit.onError = e => console.error("onError: "+e)
  urbit.subscribe({
    app: "groundseg",
    path: "/broadcast",
    event: handleEvent,
    quit: handleQuit,
    err: handleErr
  })
}
const handleEvent = event => {
  console.log(typeof event.cord)
  if (typeof event.cord === 'string') {
    let broadcast
    try {
     broadcast = JSON.parse(event.cord)
    } catch (error) {
      console.error("Failed to parse: ", error)
      return
    }
    handleBroadcast(broadcast)
  }
}
const handleQuit = () => {
 console.error("quit called") 
}
const handleErr = () => {
  console.error("error called")
}

const handleBroadcast = broadcast => {
  console.log(broadcast)
  if (broadcast.type == "init") {
    console.log("Sub initiated")
    connected.set(true)
  } else if (broadcast.type == "structure") {
    structure.set(broadcast)
    firstLoad.set(false)
  }
}
