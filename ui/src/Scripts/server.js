import { writable } from 'svelte/store';

// config
export const url = "http://localhost:5000"

// stores
export const piers = writable(null)
export const scrollDown = writable(true)
