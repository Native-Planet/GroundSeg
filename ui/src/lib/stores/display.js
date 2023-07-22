import { writable } from 'svelte/store'
  
export const wide = writable(true)
export const limbo = writable(false)
export const version = writable("v2.0.0")

export const setLimbo = b => {
  limbo.set(b)
}
