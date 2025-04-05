import { writable } from 'svelte/store'
  
export const wide = writable(true)
export const version = writable(import.meta.env.VITE_APP_VERSION || "dev")
