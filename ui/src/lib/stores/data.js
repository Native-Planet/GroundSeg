import { writable } from 'svelte/store'
export const structure = writable({})
export const firstLoad = writable(true)
export const connected = writable(false)
  // set to true when running with vite.config.urbit.js
export const URBIT_MODE = writable(true)
