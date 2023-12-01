import { writable } from 'svelte/store'
export const structure = writable({})
export const firstLoad = writable(true)
export const connected = writable(false)
