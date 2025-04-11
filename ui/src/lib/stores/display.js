import { writable } from 'svelte/store'
  
export const wide = writable(true)
export const version = writable(import.meta.env.GS_VERSION || "dev")
