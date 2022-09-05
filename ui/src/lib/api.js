import { writable } from 'svelte/store';
import cfg from '/src/config.json' assert {type: 'json'}

// api url
export const api = cfg.url + ":" + cfg.port

// stores
export const piers = writable(null)
export const scrollDown = writable(true)
